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
	JoinCode       string `json:"join_code"`
	ReconnectToken string `json:"reconnect_token"`
}

type lobbyResponse struct {
	JoinCode       string `json:"join_code"`
	ReconnectToken string `json:"reconnect_token"`
}

type leaveLobbyResponse struct {
	Left bool `json:"left"`
}

func (h *handler) createLobby(w http.ResponseWriter, r *http.Request) {
	var request createLobbyRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeHTTPError(w, err)
		return
	}
	name := strings.TrimSpace(request.PlayerName)
	if name == "" {
		writeHTTPError(w, newAPIError(http.StatusBadRequest, "player_name_required", "Player name is required."))
		return
	}
	credentials, err := h.roomManager.Create(name)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, lobbyResponse{
		JoinCode:       credentials.JoinCode,
		ReconnectToken: credentials.ReconnectToken,
	})
}

func (h *handler) joinLobby(w http.ResponseWriter, r *http.Request) {
	var request joinLobbyRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeHTTPError(w, err)
		return
	}
	code := strings.ToUpper(strings.TrimSpace(request.JoinCode))
	if code == "" {
		writeHTTPError(w, newAPIError(http.StatusBadRequest, "join_code_required", "Join code is required."))
		return
	}
	name := strings.TrimSpace(request.PlayerName)
	if name == "" {
		writeHTTPError(w, newAPIError(http.StatusBadRequest, "player_name_required", "Player name is required."))
		return
	}
	credentials, err := h.roomManager.Join(code, name)
	if err != nil {
		writeHTTPError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, lobbyResponse{
		JoinCode:       credentials.JoinCode,
		ReconnectToken: credentials.ReconnectToken,
	})
}

func (h *handler) leaveLobby(w http.ResponseWriter, r *http.Request) {
	var request leaveLobbyRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeHTTPError(w, err)
		return
	}
	if strings.TrimSpace(request.JoinCode) == "" || strings.TrimSpace(request.ReconnectToken) == "" {
		writeHTTPError(w, newAPIError(http.StatusBadRequest, "leave_identity_required", "Join code and reconnect token are required."))
		return
	}
	if err := h.roomManager.Leave(request.JoinCode, request.ReconnectToken); err != nil {
		writeHTTPError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, leaveLobbyResponse{Left: true})
}
