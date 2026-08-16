package transport

import "net/http"

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", h.Health)
	mux.HandleFunc("POST /api/lobbies", h.CreateLobby)
	mux.HandleFunc("POST /api/lobbies/join", h.JoinLobby)
	mux.HandleFunc("POST /api/lobbies/leave", h.LeaveLobby)
	mux.HandleFunc("GET /ws", h.WebSocket)
}
