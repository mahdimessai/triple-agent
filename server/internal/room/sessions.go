package room

import (
	"time"

	"tripleagent/server/internal/domain"
)

type roomSession struct {
	// id distinguishes the current connection from an older replaced connection.
	id string
	// sender delivers projections to this player's connection.
	sender Sender
	// close terminates this connection when it is replaced or the room ends.
	close Closer
}

// handleAttach returns true when applying a failed initial delivery leaves the
// lobby empty and the room should retire.
func (r *Room) handleAttach(runtime *runtimeState, message roomMessage) bool {
	player, ok := runtime.game.Players[message.playerID]
	if !ok {
		message.reply <- roomResponse{err: domain.ErrPlayerNotInRoom}
		return false
	}
	if previous, exists := runtime.sessions[message.playerID]; exists && previous.close != nil {
		previous.close()
	}

	player.Connected = true
	runtime.game.Players[message.playerID] = player
	runtime.sessions[message.playerID] = roomSession{id: message.sessionID, sender: message.sender, close: message.close}

	if err := message.sender(domain.Project(runtime.game, message.playerID)); err != nil {
		delete(runtime.sessions, message.playerID)
		transition, disconnectErr := domain.ApplyDisconnect(runtime.game, message.playerID, time.Now().UTC())
		if disconnectErr == nil && transition.Changed {
			runtime.game = transition.State
			pruneCredentials(runtime.credentials, runtime.game)
		}
		message.reply <- roomResponse{err: err}
		if len(runtime.game.PlayerOrder) == 0 {
			return true
		}
		return r.broadcast(runtime)
	}

	message.reply <- roomResponse{}
	return r.broadcastExcept(runtime, message.playerID)
}

// handleDetach returns true when the detach releases the final lobby seat.
func (r *Room) handleDetach(runtime *runtimeState, message roomMessage) bool {
	current, exists := runtime.sessions[message.playerID]
	if !exists || current.id != message.sessionID {
		return false
	}
	delete(runtime.sessions, message.playerID)

	transition, err := domain.ApplyDisconnect(runtime.game, message.playerID, time.Now().UTC())
	if err != nil || !transition.Changed {
		return false
	}
	runtime.game = transition.State
	pruneCredentials(runtime.credentials, runtime.game)
	if len(runtime.game.PlayerOrder) == 0 {
		return true
	}
	return r.broadcast(runtime)
}
