package transport

import (
	"net/http"
	"strings"

	"tripleagent/server/internal/app"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
)

type leaveLobbyRequest struct {
	// RoomID identifies the lobby to leave.
	RoomID string `json:"room_id"`
	// PlayerID identifies the seat to release.
	PlayerID string `json:"player_id"`
	// ReconnectToken authenticates the player before the seat is released.
	ReconnectToken string `json:"reconnect_token"`
}

// LeaveLobby authenticates a player with their reconnect token before asking
// the live room to give up their seat.
func (h *Handler) LeaveLobby(w http.ResponseWriter, r *http.Request) {
	var request leaveLobbyRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, err)
		return
	}

	if request.RoomID == "" || request.PlayerID == "" || strings.TrimSpace(request.ReconnectToken) == "" {
		writeError(w, fault.New("room_id, player_id, and reconnect_token are required",
			ftag.With(ftag.InvalidArgument),
			fmsg.WithDesc(
				"room_id, player_id, and reconnect_token are required",
				"Room ID, player ID, and reconnect token are required.",
			),
		))
		return
	}

	err := h.lobbies.Leave(app.LeaveInput{
		RoomID:         string(request.RoomID),
		PlayerID:       string(request.PlayerID),
		ReconnectToken: request.ReconnectToken,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"left": true})
}
