package room

import (
	"errors"

	"tripleagent/server/internal/domain"
)

// Sender delivers one player-scoped projection to an attached session.
type Sender func(domain.Projection) error

// Closer terminates an attached session after replacement, disconnect, or room cleanup.
type Closer func()

type roomMessage struct {
	// kind selects the actor operation, such as attach, detach, or command.
	kind string
	// playerID identifies the player affected by the operation.
	playerID string
	// name is the display name for a player being added to the room.
	name string
	// sessionID identifies the connection that is attaching, detaching, or submitting.
	sessionID string
	// sender delivers the initial projection to a newly attached session.
	sender Sender
	// close terminates a replaced or invalidated session.
	close Closer
	// command is the client command the actor must validate and apply.
	command *domain.Command
	// commit persists an external side effect while the actor still owns the transition.
	commit func(domain.GameState) error
	// reply returns the actor result to the goroutine that submitted the message.
	reply chan roomResponse
}

type roomReply struct {
	// projection is the actor's player-scoped view after a command or snapshot.
	projection domain.Projection
	// state is returned when a player removal needs the updated room state for cleanup.
	state domain.GameState
	// replayed reports that a request ID reused an already completed command.
	replayed bool
}

type roomResponse struct {
	// reply contains successful operation data returned by the actor.
	reply roomReply
	// err contains the domain or session failure produced while handling the message.
	err error
}

func (r *Room) AddPlayer(playerID string, name string) error {
	return r.AddPlayerWithCommit(playerID, name, nil)
}

// AddPlayerWithCommit seats a player and runs commit inside the actor before the
// seat becomes visible. A commit failure discards the seat, so a credential is
// only ever written for a player the room actually admitted.
func (r *Room) AddPlayerWithCommit(playerID string, name string, commit func(domain.GameState) error) error {
	reply := make(chan roomResponse, 1)
	if !r.send(roomMessage{kind: "add_player", playerID: playerID, name: name, commit: commit, reply: reply}) {
		return errors.New("room is closed")
	}
	_, err := r.wait(reply)
	return err
}

func (r *Room) Attach(playerID string, sessionID string, sender Sender, close Closer) error {
	reply := make(chan roomResponse, 1)
	if !r.send(roomMessage{
		kind:      "attach",
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

// Detach releases a session while preserving the player's seat for reconnect.
func (r *Room) Detach(playerID string, sessionID string) {
	reply := make(chan roomResponse, 1)
	if !r.send(roomMessage{kind: "detach", playerID: playerID, sessionID: sessionID, reply: reply}) {
		return
	}
	_, _ = r.wait(reply)
}

func (r *Room) RemovePlayerWithCommit(playerID string, commit func(domain.GameState) error) (domain.GameState, error) {
	reply := make(chan roomResponse, 1)
	if !r.send(roomMessage{kind: "remove_player", playerID: playerID, commit: commit, reply: reply}) {
		return domain.GameState{}, errors.New("room is closed")
	}
	result, err := r.wait(reply)
	return result.state, err
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
	if !r.send(roomMessage{kind: "command", sessionID: sessionID, command: &command, reply: reply}) {
		return domain.Projection{}, false, errors.New("room is closed")
	}
	result, err := r.wait(reply)
	return result.projection, result.replayed, err
}

func (r *Room) Snapshot(playerID string) (domain.Projection, error) {
	reply := make(chan roomResponse, 1)
	if !r.send(roomMessage{kind: "snapshot", playerID: playerID, reply: reply}) {
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
		return roomReply{}, errors.New("room is closed")
	}
}
