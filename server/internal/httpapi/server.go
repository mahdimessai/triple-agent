package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
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

type Options struct {
	Logger         *slog.Logger
	AllowedOrigins []string
}

type handler struct {
	rooms    *room.Registry
	logger   *slog.Logger
	origins  originPolicy
	upgrader websocket.Upgrader
}

func New(rooms *room.Registry) http.Handler {
	return NewWithOptions(rooms, Options{})
}

func NewWithOptions(rooms *room.Registry, options Options) http.Handler {
	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}
	origins := newOriginPolicy(options.AllowedOrigins)
	h := &handler{
		rooms:   rooms,
		logger:  logger,
		origins: origins,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     origins.allowsRequest,
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("POST /api/lobbies", h.createLobby)
	mux.HandleFunc("POST /api/lobbies/join", h.joinLobby)
	mux.HandleFunc("POST /api/lobbies/leave", h.leaveLobby)
	mux.HandleFunc("GET /ws", h.websocket)
	return origins.middleware(mux)
}

func (h *handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return errInvalidJSON
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errMultipleJSON
	}
	return nil
}

func (h *handler) writeHTTPError(w http.ResponseWriter, r *http.Request, err error) {
	status, code, message := classifyHTTPError(err)
	if status >= http.StatusInternalServerError {
		h.logger.Error("http request failed", "method", r.Method, "path", r.URL.Path, "error", err)
	}
	writeJSON(w, status, errorResponse{Error: message, Code: code})
}

func classifyHTTPError(err error) (int, string, string) {
	switch {
	case errors.Is(err, errInvalidJSON):
		return http.StatusBadRequest, "invalid_json", "Invalid JSON."
	case errors.Is(err, errMultipleJSON):
		return http.StatusBadRequest, "invalid_json", "Request must contain one JSON value."
	case errors.Is(err, room.ErrRoomNotFound), errors.Is(err, room.ErrClosed):
		return http.StatusNotFound, "room_not_found", "Lobby not found."
	case errors.Is(err, room.ErrUnauthorized):
		return http.StatusUnauthorized, "unauthorized", "Invalid reconnect token."
	case errors.Is(err, game.ErrRoomFull):
		return http.StatusConflict, "room_full", "Room is full."
	case errors.Is(err, game.ErrNotAllowed):
		return http.StatusConflict, "not_allowed", "Lobby has already started."
	case errors.Is(err, game.ErrPlayerNotInRoom):
		return http.StatusUnauthorized, "player_not_in_room", "This player is no longer seated in the room."
	default:
		return http.StatusInternalServerError, "internal", "Internal server error."
	}
}

func sessionError(err error) (int, string, string) {
	switch {
	case errors.Is(err, room.ErrRoomNotFound), errors.Is(err, room.ErrClosed):
		return http.StatusGone, "room_gone", "Room is no longer available."
	case errors.Is(err, room.ErrUnauthorized), errors.Is(err, game.ErrPlayerNotInRoom):
		return http.StatusUnauthorized, "unauthorized", "Authentication is required."
	default:
		return http.StatusInternalServerError, "internal", "Room authentication unavailable."
	}
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
