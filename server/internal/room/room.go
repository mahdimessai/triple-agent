package room

import (
	"sync"
	"time"

	"tripleagent/server/internal/game"
)

const (
	defaultRoomLifetime      = 1 * time.Hour
	defaultEndedRoomLifetime = 15 * time.Minute
)

// Room represents an active room actor orchestrating game state, timers, and client sessions.
type Room struct {
	id    string
	inbox chan RoomMessage
	done  chan struct{}

	closeOnce sync.Once
	onClose   func(*Room)

	lifetime   time.Duration
	endedAfter time.Duration
}

func newRoom(roomID string, state game.State, tokens *TokenDirectory, onClose func(*Room)) *Room {
	return newRoomWithLifetimes(roomID, state, tokens, onClose, defaultRoomLifetime, defaultEndedRoomLifetime)
}

func newRoomWithLifetimes(roomID string, state game.State, tokens *TokenDirectory, onClose func(*Room), lifetime, endedAfter time.Duration) *Room {
	r := &Room{
		id:         roomID,
		inbox:      make(chan RoomMessage),
		done:       make(chan struct{}),
		onClose:    onClose,
		lifetime:   lifetime,
		endedAfter: endedAfter,
	}
	core := NewRoomCore(roomID, state, tokens)
	go r.loop(core)
	return r
}

func (r *Room) Close() {
	r.closeOnce.Do(func() {
		close(r.done)
	})
}

func (r *Room) Join(playerID, name, reconnectToken string) error {
	reply := make(chan error, 1)
	if err := r.dispatch(JoinCmd{PlayerID: playerID, Name: name, Token: reconnectToken, Reply: reply}); err != nil {
		return err
	}
	select {
	case <-r.done:
		return ErrClosed
	case err := <-reply:
		return err
	}
}

func (r *Room) Leave(playerID, reconnectToken string) error {
	reply := make(chan error, 1)
	if err := r.dispatch(LeaveCmd{PlayerID: playerID, Token: reconnectToken, Reply: reply}); err != nil {
		return err
	}
	select {
	case <-r.done:
		return ErrClosed
	case err := <-reply:
		return err
	}
}

func (r *Room) PlayerIDForToken(token string) (string, error) {
	reply := make(chan PlayerIDResult, 1)
	if err := r.dispatch(PlayerIDForTokenQuery{Token: token, Reply: reply}); err != nil {
		return "", err
	}
	select {
	case <-r.done:
		return "", ErrClosed
	case res := <-reply:
		return res.PlayerID, res.Err
	}
}

func (r *Room) AttachSession(token string, session ClientSession) (string, game.Projection, error) {
	reply := make(chan AttachResult, 1)
	if err := r.dispatch(AttachSessionCmd{Token: token, Session: session, Reply: reply}); err != nil {
		return "", game.Projection{}, err
	}
	select {
	case <-r.done:
		return "", game.Projection{}, ErrClosed
	case res := <-reply:
		return res.PlayerID, res.Projection, res.Err
	}
}

func (r *Room) Attach(playerID, reconnectToken, sessionID string, send func(game.Projection) error, close func()) error {
	session := CallbackSession{
		SessionID: sessionID,
		SendFunc:  send,
		CloseFunc: close,
	}
	_, _, err := r.AttachSession(reconnectToken, session)
	return err
}

func (r *Room) Detach(playerID, sessionID string) {
	reply := make(chan error, 1)
	_ = r.dispatch(DetachSessionCmd{PlayerID: playerID, SessionID: sessionID, Reply: reply})
}

func (r *Room) Command(playerID, sessionID string, expectedVersion uint64, command game.Command) error {
	reply := make(chan error, 1)
	if err := r.dispatch(ExecGameCmd{
		PlayerID:        playerID,
		SessionID:       sessionID,
		ExpectedVersion: expectedVersion,
		Command:         command,
		Reply:           reply,
	}); err != nil {
		return err
	}
	select {
	case <-r.done:
		return ErrClosed
	case err := <-reply:
		return err
	}
}

func (r *Room) Snapshot(playerID string) (game.Projection, error) {
	reply := make(chan SnapshotResult, 1)
	if err := r.dispatch(SnapshotQuery{PlayerID: playerID, Reply: reply}); err != nil {
		return game.Projection{}, err
	}
	select {
	case <-r.done:
		return game.Projection{}, ErrClosed
	case res := <-reply:
		return res.Projection, res.Err
	}
}

