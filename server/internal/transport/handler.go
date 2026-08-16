package transport

import (
	"time"

	"tripleagent/server/internal/admission"
	"tripleagent/server/internal/app"
	"tripleagent/server/internal/room"
)

const (
	maxJSONBody = 2 << 10
	authTimeout = 5 * time.Second
	readLimit   = 16 << 10
	writeWait   = 10 * time.Second
	pongWait    = 60 * time.Second
	pingPeriod  = (pongWait * 9) / 10
)

type Handler struct {
	// lobbies handles HTTP operations that create, join, and leave rooms.
	lobbies *app.Lobbies
	// sessions handles reconnect authentication and live room attachment.
	sessions *app.Sessions
}

func NewHandler(admit *admission.Store, rooms *room.Manager) *Handler {
	return &Handler{
		lobbies:  app.NewLobbies(admit, rooms),
		sessions: app.NewSessions(admit, rooms),
	}
}
