package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tripleagent/server/internal/room"
)

func doJSON(t *testing.T, client http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	client.ServeHTTP(response, req)
	return response
}

func TestHealth(t *testing.T) {
	registry := room.NewRegistry()
	defer registry.Close()
	response := doJSON(t, New(registry), http.MethodGet, "/healthz", "")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	var payload map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["status"] != "ok" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestCreateJoinLeaveContract(t *testing.T) {
	registry := room.NewRegistry()
	defer registry.Close()
	handler := New(registry)

	createdResponse := doJSON(t, handler, http.MethodPost, "/api/lobbies", `{"player_name":" Host "}`)
	if createdResponse.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", createdResponse.Code, createdResponse.Body.String())
	}
	var created lobbyResponse
	if err := json.Unmarshal(createdResponse.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.RoomID == "" || created.JoinCode == "" || created.PlayerID == "" || created.ReconnectToken == "" {
		t.Fatalf("incomplete create response: %+v", created)
	}

	joinedResponse := doJSON(t, handler, http.MethodPost, "/api/lobbies/join", `{"join_code":"`+created.JoinCode+`","player_name":"Guest"}`)
	if joinedResponse.Code != http.StatusCreated {
		t.Fatalf("join status=%d body=%s", joinedResponse.Code, joinedResponse.Body.String())
	}
	var joined lobbyResponse
	if err := json.Unmarshal(joinedResponse.Body.Bytes(), &joined); err != nil {
		t.Fatal(err)
	}
	if joined.RoomID != created.RoomID || joined.PlayerID == created.PlayerID || joined.ReconnectToken == "" {
		t.Fatalf("bad join response: %+v", joined)
	}

	leaveResponse := doJSON(t, handler, http.MethodPost, "/api/lobbies/leave", `{"room_id":"`+joined.RoomID+`","player_id":"`+joined.PlayerID+`","reconnect_token":"`+joined.ReconnectToken+`"}`)
	if leaveResponse.Code != http.StatusOK {
		t.Fatalf("leave status=%d body=%s", leaveResponse.Code, leaveResponse.Body.String())
	}
}

func TestJSONValidationPreservesStrictContract(t *testing.T) {
	registry := room.NewRegistry()
	defer registry.Close()
	handler := New(registry)

	cases := []struct {
		name string
		body string
	}{
		{name: "invalid", body: `{"player_name":`},
		{name: "unknown field", body: `{"player_name":"A","mystery":true}`},
		{name: "two values", body: `{"player_name":"A"} {"player_name":"B"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			response := doJSON(t, handler, http.MethodPost, "/api/lobbies", tc.body)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestMissingLobbyAndBadTokenStatusCodes(t *testing.T) {
	registry := room.NewRegistry()
	defer registry.Close()
	handler := New(registry)
	missing := doJSON(t, handler, http.MethodPost, "/api/lobbies/join", `{"join_code":"ABC123","player_name":"Guest"}`)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing join status=%d body=%s", missing.Code, missing.Body.String())
	}

	createdResponse := doJSON(t, handler, http.MethodPost, "/api/lobbies", `{"player_name":"Host"}`)
	var created lobbyResponse
	_ = json.Unmarshal(createdResponse.Body.Bytes(), &created)
	badLeave := doJSON(t, handler, http.MethodPost, "/api/lobbies/leave", `{"room_id":"`+created.RoomID+`","player_id":"`+created.PlayerID+`","reconnect_token":"wrong"}`)
	if badLeave.Code != http.StatusUnauthorized {
		t.Fatalf("bad token status=%d body=%s", badLeave.Code, badLeave.Body.String())
	}
}

func TestSessionMissingRoomMapsToGone(t *testing.T) {
	status, code, _ := sessionError(room.ErrRoomNotFound)
	if status != http.StatusGone || code != "room_gone" {
		t.Fatalf("status=%d code=%q", status, code)
	}
}

func TestCommandAckDoesNotExposeUnknownInternalError(t *testing.T) {
	code, message := commandError(assertionError("database password leaked"))
	if code != "internal" || message != "internal server error" {
		t.Fatalf("code=%q message=%q", code, message)
	}
}

type assertionError string

func (e assertionError) Error() string { return string(e) }

func TestCORSAndOptionsContract(t *testing.T) {
	registry := room.NewRegistry()
	defer registry.Close()
	handler := New(registry)

	req := httptest.NewRequest(http.MethodOptions, "/api/lobbies", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status=%d", response.Code)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "*" || response.Header().Get("Access-Control-Allow-Headers") != "*" || response.Header().Get("Access-Control-Allow-Methods") != "*" {
		t.Fatalf("unexpected cors headers: %#v", response.Header())
	}
}

func TestJSONBodyLimit(t *testing.T) {
	registry := room.NewRegistry()
	defer registry.Close()
	body := `{"player_name":"` + string(bytes.Repeat([]byte{'x'}, maxJSONBody+1)) + `"}`
	response := doJSON(t, New(registry), http.MethodPost, "/api/lobbies", body)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestErrorClassifiersCoverProtocolCases(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "invalid json", err: errInvalidJSON, status: http.StatusBadRequest, code: "invalid_json"},
		{name: "room missing", err: room.ErrRoomNotFound, status: http.StatusNotFound, code: "room_not_found"},
		{name: "unauthorized", err: room.ErrUnauthorized, status: http.StatusUnauthorized, code: "unauthorized"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, code, _ := classifyHTTPError(tc.err)
			if status != tc.status || code != tc.code {
				t.Fatalf("status=%d code=%q", status, code)
			}
		})
	}

	commandCases := []struct {
		err  error
		code string
	}{
		{err: room.ErrVersionConflict, code: "stale_version"},
		{err: room.ErrSessionGone, code: "session_gone"},
	}
	for _, tc := range commandCases {
		code, message := commandError(tc.err)
		if code != tc.code || message == "" {
			t.Fatalf("command error %v => code=%q message=%q", tc.err, code, message)
		}
	}
}
