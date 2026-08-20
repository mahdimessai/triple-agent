package app

import (
	"tripleagent/server/internal/admission"
	"tripleagent/server/internal/room"
)

// Lobbies coordinates cross-room admission with one live room actor.
type Lobbies struct {
	// admit owns only the global join-code index.
	admit *admission.Store
	// rooms owns live room actors, including seats and reconnect credentials.
	rooms *room.Manager
}

func NewLobbies(admit *admission.Store, rooms *room.Manager) *Lobbies {
	return &Lobbies{admit: admit, rooms: rooms}
}

// Sessions covers one player's live connection to a room they already belong to.
type Sessions struct {
	rooms *room.Manager
}

func NewSessions(rooms *room.Manager) *Sessions {
	return &Sessions{rooms: rooms}
}
