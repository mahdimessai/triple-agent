package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"tripleagent/server/internal/game"
	"tripleagent/server/internal/room"

	"github.com/gorilla/websocket"
)

func websocketURL(serverURL string, identity room.Identity) string {
	parsed, _ := url.Parse(serverURL)
	parsed.Scheme = "ws"
	parsed.Path = "/ws"
	query := parsed.Query()
	query.Set("room_id", identity.RoomID)
	query.Set("player_id", identity.PlayerID)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func authenticateWebSocket(t *testing.T, conn *websocket.Conn, identity room.Identity) game.Projection {
	t.Helper()
	if err := conn.WriteJSON(authMessage{Kind: "room.auth", ReconnectToken: identity.ReconnectToken}); err != nil {
		t.Fatal(err)
	}
	var authenticated map[string]any
	if err := conn.ReadJSON(&authenticated); err != nil {
		t.Fatal(err)
	}
	if authenticated["type"] != "session.authenticated" {
		t.Fatalf("first message = %#v", authenticated)
	}
	var projection game.Projection
	if err := conn.ReadJSON(&projection); err != nil {
		t.Fatal(err)
	}
	if projection.Type != "room.projection" {
		t.Fatalf("initial projection = %+v", projection)
	}
	return projection
}

func TestWebSocketAuthenticationCommandAndProjectionContract(t *testing.T) {
	registry := room.NewRegistry()
	defer registry.Close()
	identity, err := registry.Create("Host")
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(New(registry))
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial(websocketURL(server.URL, identity), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	initial := authenticateWebSocket(t, conn, identity)
	if err := conn.WriteJSON(commandMessage{Kind: string(game.CommandSetReady), RequestID: "request-1", ExpectedVersion: initial.Public.Version}); err != nil {
		t.Fatal(err)
	}

	var ack *commandAck
	var updated *game.Projection
	for range 2 {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
		var header struct{ Type string `json:"type"` }
		if err := json.Unmarshal(payload, &header); err != nil {
			t.Fatal(err)
		}
		switch header.Type {
		case "command.ack":
			var value commandAck
			if err := json.Unmarshal(payload, &value); err != nil {
				t.Fatal(err)
			}
			ack = &value
		case "room.projection":
			var value game.Projection
			if err := json.Unmarshal(payload, &value); err != nil {
				t.Fatal(err)
			}
			updated = &value
		default:
			t.Fatalf("unexpected message: %s", payload)
		}
	}
	if ack == nil || !ack.OK || ack.RequestID != "request-1" {
		t.Fatalf("ack = %+v", ack)
	}
	if updated == nil || updated.Public.Version != initial.Public.Version+1 || !updated.Public.Players[0].Ready {
		t.Fatalf("updated projection = %+v", updated)
	}
}

func TestWebSocketRejectsUnknownCommandsBeforeGameDispatch(t *testing.T) {
	registry := room.NewRegistry()
	defer registry.Close()
	identity, err := registry.Create("Host")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(New(registry))
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial(websocketURL(server.URL, identity), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	initial := authenticateWebSocket(t, conn, identity)

	if err := conn.WriteJSON(commandMessage{Kind: "not.a.command", RequestID: "request-2", ExpectedVersion: initial.Public.Version}); err != nil {
		t.Fatal(err)
	}
	var ack commandAck
	if err := conn.ReadJSON(&ack); err != nil {
		t.Fatal(err)
	}
	if ack.OK || ack.Code != "unknown_command" || ack.RequestID != "request-2" {
		t.Fatalf("ack = %+v", ack)
	}
}

func TestWebSocketJSONRejectsUnknownFields(t *testing.T) {
	var auth authMessage
	if err := decodeWebSocketJSON([]byte(`{"type":"room.auth","reconnect_token":"secret","mystery":true}`), &auth); err == nil {
		t.Fatal("unknown websocket field was accepted")
	}
}

func TestWebSocketOriginPolicy(t *testing.T) {
	registry := room.NewRegistry()
	defer registry.Close()
	identity, err := registry.Create("Host")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(NewWithOptions(registry, Options{AllowedOrigins: []string{"https://game.example"}}))
	defer server.Close()

	badHeader := http.Header{"Origin": []string{"https://evil.example"}}
	conn, response, err := websocket.DefaultDialer.Dial(websocketURL(server.URL, identity), badHeader)
	if conn != nil {
		conn.Close()
	}
	if err == nil || response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatalf("bad origin response=%v err=%v", response, err)
	}

	goodHeader := http.Header{"Origin": []string{"https://game.example"}}
	conn, _, err = websocket.DefaultDialer.Dial(websocketURL(server.URL, identity), goodHeader)
	if err != nil {
		t.Fatalf("allowed origin rejected: %v", err)
	}
	conn.Close()
}
