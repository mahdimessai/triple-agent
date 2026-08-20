package room

import (
	"errors"

	"tripleagent/server/internal/domain"
)

// Sender delivers one player-scoped projection to an attached session.
type Sender func(domain.Projection) error

// Closer terminates an attached session after replacement, disconnect, or room cleanup.
type Closer func()

type messageKind uint8

const (
	messageAddPlayer messageKind = iota
	messageAuthenticate
	messageAttach
	messageDetach
	messageRemovePlayer
	messageSnapshot
	messageCommand
)

type roomMessage struct {
	kind      messageKind
	playerID  string
	name      string
	token     string
	sessionID string
	sender    Sender
	close     Closer
	command   *domain.Command
	reply     chan roomResponse
}

type roomReply struct {
	projection domain.Projection
	replayed   bool
}

type roomResponse struct {
	reply roomReply
	err   error
}

func (r *Room) AddPlayer(playerID, name string) error {
	return r.AddPlayerWithCredential(playerID, name, "")
}

// AddPlayerWithCredential seats a player and registers the reconnect credential
// in the same actor turn, so the seat and credential cannot diverge.
func (r *Room) AddPlayerWithCredential(playerID, name, token string) error {
	reply := make(chan roomResponse, 1)
	if !r.send(roomMessage{
		kind:     messageAddPlayer,
		playerID: playerID,
		name:     name,
		token:    token,
		reply:    reply,
	}) {
		return errors.New("room is closed")
	}
	_, err := r.wait(reply)
	return err
}

// Authenticate verifies that a live seat owns the supplied reconnect token.
func (r *Room) Authenticate(playerID, token string) error {
	reply := make(chan roomResponse, 1)
	if !r.send(roomMessage{kind: messageAuthenticate, playerID: playerID, token: token, reply: reply}) {
		return errors.New("room is closed")
	}
	_, err := r.wait(reply)
	return err
}

func (r *Room) Attach(playerID, sessionID string, sender Sender, close Closer) error {
	reply := make(chan roomResponse, 1)
	if !r.send(roomMessage{
		kind:      messageAttach,
		playerID:  playerID,
		sessionID: sessionID,
		sender:    sender,
		close:     close,
		reply:     reply,
	}) {
		return errors.New("room is closed")
	}
	_, err := r.wait(reply)
	return err
}

// Detach releases a session. The send is synchronous with actor receipt, so a
// subsequent actor operation observes the detach even though no reply is needed.
func (r *Room) Detach(playerID, sessionID string) {
	_ = r.send(roomMessage{kind: messageDetach, playerID: playerID, sessionID: sessionID})
}

// RemovePlayer releases a lobby seat and its reconnect credential atomically.
func (r *Room) RemovePlayer(playerID string) error {
	reply := make(chan roomResponse, 1)
	if !r.send(roomMessage{kind: messageRemovePlayer, playerID: playerID, reply: reply}) {
		return errors.New("room is closed")
	}
	_, err := r.wait(reply)
	return err
}

func (r *Room) Submit(command domain.Command) error {
	_, _, err := r.SubmitForSession("", command)
	return err
}

// SubmitForSession applies one client command. A non-empty session ID makes the
// actor reject the command when that session is no longer the player's current
// one, so a superseded connection cannot act.
func (r *Room) SubmitForSession(sessionID string, command domain.Command) (domain.Projection, bool, error) {
	reply := make(chan roomResponse, 1)
	if !r.send(roomMessage{kind: messageCommand, sessionID: sessionID, command: &command, reply: reply}) {
		return domain.Projection{}, false, errors.New("room is closed")
	}
	result, err := r.wait(reply)
	return result.projection, result.replayed, err
}

func (r *Room) Snapshot(playerID string) (domain.Projection, error) {
	reply := make(chan roomResponse, 1)
	if !r.send(roomMessage{kind: messageSnapshot, playerID: playerID, reply: reply}) {
		return domain.Projection{}, errors.New("room is closed")
	}
	result, err := r.wait(reply)
	return result.projection, err
}

func (r *Room) send(message roomMessage) bool {
	select {
	case <-r.done:
		return false
	case r.commands <- message:
		return true
	}
}

func (r *Room) wait(reply <-chan roomResponse) (roomReply, error) {
	select {
	case result := <-reply:
		return result.reply, result.err
	case <-r.done:
		// A room can retire immediately after placing its final response in the
		// buffered reply channel. Prefer that result over a racing close signal.
		select {
		case result := <-reply:
			return result.reply, result.err
		default:
			return roomReply{}, errors.New("room is closed")
		}
	}
}
