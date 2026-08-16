package app

import (
	"tripleagent/server/internal/admission"
	"tripleagent/server/internal/room"
)

// Lobbies covers the room's entry points: creating one, joining one, leaving one.
type Lobbies struct {
	// admit owns cross-room join codes, credentials, and idempotent join claims.
	admit *admission.Store
	// rooms owns live room actors and their game state.
	rooms *room.Manager
}

func NewLobbies(admit *admission.Store, rooms *room.Manager) *Lobbies {
	return &Lobbies{admit: admit, rooms: rooms}
}

// Sessions covers one player's live connection to a room they already belong to.
type Sessions struct {
	// admit authenticates reconnect credentials and revokes them on seat release.
	admit *admission.Store
	// rooms resolves authenticated room IDs to their live actors.
	rooms *room.Manager
}

func NewSessions(admit *admission.Store, rooms *room.Manager) *Sessions {
	return &Sessions{admit: admit, rooms: rooms}
}
