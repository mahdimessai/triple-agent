package room

import (
	"errors"
	"sync"
	"time"

	"tripleagent/server/internal/game"
)

const (
	roomLifetime      = 4 * time.Hour
	endedRoomLifetime = 15 * time.Minute
)

var (
	ErrClosed          = errors.New("room is closed")
	ErrUnauthorized    = errors.New("invalid reconnect token")
	ErrSessionGone     = errors.New("session is no longer active")
	ErrVersionConflict = errors.New("stale room version")
)

type Room struct {
	id       string
	requests chan request
	done     chan struct{}

	closeOnce sync.Once
	onClose   func(*Room)

	lifetime   time.Duration
	endedAfter time.Duration
}

type runtime struct {
	state    game.State
	tokens   map[string]string
	sessions map[string]session
}

type session struct {
	id    string
	send  func(game.Projection) error
	close func()
}

type requestKind uint8

const (
	requestJoin requestKind = iota
	requestLeave
	requestAttach
	requestDetach
	requestCommand
	requestSnapshot
)

type request struct {
	kind requestKind

	playerID string
	name     string
	token    string

	sessionID string
	send      func(game.Projection) error
	close     func()

	expectedVersion uint64
	command         game.Command

	reply chan response
}

type response struct {
	projection game.Projection
	err        error
}

func newRoom(roomID string, state game.State, tokens map[string]string, onClose func(*Room)) *Room {
	return newRoomWithLifetimes(roomID, state, tokens, onClose, roomLifetime, endedRoomLifetime)
}

func newRoomWithLifetimes(roomID string, state game.State, tokens map[string]string, onClose func(*Room), lifetime, endedAfter time.Duration) *Room {
	r := &Room{id: roomID, requests: make(chan request), done: make(chan struct{}), onClose: onClose, lifetime: lifetime, endedAfter: endedAfter}
	go r.loop(runtime{state: state, tokens: cloneTokens(tokens), sessions: make(map[string]session)})
	return r
}

func (r *Room) Close() { r.closeOnce.Do(func() { close(r.done) }) }

func (r *Room) Join(playerID, name, reconnectToken string) error {
	_, err := r.call(request{kind: requestJoin, playerID: playerID, name: name, token: reconnectToken})
	return err
}

func (r *Room) Leave(playerID, reconnectToken string) error {
	_, err := r.call(request{kind: requestLeave, playerID: playerID, token: reconnectToken})
	return err
}

func (r *Room) Attach(playerID, reconnectToken, sessionID string, send func(game.Projection) error, close func()) error {
	_, err := r.call(request{kind: requestAttach, playerID: playerID, token: reconnectToken, sessionID: sessionID, send: send, close: close})
	return err
}

func (r *Room) Detach(playerID, sessionID string) {
	_, _ = r.call(request{kind: requestDetach, playerID: playerID, sessionID: sessionID})
}

func (r *Room) Command(playerID, sessionID string, expectedVersion uint64, command game.Command) error {
	_, err := r.call(request{kind: requestCommand, playerID: playerID, sessionID: sessionID, expectedVersion: expectedVersion, command: command})
	return err
}

func (r *Room) Snapshot(playerID string) (game.Projection, error) {
	result, err := r.call(request{kind: requestSnapshot, playerID: playerID})
	return result.projection, err
}

func (r *Room) call(req request) (response, error) {
	req.reply = make(chan response, 1)
	select {
	case <-r.done:
		return response{}, ErrClosed
	case r.requests <- req:
	}
	select {
	case result := <-req.reply:
		return result, result.err
	case <-r.done:
		select {
		case result := <-req.reply:
			return result, result.err
		default:
			return response{}, ErrClosed
		}
	}
}

