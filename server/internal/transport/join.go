package transport

import (
	"net/http"
	"strings"

	"tripleagent/server/internal/app"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
)

type joinLobbyRequest struct {
	// JoinCode identifies the lobby to join.
	JoinCode string `json:"join_code"`
	// PlayerName is the display name to add to the lobby.
	PlayerName string `json:"player_name"`
}

// JoinLobby validates the join request, adds the player to the room, and returns their session credentials.
func (h *Handler) JoinLobby(w http.ResponseWriter, r *http.Request) {
	var request joinLobbyRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, err)
		return
	}

	joinCode := strings.ToUpper(strings.TrimSpace(request.JoinCode))
	if joinCode == "" {
		writeError(w, fault.New("join_code is required",
			ftag.With(ftag.InvalidArgument),
			fmsg.WithDesc("join_code is required", "Join code is required."),
		))
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

	joined, err := h.lobbies.Join(app.JoinInput{
		JoinCode:   joinCode,
		PlayerName: playerName,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, lobbyResponse{
		RoomID:         string(joined.RoomID),
		JoinCode:       joined.JoinCode,
		PlayerID:       string(joined.PlayerID),
		ReconnectToken: joined.ReconnectToken,
	})
}
