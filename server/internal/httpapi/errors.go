package httpapi

import (
	"errors"
	"net/http"

	"tripleagent/server/internal/game"
	"tripleagent/server/internal/room"
)

type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	return e.Message
}

func newAPIError(status int, code, message string) *APIError {
	return &APIError{Status: status, Code: code, Message: message}
}

func apiErrorFor(err error) *APIError {
	if err == nil {
		return newAPIError(http.StatusInternalServerError, "internal", "Internal server error.")
	}
	if apiErr, ok := errors.AsType[*APIError](err); ok {
		return apiErr
	}

	switch {
	case errors.Is(err, errInvalidJSON):
		return newAPIError(http.StatusBadRequest, "invalid_json", "Invalid JSON.")
	case errors.Is(err, errMultipleJSON):
		return newAPIError(http.StatusBadRequest, "invalid_json", "Request must contain one JSON value.")
	case errors.Is(err, room.ErrRoomNotFound), errors.Is(err, room.ErrClosed):
		return newAPIError(http.StatusNotFound, "room_not_found", "Lobby not found.")
	case errors.Is(err, room.ErrUnauthorized):
		return newAPIError(http.StatusUnauthorized, "unauthorized", "Invalid reconnect token.")
	case errors.Is(err, game.ErrRoomFull):
		return newAPIError(http.StatusConflict, "room_full", "Room is full.")
	case errors.Is(err, game.ErrNameTaken):
		return newAPIError(http.StatusConflict, "name_taken", "That agent name is already seated in this lobby.")
	case errors.Is(err, game.ErrNotAllowed):
		return newAPIError(http.StatusConflict, "not_allowed", "Lobby has already started.")
	case errors.Is(err, game.ErrPlayerNotInRoom):
		return newAPIError(http.StatusUnauthorized, "player_not_in_room", "This player is no longer seated in the room.")
	default:
		return newAPIError(http.StatusInternalServerError, "internal", "Internal server error.")
	}
}

func sessionError(err error) *APIError {
	switch {
	case errors.Is(err, room.ErrRoomNotFound), errors.Is(err, room.ErrClosed):
		return newAPIError(http.StatusGone, "room_gone", "Room is no longer available.")
	case errors.Is(err, room.ErrUnauthorized), errors.Is(err, game.ErrPlayerNotInRoom):
		return newAPIError(http.StatusUnauthorized, "unauthorized", "Authentication is required.")
	case errors.Is(err, room.ErrSessionActive):
		return newAPIError(http.StatusConflict, "session_active", "This seat is already connected in another tab.")
	default:
		return newAPIError(http.StatusInternalServerError, "internal", "Room authentication unavailable.")
	}
}