func (r *Room) dispatch(msg RoomMessage) error {
	select {
	case <-r.done:
		return ErrClosed
	case r.inbox <- msg:
		return nil
	}
}

func (r *Room) loop(core *RoomCore) {
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
		if core.State.DiscussionDeadline == nil || (core.State.Phase != game.PhaseDiscussion && core.State.Phase != game.PhaseOperationInterlude) {
			return
		}
		duration := time.Until(*core.State.DiscussionDeadline)
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
		core.CloseAllSessions()
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
			next, err := game.AdvanceDeadline(core.State, now.UTC())
			if err == nil && next.Version != core.State.Version {
				core.State = next
				if r.broadcast(core, now.UTC(), "") {
					return
				}
			}
			resetDeadline()

		case rawMsg := <-r.inbox:
			now := time.Now().UTC()
			switch msg := rawMsg.(type) {
			case PlayerIDForTokenQuery:
				playerID, ok := core.Tokens.PlayerID(msg.Token)
				if !ok {
					msg.Reply <- PlayerIDResult{Err: ErrUnauthorized}
					continue
				}
				msg.Reply <- PlayerIDResult{PlayerID: playerID}

			case JoinCmd:
				err := core.HandleJoin(msg.PlayerID, msg.Name, msg.Token)
				if err == nil {
					if r.broadcast(core, now, "") {
						msg.Reply <- nil
						return
					}
					resetExpiry(r.lifetime)
				}
				msg.Reply <- err

			case LeaveCmd:
				err := core.HandleLeave(msg.PlayerID, msg.Token)
				msg.Reply <- err
				if err != nil {
					continue
				}
				if core.IsEmpty() {
					return
				}
				if r.broadcast(core, now, "") {
					return
				}
				resetExpiry(r.lifetime)

			case AttachSessionCmd:
				playerID, projection, err := core.HandleAttach(msg.Token, msg.Session)
				if err != nil {
					msg.Reply <- AttachResult{Err: err}
					continue
				}
				// Deliver initial projection directly to the newly attached session
				if sendErr := msg.Session.Send(projection); sendErr != nil {
					core.EvictFailedSessions([]string{playerID}, now)
					msg.Reply <- AttachResult{Err: sendErr}
					if core.IsEmpty() {
						return
					}
					continue
				}
				msg.Reply <- AttachResult{PlayerID: playerID, Projection: projection}
				if r.broadcast(core, now, playerID) {
					return
				}
				resetDeadline()

			case DetachSessionCmd:
				changed, err := core.HandleDetach(msg.PlayerID, msg.SessionID, now)
				msg.Reply <- err
				if err != nil {
					continue
				}
				if core.IsEmpty() {
					return
				}
				if changed {
					if r.broadcast(core, now, "") {
						return
					}
					resetDeadline()
				}

			case ExecGameCmd:
				err := core.HandleGameCommand(msg.PlayerID, msg.SessionID, msg.ExpectedVersion, msg.Command, now)
				msg.Reply <- err
				if err != nil {
					continue
				}
				if core.IsEmpty() {
					return
				}
				if r.broadcast(core, now, "") {
					return
				}
				resetDeadline()
				switch core.State.Phase {
				case game.PhaseEnd:
					resetExpiry(r.endedAfter)
				default:
					if msg.Command.Kind == game.CommandRematch || msg.Command.Kind == game.CommandStartMatch {
						resetExpiry(r.lifetime)
					}
				}

			case SnapshotQuery:
				proj, err := core.Snapshot(msg.PlayerID)
				msg.Reply <- SnapshotResult{Projection: proj, Err: err}
			}
		}
	}
}

func (r *Room) broadcast(core *RoomCore, now time.Time, excludePlayerID string) bool {
	for {
		failed := r.deliverProjections(core, excludePlayerID)
		if len(failed) == 0 {
			return core.IsEmpty()
		}
		changed := core.EvictFailedSessions(failed, now)
		if core.IsEmpty() {
			return true
		}
		if !changed {
			return false
		}
		excludePlayerID = "" // Retrying full broadcast as state changed
	}
}

func (r *Room) deliverProjections(core *RoomCore, excludePlayerID string) []string {
	var failed []string
	public := game.PublicProjectionFor(r.id, core.State)
	for playerID, session := range core.Sessions {
		if playerID == excludePlayerID {
			continue
		}
		projection := game.ProjectWithPublic(core.State, playerID, public)
		if session == nil || session.Send(projection) != nil {
			failed = append(failed, playerID)
		}
	}
	return failed
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
