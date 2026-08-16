package transport

import (
	"net/http"
	"strings"

	"tripleagent/server/internal/app"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
)

type createLobbyRequest struct {
	// PlayerName is the requested display name for the host seat.
	PlayerName string `json:"player_name"`
}

// CreateLobby validates and stores a new lobby, then creates its in-memory room.
func (h *Handler) CreateLobby(w http.ResponseWriter, r *http.Request) {
	var request createLobbyRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, err)
		return
	}

	playerName := strings.TrimSpace(request.PlayerName)
	if playerName == "" {
		writeError(w, fault.New("player_name is required",
			ftag.With(ftag.InvalidArgument),
			fmsg.WithDesc("player_name is required", "Player name is required."),
		))
		return
	}

	created, err := h.lobbies.Create(app.CreateInput{PlayerName: playerName})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, lobbyResponse{
		RoomID:         string(created.RoomID),
		JoinCode:       created.JoinCode,
		PlayerID:       string(created.PlayerID),
		ReconnectToken: created.ReconnectToken,
	})
}
