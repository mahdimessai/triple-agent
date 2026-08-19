package httpapi

import (
	"net/http"
	"strings"
)

type createLobbyRequest struct {
	PlayerName string `json:"player_name"`
}

type joinLobbyRequest struct {
	JoinCode   string `json:"join_code"`
	PlayerName string `json:"player_name"`
}

type leaveLobbyRequest struct {
	RoomID         string `json:"room_id"`
	PlayerID       string `json:"player_id"`
	ReconnectToken string `json:"reconnect_token"`
}

type lobbyResponse struct {
	RoomID         string `json:"room_id"`
	JoinCode       string `json:"join_code"`
	PlayerID       string `json:"player_id"`
	ReconnectToken string `json:"reconnect_token"`
}

func (h *handler) createLobby(w http.ResponseWriter, r *http.Request) {
	var request createLobbyRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeHTTPError(w, err)
		return
	}
	name := strings.TrimSpace(request.PlayerName)
	if name == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Player name is required.", Code: "player_name_required"})
		return
	}
	created, err := h.rooms.Create(name)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, lobbyResponse{RoomID: created.RoomID, JoinCode: created.JoinCode, PlayerID: created.PlayerID, ReconnectToken: created.ReconnectToken})
}

func (h *handler) joinLobby(w http.ResponseWriter, r *http.Request) {
	var request joinLobbyRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeHTTPError(w, err)
		return
	}
	code := strings.ToUpper(strings.TrimSpace(request.JoinCode))
	if code == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Join code is required.", Code: "join_code_required"})
		return
	}
	name := strings.TrimSpace(request.PlayerName)
	if name == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Player name is required.", Code: "player_name_required"})
		return
	}
	joined, err := h.rooms.Join(code, name)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, lobbyResponse{RoomID: joined.RoomID, JoinCode: joined.JoinCode, PlayerID: joined.PlayerID, ReconnectToken: joined.ReconnectToken})
}

func (h *handler) leaveLobby(w http.ResponseWriter, r *http.Request) {
	var request leaveLobbyRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeHTTPError(w, err)
		return
	}
	if strings.TrimSpace(request.RoomID) == "" || strings.TrimSpace(request.PlayerID) == "" || strings.TrimSpace(request.ReconnectToken) == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Room ID, player ID, and reconnect token are required.", Code: "leave_identity_required"})
		return
	}
	if err := h.rooms.Leave(request.RoomID, request.PlayerID, request.ReconnectToken); err != nil {
		writeHTTPError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"left": true})
}
