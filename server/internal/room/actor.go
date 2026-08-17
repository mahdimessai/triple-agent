package room

import (
	"errors"
	"sync"
	"time"

	"tripleagent/server/internal/domain"
)

type Room struct {
	// commands is the actor mailbox; all room state changes pass through it.
	commands chan roomMessage
	// done closes when the room stops accepting work or its actor exits.
	done chan struct{}
	// closeOnce makes room shutdown safe when expiry and explicit removal race.
	closeOnce sync.Once
	// lifetime is the normal idle/lobby lifetime before the room expires.
	lifetime time.Duration
	// endedAfter is the shorter lifetime retained after a match reaches its end.
	endedAfter time.Duration
	// onExpire removes the room from its manager when the lifetime timer fires.
	onExpire func()
	// cleanup releases whatever outlives the actor - the join code and the
	// credentials - and runs however the room ends.
	cleanup func()
}

func newRoom(state domain.GameState, lifetime, endedAfter time.Duration, onExpire func()) *Room {
	room := &Room{
		commands:   make(chan roomMessage),
		done:       make(chan struct{}),
		lifetime:   lifetime,
		endedAfter: endedAfter,
		onExpire:   onExpire,
	}
	go room.loop(state)
	return room
}

func (r *Room) Close() {
	r.closeOnce.Do(func() { close(r.done) })
}

func (r *Room) loop(state domain.GameState) {
	sessions := make(map[string]roomSession)
	dedupe := make(map[string]dedupeEntry)
	dedupeOrder := make([]string, 0, maxDedupeEntries)

	timers := newRoomTimers(r)
	timers.resetDiscussion(state)

	defer func() {
		timers.stopDiscussion()
		timers.stopExpiry()
		for _, session := range sessions {
			if session.close != nil {
				session.close()
			}
		}
	}()

	for {
		select {
		case <-r.done:
			return
		case <-timers.expiryC:
			if r.onExpire != nil {
				r.onExpire()
			}
			r.Close()
			return
		case <-timers.discussionC:
			if (state.Phase == domain.PhaseDiscussion || state.Phase == domain.PhaseOperationInterlude) && state.DiscussionDeadline != nil && !time.Now().UTC().Before(*state.DiscussionDeadline) {
				kind := domain.CommandAdvanceDiscussion
				if state.Phase == domain.PhaseOperationInterlude {
					kind = domain.CommandAdvanceInterlude
				}
				command := domain.Command{
					ActorID:         state.HostID,
					ExpectedVersion: state.Version,
					Kind:            kind,
				}
				transition, err := domain.Apply(state, command, time.Now().UTC())
				if err == nil {
					state = transition.State
					r.broadcast(&state, sessions)
				}
			}
			timers.resetDiscussion(state)
		case message := <-r.commands:
			switch message.kind {
			case "add_player":
				transition, err := domain.ApplyJoin(state, message.playerID, message.name, time.Now().UTC())
				if err == nil && message.commit != nil {
					err = message.commit(transition.State)
				}
				if message.reply != nil {
					message.reply <- roomResponse{err: err}
				}
				if err != nil {
					continue
				}
				// A replayed join finds its seat already taken and changes
				// nothing, so there is no new state to publish.
				if transition.Changed {
					state = transition.State
					r.broadcast(&state, sessions)
				}
				timers.resetExpiry(r.lifetime)

			case "attach":
				r.handleAttach(&state, sessions, message)

			case "detach":
				r.handleDetach(&state, sessions, message)

			case "remove_player":
				transition, err := domain.ApplyLeave(state, message.playerID, time.Now().UTC())
				if err == nil && message.commit != nil {
					err = message.commit(transition.State)
				}
				if err != nil {
					message.reply <- roomResponse{err: err}
					continue
				}
				state = transition.State
				if session, exists := sessions[message.playerID]; exists {
					delete(sessions, message.playerID)
					if session.close != nil {
						session.close()
					}
				}
				r.broadcast(&state, sessions)
				message.reply <- roomResponse{reply: roomReply{state: state}}
				timers.resetExpiry(r.lifetime)

			case "snapshot":
				if _, ok := state.Players[message.playerID]; !ok {
					message.reply <- roomResponse{err: domain.ErrPlayerNotInRoom}
					continue
				}
				message.reply <- roomResponse{reply: roomReply{projection: domain.Project(state, message.playerID)}}

			case "command":
				command := *message.command
				if message.sessionID != "" {
					current, exists := sessions[command.ActorID]
					if !exists || current.id != message.sessionID {
						message.reply <- roomResponse{err: errors.New("session is no longer active")}
						continue
					}
				}
				if command.RequestID != "" {
					if entry, exists := dedupe[command.RequestID]; exists {
						if entry.actorID != command.ActorID {
							message.reply <- roomResponse{err: errors.New("request id already used by another player")}
						} else {
							message.reply <- roomResponse{
								reply: roomReply{projection: domain.Project(state, command.ActorID), replayed: entry.err == nil},
								err:   entry.err,
							}
						}
						continue
					}
				}
				transition, err := domain.Apply(state, command, time.Now().UTC())
				if err != nil {
					if command.RequestID != "" {
						dedupe, dedupeOrder = rememberDedupe(dedupe, dedupeOrder, command.RequestID, dedupeEntry{actorID: command.ActorID, err: err})
					}
					message.reply <- roomResponse{err: err}
					continue
				}
				if !transition.Changed {
					projection := domain.Project(state, command.ActorID)
					if command.RequestID != "" {
						dedupe, dedupeOrder = rememberDedupe(dedupe, dedupeOrder, command.RequestID, dedupeEntry{actorID: command.ActorID, projection: projection})
					}
					message.reply <- roomResponse{reply: roomReply{projection: projection}}
					continue
				}

				state = transition.State
				if command.Kind == domain.CommandRematch {
					dedupe = make(map[string]dedupeEntry)
					dedupeOrder = dedupeOrder[:0]
				}
				for playerID, session := range sessions {
					if _, exists := state.Players[playerID]; !exists {
						delete(sessions, playerID)
						if session.close != nil {
							session.close()
						}
					}
				}
				projection := domain.Project(state, command.ActorID)
				if command.RequestID != "" {
					dedupe, dedupeOrder = rememberDedupe(dedupe, dedupeOrder, command.RequestID, dedupeEntry{actorID: command.ActorID, projection: projection})
				}
				r.broadcast(&state, sessions)
				message.reply <- roomResponse{reply: roomReply{projection: projection}}
				timers.resetDiscussion(state)
				if state.Phase == domain.PhaseEnd {
					timers.resetExpiry(r.endedAfter)
				} else if command.Kind == domain.CommandRematch || command.Kind == domain.CommandStartMatch {
					timers.resetExpiry(r.lifetime)
				}
			}
		}
	}
}