func (r *Room) loop(rt runtime) {
	expiryTimer := time.NewTimer(r.lifetime)
	var deadlineTimer *time.Timer
	var deadlineC <-chan time.Time

	resetExpiry := func(duration time.Duration) {
		stopTimer(expiryTimer)
		expiryTimer.Reset(duration)
	}
	resetDeadline := func() {
		if deadlineTimer != nil {
			stopTimer(deadlineTimer)
			deadlineTimer = nil
			deadlineC = nil
		}
		if rt.state.DiscussionDeadline == nil || (rt.state.Phase != game.PhaseDiscussion && rt.state.Phase != game.PhaseOperationInterlude) {
			return
		}
		duration := time.Until(*rt.state.DiscussionDeadline)
		if duration <= 0 {
			duration = time.Nanosecond
		}
		deadlineTimer = time.NewTimer(duration)
		deadlineC = deadlineTimer.C
	}
	resetDeadline()

	defer func() {
		stopTimer(expiryTimer)
		if deadlineTimer != nil {
			stopTimer(deadlineTimer)
		}
		for _, current := range rt.sessions {
			if current.close != nil {
				current.close()
			}
		}
		r.Close()
		if r.onClose != nil {
			r.onClose(r)
		}
	}()

	for {
		select {
		case <-r.done:
			return

		case <-expiryTimer.C:
			return

		case now := <-deadlineC:
			next, err := game.AdvanceDeadline(rt.state, now.UTC())
			if err == nil && next.Version != rt.state.Version {
				rt.state = next
				if r.broadcast(&rt, now.UTC()) {
					return
				}
			}
			resetDeadline()

		case req := <-r.requests:
			now := time.Now().UTC()
			switch req.kind {
			case requestJoin:
				if req.token == "" {
					req.reply <- response{err: ErrUnauthorized}
					continue
				}
				next, err := game.AddPlayer(rt.state, req.playerID, req.name)
				if err == nil {
					rt.state = next
					rt.tokens[req.playerID] = req.token
					if r.broadcast(&rt, now) {
						req.reply <- response{}
						return
					}
					resetExpiry(r.lifetime)
				}
				req.reply <- response{err: err}

			case requestLeave:
				if !authorized(rt.tokens, req.playerID, req.token) {
					req.reply <- response{err: ErrUnauthorized}
					continue
				}
				next, err := game.Leave(rt.state, req.playerID)
				if err != nil {
					req.reply <- response{err: err}
					continue
				}
				rt.state = next
				delete(rt.tokens, req.playerID)
				closeSession(rt.sessions, req.playerID)
				req.reply <- response{}
				if game.Empty(rt.state) {
					return
				}
				if r.broadcast(&rt, now) {
					return
				}
				resetExpiry(r.lifetime)

			case requestAttach:
				if !authorized(rt.tokens, req.playerID, req.token) {
					req.reply <- response{err: ErrUnauthorized}
					continue
				}
				if !game.HasPlayer(rt.state, req.playerID) {
					req.reply <- response{err: game.ErrPlayerNotInRoom}
					continue
				}
				closeSession(rt.sessions, req.playerID)
				next, err := game.Connect(rt.state, req.playerID)
				if err != nil {
					req.reply <- response{err: err}
					continue
				}
				rt.state = next
				rt.sessions[req.playerID] = session{id: req.sessionID, send: req.send, close: req.close}
				projection := game.Project(r.id, rt.state, req.playerID)
				if req.send == nil {
					err = errors.New("session sender is required")
				} else {
					err = req.send(projection)
				}
				if err != nil {
					delete(rt.sessions, req.playerID)
					disconnected, disconnectErr := game.Disconnect(rt.state, req.playerID, now)
					if disconnectErr == nil {
						rt.state = disconnected
						if !game.HasPlayer(rt.state, req.playerID) {
							delete(rt.tokens, req.playerID)
						}
					}
					req.reply <- response{err: err}
					if game.Empty(rt.state) {
						return
					}
					continue
				}
				req.reply <- response{}
				if r.broadcastExcept(&rt, req.playerID, now) {
					return
				}
				resetDeadline()

			case requestDetach:
				current, exists := rt.sessions[req.playerID]
				if !exists || current.id != req.sessionID {
					req.reply <- response{}
					continue
				}
				delete(rt.sessions, req.playerID)
				next, err := game.Disconnect(rt.state, req.playerID, now)
				if err == nil {
					rt.state = next
					if !game.HasPlayer(rt.state, req.playerID) {
						delete(rt.tokens, req.playerID)
					}
				}
				req.reply <- response{err: err}
				if err != nil {
					continue
				}
				if game.Empty(rt.state) {
					return
				}
				if r.broadcast(&rt, now) {
					return
				}
				resetDeadline()

			case requestCommand:
				current, exists := rt.sessions[req.playerID]
				if !exists || current.id != req.sessionID {
					req.reply <- response{err: ErrSessionGone}
					continue
				}
				if req.expectedVersion != rt.state.Version {
					req.reply <- response{err: ErrVersionConflict}
					continue
				}
				before := rt.state
				next, err := game.Apply(before, req.playerID, req.command, now)
				if err != nil {
					req.reply <- response{err: err}
					continue
				}
				if next.Version == before.Version {
					req.reply <- response{}
					continue
				}
				rt.state = next
				cleanupRemovedPlayers(&rt)
				req.reply <- response{}
				if game.Empty(rt.state) {
					return
				}
				if r.broadcast(&rt, now) {
					return
				}
				resetDeadline()
				switch rt.state.Phase {
				case game.PhaseEnd:
					resetExpiry(r.endedAfter)
				default:
					if req.command.Kind == game.CommandRematch || req.command.Kind == game.CommandStartMatch {
						resetExpiry(r.lifetime)
					}
				}

			case requestSnapshot:
				if !game.HasPlayer(rt.state, req.playerID) {
					req.reply <- response{err: game.ErrPlayerNotInRoom}
					continue
				}
				req.reply <- response{projection: game.Project(r.id, rt.state, req.playerID)}
			}
		}
	}
}

