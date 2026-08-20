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
	lobbies  *app.Lobbies
	sessions *app.Sessions
}

// NewHandler is retained for existing tests and callers. New process entry
// points should compose dependencies themselves and use NewHandlerWithServices.
func NewHandler(admit *admission.Store, rooms *room.Manager) *Handler {
	return NewHandlerWithServices(app.NewLobbies(admit, rooms), app.NewSessions(rooms))
}

// NewHandlerWithServices keeps transport focused on HTTP/WebSocket mechanics;
// the process composition root decides which application services it receives.
func NewHandlerWithServices(lobbies *app.Lobbies, sessions *app.Sessions) *Handler {
	return &Handler{lobbies: lobbies, sessions: sessions}
}
