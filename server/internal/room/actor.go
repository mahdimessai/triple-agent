package room

import (
	"errors"
	"sync"
	"time"

	"tripleagent/server/internal/domain"
)

type Room struct {
	// commands is the actor mailbox; all room runtime changes pass through it.
	commands chan roomMessage
	// done closes when the room stops accepting work.
	done chan struct{}
	// closeOnce makes room shutdown safe across competing retirement paths.
	closeOnce sync.Once
	// lifetime is the normal idle/lobby lifetime before the room expires.
	lifetime time.Duration
	// endedAfter is the shorter lifetime retained after a match reaches its end.
	endedAfter time.Duration
	// retire removes this exact room instance from its manager and runs cleanup.
	retire func()
	// cleanup is retained so explicit manager removal uses the same lifecycle hook.
	cleanup func()
}

type runtimeState struct {
	game        domain.GameState
	sessions    map[string]roomSession
	credentials map[string]string
	dedupe      map[string]dedupeEntry
	dedupeOrder []string
}

// newRoom keeps tests and low-level callers simple when reconnect credentials
// are irrelevant.
func newRoom(state domain.GameState, lifetime, endedAfter time.Duration, retire func()) *Room {
	return newRoomWithCredentials(state, nil, lifetime, endedAfter, retire)
}

func newRoomWithCredentials(state domain.GameState, credentials map[string]string, lifetime, endedAfter time.Duration, retire func()) *Room {
	room := &Room{
		commands:   make(chan roomMessage),
		done:       make(chan struct{}),
		lifetime:   lifetime,
		endedAfter: endedAfter,
		retire:     retire,
	}
	go room.loop(state, cloneCredentials(credentials))
	return room
}

func (r *Room) Close() {
	r.closeOnce.Do(func() { close(r.done) })
}

func (r *Room) retireNow() {
	if r.retire != nil {
		r.retire()
	}
	r.Close()
}