func (r *Room) broadcast(rt *runtime, now time.Time) bool {
	return r.broadcastExcept(rt, "", now)
}

func (r *Room) broadcastExcept(rt *runtime, excluded string, now time.Time) bool {
	failed := r.deliver(rt, excluded)
	if len(failed) == 0 {
		return game.Empty(rt.state)
	}
	changed := r.disconnectFailed(rt, failed, now)
	if game.Empty(rt.state) {
		return true
	}
	if !changed {
		return false
	}
	failed = r.deliver(rt, "")
	if len(failed) > 0 {
		r.disconnectFailed(rt, failed, now)
	}
	return game.Empty(rt.state)
}

func (r *Room) deliver(rt *runtime, excluded string) []string {
	failed := make([]string, 0)
	public := game.PublicProjectionFor(r.id, rt.state)
	for playerID, current := range rt.sessions {
		if playerID == excluded {
			continue
		}
		if current.send == nil || current.send(game.ProjectWithPublic(rt.state, playerID, public)) != nil {
			delete(rt.sessions, playerID)
			if current.close != nil {
				current.close()
			}
			failed = append(failed, playerID)
		}
	}
	return failed
}

func (r *Room) disconnectFailed(rt *runtime, failed []string, now time.Time) bool {
	changed := false
	for _, playerID := range failed {
		before := rt.state.Version
		next, err := game.Disconnect(rt.state, playerID, now)
		if err != nil {
			continue
		}
		rt.state = next
		if next.Version != before {
			changed = true
		}
		if !game.HasPlayer(rt.state, playerID) {
			delete(rt.tokens, playerID)
		}
	}
	return changed
}

func cleanupRemovedPlayers(rt *runtime) {
	for playerID := range rt.tokens {
		if !game.HasPlayer(rt.state, playerID) {
			delete(rt.tokens, playerID)
			closeSession(rt.sessions, playerID)
		}
	}
}

func closeSession(sessions map[string]session, playerID string) {
	current, ok := sessions[playerID]
	if !ok {
		return
	}
	delete(sessions, playerID)
	if current.close != nil {
		current.close()
	}
}

func authorized(tokens map[string]string, playerID, token string) bool {
	expected, ok := tokens[playerID]
	return ok && expected == token && token != ""
}

func cloneTokens(source map[string]string) map[string]string {
	copyMap := make(map[string]string, len(source))
	for id, token := range source {
		copyMap[id] = token
	}
	return copyMap
}

func stopTimer(timer *time.Timer) {
	if timer == nil {
		return
	}
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}
