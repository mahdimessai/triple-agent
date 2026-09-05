package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"tripleagent/server/internal/game"
	"tripleagent/server/internal/room"

	"github.com/gorilla/websocket"
)

const maxJSONBody = 2 << 10

var (
	errInvalidJSON  = errors.New("invalid JSON")
	errMultipleJSON = errors.New("request must contain one JSON value")
)

type handler struct {
	roomManager *room.RoomManager
	upgrader    websocket.Upgrader
}

func New(roomManager *room.RoomManager) http.Handler {
	h := &handler{
		roomManager: roomManager,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(*http.Request) bool {
				return true
			},
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("POST /api/lobbies", h.createLobby)
	mux.HandleFunc("POST /api/lobbies/join", h.joinLobby)
	mux.HandleFunc("POST /api/lobbies/leave", h.leaveLobby)
	mux.HandleFunc("GET /ws", h.websocket)
	return withCORS(mux)
}

func (h *handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header["Access-Control-Allow-Origin"] = []string{"*"}
		header["Access-Control-Allow-Headers"] = []string{"*"}
		header["Access-Control-Allow-Methods"] = []string{"*"}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

type healthResponse struct {
	Status string `json:"status"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header()["Content-Type"] = []string{"application/json"}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return newAPIError(http.StatusBadRequest, "invalid_json", "Invalid JSON.")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return newAPIError(http.StatusBadRequest, "invalid_json", "Request must contain one JSON value.")
	}
	return nil
}

func writeHTTPError(w http.ResponseWriter, err error) {
	apiErr := apiErrorFor(err)
	writeJSON(w, apiErr.Status, errorResponse{Error: apiErr.Message, Code: apiErr.Code})
}

func commandError(err error) (string, string) {
	if err == nil {
		return "", ""
	}
	switch {
	case errors.Is(err, room.ErrVersionConflict):
		return "stale_version", "stale room version"
	case errors.Is(err, room.ErrSessionGone):
		return "session_gone", "session is no longer active"
	case errors.Is(err, game.ErrNotAllowed):
		return "not_allowed", "command is not allowed"
	case errors.Is(err, game.ErrNotEnoughPlayers):
		return "not_enough_players", "not enough players"
	case errors.Is(err, game.ErrNotReady):
		return "not_ready", "all players must be ready"
	case errors.Is(err, game.ErrInvalidTarget):
		return "invalid_target", "invalid target"
	case errors.Is(err, game.ErrUnknownOperation):
		return "unknown_operation", "unknown operation"
	case errors.Is(err, game.ErrNoEligibleOperations):
		return "no_eligible_operations", "no eligible operations are enabled"
	case errors.Is(err, game.ErrAlreadySubmitted):
		return "already_submitted", "command already submitted"
	case errors.Is(err, game.ErrPlayerNotInRoom):
		return "player_not_in_room", "player is not in room"
	default:
		return "internal", "internal server error"
	}
}