func (r *Room) loop(state domain.GameState, credentials map[string]string) {
	runtime := runtimeState{
		game:        state,
		sessions:    make(map[string]roomSession),
		credentials: credentials,
		dedupe:      make(map[string]dedupeEntry),
		dedupeOrder: make([]string, 0, maxDedupeEntries),
	}

	timers := newRoomTimers(r)
	timers.resetDiscussion(runtime.game)

	defer func() {
		timers.stopDiscussion()
		timers.stopExpiry()
		for _, session := range runtime.sessions {
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
			r.retireNow()
			return

		case <-timers.discussionC:
			if (runtime.game.Phase == domain.PhaseDiscussion || runtime.game.Phase == domain.PhaseOperationInterlude) && runtime.game.DiscussionDeadline != nil && !time.Now().UTC().Before(*runtime.game.DiscussionDeadline) {
				kind := domain.CommandAdvanceDiscussion
				if runtime.game.Phase == domain.PhaseOperationInterlude {
					kind = domain.CommandAdvanceInterlude
				}
				command := domain.Command{
					ActorID:         runtime.game.HostID,
					ExpectedVersion: runtime.game.Version,
					Kind:            kind,
				}
				transition, err := domain.Apply(runtime.game, command, time.Now().UTC())
				if err == nil {
					runtime.game = transition.State
					pruneCredentials(runtime.credentials, runtime.game)
					if r.broadcast(&runtime) {
						r.retireNow()
						return
					}
				}
			}
			timers.resetDiscussion(runtime.game)

		case message := <-r.commands:
			switch message.kind {
			case messageAddPlayer:
				transition, err := domain.ApplyJoin(runtime.game, message.playerID, message.name, time.Now().UTC())
				if err != nil {
					message.reply <- roomResponse{err: err}
					continue
				}
				if transition.Changed {
					runtime.game = transition.State
				}
				if message.token != "" {
					runtime.credentials[message.playerID] = message.token
				}
				message.reply <- roomResponse{}
				if transition.Changed && r.broadcast(&runtime) {
					r.retireNow()
					return
				}
				timers.resetExpiry(r.lifetime)

			case messageAuthenticate:
				if _, exists := runtime.game.Players[message.playerID]; !exists || !validCredential(runtime.credentials, message.playerID, message.token) {
					message.reply <- roomResponse{err: ErrInvalidCredential}
					continue
				}
				message.reply <- roomResponse{}

			case messageAttach:
				if r.handleAttach(&runtime, message) {
					r.retireNow()
					return
				}

			case messageDetach:
				if r.handleDetach(&runtime, message) {
					r.retireNow()
					return
				}

			case messageRemovePlayer:
				transition, err := domain.ApplyLeave(runtime.game, message.playerID, time.Now().UTC())
				if err != nil {
					message.reply <- roomResponse{err: err}
					continue
				}
				runtime.game = transition.State
				delete(runtime.credentials, message.playerID)
				if session, exists := runtime.sessions[message.playerID]; exists {
					delete(runtime.sessions, message.playerID)
					if session.close != nil {
						session.close()
					}
				}
				retire := len(runtime.game.PlayerOrder) == 0
				if !retire {
					retire = r.broadcast(&runtime)
				}
				message.reply <- roomResponse{}
				timers.resetExpiry(r.lifetime)
				if retire {
					r.retireNow()
					return
				}

			case messageSnapshot:
				if _, ok := runtime.game.Players[message.playerID]; !ok {
					message.reply <- roomResponse{err: domain.ErrPlayerNotInRoom}
					continue
				}
				message.reply <- roomResponse{reply: roomReply{projection: domain.Project(runtime.game, message.playerID)}}

			case messageCommand:
				command := *message.command
				if message.sessionID != "" {
					current, exists := runtime.sessions[command.ActorID]
					if !exists || current.id != message.sessionID {
						message.reply <- roomResponse{err: errors.New("session is no longer active")}
						continue
					}
				}

				if command.RequestID != "" {
					if entry, exists := runtime.dedupe[command.RequestID]; exists {
						if entry.actorID != command.ActorID {
							message.reply <- roomResponse{err: errors.New("request id already used by another player")}
						} else {
							message.reply <- roomResponse{
								reply: roomReply{projection: domain.Project(runtime.game, command.ActorID), replayed: entry.err == nil},
								err:   entry.err,
							}
						}
						continue
					}
				}

				transition, err := domain.Apply(runtime.game, command, time.Now().UTC())
				if err != nil {
					if command.RequestID != "" {
						runtime.dedupe, runtime.dedupeOrder = rememberDedupe(runtime.dedupe, runtime.dedupeOrder, command.RequestID, dedupeEntry{actorID: command.ActorID, err: err})
					}
					message.reply <- roomResponse{err: err}
					continue
				}

				if !transition.Changed {
					projection := domain.Project(runtime.game, command.ActorID)
					if command.RequestID != "" {
						runtime.dedupe, runtime.dedupeOrder = rememberDedupe(runtime.dedupe, runtime.dedupeOrder, command.RequestID, dedupeEntry{actorID: command.ActorID, projection: projection})
					}
					message.reply <- roomResponse{reply: roomReply{projection: projection}}
					continue
				}

				runtime.game = transition.State
				if command.Kind == domain.CommandRematch {
					runtime.dedupe = make(map[string]dedupeEntry)
					runtime.dedupeOrder = runtime.dedupeOrder[:0]
				}
				pruneCredentials(runtime.credentials, runtime.game)
				for playerID, session := range runtime.sessions {
					if _, exists := runtime.game.Players[playerID]; !exists {
						delete(runtime.sessions, playerID)
						if session.close != nil {
							session.close()
						}
					}
				}

				projection := domain.Project(runtime.game, command.ActorID)
				if command.RequestID != "" {
					runtime.dedupe, runtime.dedupeOrder = rememberDedupe(runtime.dedupe, runtime.dedupeOrder, command.RequestID, dedupeEntry{actorID: command.ActorID, projection: projection})
				}
				retire := r.broadcast(&runtime)
				message.reply <- roomResponse{reply: roomReply{projection: projection}}
				timers.resetDiscussion(runtime.game)
				if runtime.game.Phase == domain.PhaseEnd {
					timers.resetExpiry(r.endedAfter)
				} else if command.Kind == domain.CommandRematch || command.Kind == domain.CommandStartMatch {
					timers.resetExpiry(r.lifetime)
				}
				if retire {
					r.retireNow()
					return
				}
			}
		}
	}
}
