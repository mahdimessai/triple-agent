package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

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
	var createdFields map[string]json.RawMessage
	if err := json.Unmarshal(createdResponse.Body.Bytes(), &createdFields); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"room_id", "player_id"} {
		if _, exists := createdFields[field]; exists {
			t.Fatalf("create response still exposes internal field %q: %s", field, createdResponse.Body.String())
		}
	}
	if created.JoinCode == "" || created.ReconnectToken == "" {
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
	if joined.JoinCode != created.JoinCode || joined.ReconnectToken == "" {
		t.Fatalf("bad join response: %+v", joined)
	}

	leaveResponse := doJSON(t, handler, http.MethodPost, "/api/lobbies/leave", `{"join_code":"`+joined.JoinCode+`","reconnect_token":"`+joined.ReconnectToken+`"}`)
	if leaveResponse.Code != http.StatusOK {
		t.Fatalf("leave status=%d body=%s", leaveResponse.Code, leaveResponse.Body.String())
	}
}

func TestJoinRejectsDuplicateName(t *testing.T) {
	registry := room.NewRegistry()
	defer registry.Close()
	handler := New(registry)

	createdResponse := doJSON(t, handler, http.MethodPost, "/api/lobbies", `{"player_name":"Host"}`)
	if createdResponse.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", createdResponse.Code, createdResponse.Body.String())
	}
	var created lobbyResponse
	if err := json.Unmarshal(createdResponse.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	duplicate := doJSON(t, handler, http.MethodPost, "/api/lobbies/join", `{"join_code":"`+created.JoinCode+`","player_name":"host"}`)
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate join status=%d body=%s", duplicate.Code, duplicate.Body.String())
	}
	var payload errorResponse
	if err := json.Unmarshal(duplicate.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Code != "name_taken" || payload.Error == "" {
		t.Fatalf("duplicate join payload=%+v", payload)
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
	badLeave := doJSON(t, handler, http.MethodPost, "/api/lobbies/leave", `{"join_code":"`+created.JoinCode+`","reconnect_token":"wrong"}`)
	if badLeave.Code != http.StatusUnauthorized {
		t.Fatalf("bad token status=%d body=%s", badLeave.Code, badLeave.Body.String())
	}
}

func TestSessionMissingRoomMapsToGone(t *testing.T) {
	apiErr := sessionError(room.ErrRoomNotFound)
	if apiErr.Status != http.StatusGone || apiErr.Code != "room_gone" || apiErr.Message != "Room is no longer available." {
		t.Fatalf("api error=%+v", apiErr)
	}
}

func TestSessionActiveMapsToConflictForWebSocket(t *testing.T) {
	apiErr := sessionError(room.ErrSessionActive)
	if apiErr.Status != http.StatusConflict || apiErr.Code != "session_active" || apiErr.Message != "This seat is already connected in another tab." {
		t.Fatalf("api error=%+v", apiErr)
	}
}

func TestWebSocketClosePreservesSeatForReconnect(t *testing.T) {
	registry := room.NewRegistry()
	defer registry.Close()
	created, err := registry.Create("Host")
	if err != nil {
		t.Fatal(err)
	}
	active, playerID, err := registry.Resolve(created.JoinCode, created.ReconnectToken)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(New(registry))
	defer server.Close()
	connect := func() *websocket.Conn {
		t.Helper()
		conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/ws?join_code="+created.JoinCode, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = conn.Close() })
		if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
			t.Fatal(err)
		}
		if err := conn.WriteJSON(authMessage{Kind: "room.auth", ReconnectToken: created.ReconnectToken}); err != nil {
			t.Fatal(err)
		}
		var authenticated sessionAuthenticatedResponse
		if err := conn.ReadJSON(&authenticated); err != nil {
			t.Fatal(err)
		}
		if authenticated.Type != "session.authenticated" {
			t.Fatalf("expected authentication, got %+v", authenticated)
		}
		return conn
	}
	conn := connect()
	if err := conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	deadline := time.Now().Add(2 * time.Second)
	for {
		projection, err := active.Snapshot(playerID)
		if err != nil {
			t.Fatalf("socket close removed the seat: %v", err)
		}
		if !projection.Public.Players[0].Connected {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("socket close did not detach the player")
		}
		time.Sleep(time.Millisecond)
	}
	connect()
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

func TestAPIErrorsCoverProtocolCases(t *testing.T) {
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
			apiErr := apiErrorFor(tc.err)
			if apiErr.Status != tc.status || apiErr.Code != tc.code {
				t.Fatalf("api error=%+v", apiErr)
			}
		})
	}

	apiErr := newAPIError(http.StatusConflict, "conflict", "Conflict.")
	if apiErr.Status != http.StatusConflict || apiErr.Code != "conflict" || apiErr.Message != "Conflict." || apiErr.Error() != "Conflict." {
		t.Fatalf("unexpected API error: %+v", apiErr)
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
