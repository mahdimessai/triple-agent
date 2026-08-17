package room

import (
	"errors"
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

func (r *Room) handleAttach(state *domain.GameState, sessions map[string]roomSession, message roomMessage) {
	player, ok := state.Players[message.playerID]
	if !ok {
		message.reply <- roomResponse{err: errors.New("player is not in room")}
		return
	}
	previousHost := state.HostID
	if previous, exists := sessions[message.playerID]; exists && previous.close != nil {
		previous.close()
	}
	player.Connected = true
	state.Players[message.playerID] = player
	// Reconnecting only restores presence. Host transfer is decided when the
	// player disconnects, so a returning player never reclaims a role that has
	// already moved to someone else.
	sessions[message.playerID] = roomSession{id: message.sessionID, sender: message.sender, close: message.close}
	if err := message.sender(domain.Project(*state, message.playerID)); err != nil {
		delete(sessions, message.playerID)
		player.Connected = false
		state.Players[message.playerID] = player
		state.HostID = previousHost
		message.reply <- roomResponse{err: err}
		return
	}
	message.reply <- roomResponse{}
	r.broadcastExcept(state, sessions, message.playerID)
}

func (r *Room) handleDetach(state *domain.GameState, sessions map[string]roomSession, message roomMessage) {
	current, exists := sessions[message.playerID]
	if !exists || current.id != message.sessionID {
		if message.reply != nil {
			message.reply <- roomResponse{}
		}
		return
	}
	delete(sessions, message.playerID)
	transition, err := domain.ApplyDisconnect(*state, message.playerID, time.Now().UTC())
	if err == nil && transition.Changed {
		*state = transition.State
		r.broadcast(state, sessions)
	}
	if message.reply != nil {
		message.reply <- roomResponse{err: err}
	}
}
