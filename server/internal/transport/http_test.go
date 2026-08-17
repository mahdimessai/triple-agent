package transport

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"tripleagent/server/internal/admission"
	"tripleagent/server/internal/domain"
	"tripleagent/server/internal/room"

	"github.com/gorilla/websocket"
)

func TestCreateLobbyAndWebSocketProjection(t *testing.T) {
	handler := NewHandler(admission.NewStore(), room.NewManager())
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/lobbies", handler.CreateLobby)
	mux.HandleFunc("GET /ws", handler.WebSocket)
	server := httptest.NewServer(mux)
	defer server.Close()

	requestBody := strings.NewReader(`{"player_name":"Agent A"}`)
	response, err := http.Post(server.URL+"/api/lobbies", "application/json", requestBody)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d", response.StatusCode)
	}
	var lobbyResponse lobbyResponse
	if err := json.NewDecoder(response.Body).Decode(&lobbyResponse); err != nil {
		t.Fatal(err)
	}

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?" + url.Values{
		"room_id":   []string{string(lobbyResponse.RoomID)},
		"player_id": []string{string(lobbyResponse.PlayerID)},
	}.Encode()
	connection, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()

	initial := authenticateConnection(t, connection, lobbyResponse.ReconnectToken)
	if initial.Type != "room.projection" || initial.Public.Phase != domain.PhaseLobby {
		t.Fatalf("initial projection = %#v", initial)
	}

	if err := connection.WriteJSON(map[string]any{
		"kind":             domain.CommandSetReady,
		"request_id":       "req-ready",
		"expected_version": initial.Public.Version,
	}); err != nil {
		t.Fatal(err)
	}
	var projection domain.Projection
	if err := connection.ReadJSON(&projection); err != nil {
		t.Fatal(err)
	}
	if projection.Public.Version != initial.Public.Version+1 || !projection.Public.Players[0].Ready || projection.Private.CanSubmit {
		t.Fatalf("ready projection = %#v", projection)
	}
	var ack map[string]any
	if err := connection.ReadJSON(&ack); err != nil {
		t.Fatal(err)
	}
	if ack["type"] != "command.ack" || ack["ok"] != true {
		t.Fatalf("ack = %#v", ack)
	}

	// A browser reload in the lobby keeps the seat and returns the current
	// projection, including changes made before the connection closed.
	_ = connection.Close()
	reconnected, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer reconnected.Close()
	if err := reconnected.WriteJSON(map[string]any{"type": "room.auth", "reconnect_token": lobbyResponse.ReconnectToken}); err != nil {
		t.Fatal(err)
	}
	if err := reconnected.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	reconnectedProjection := authenticateConnection(t, reconnected, lobbyResponse.ReconnectToken)
	if reconnectedProjection.Public.Version <= projection.Public.Version || reconnectedProjection.Public.Players[0].Ready || !reconnectedProjection.Public.Players[0].Connected {
		t.Fatalf("reconnected lobby projection = %#v, want the post-disconnect state", reconnectedProjection)
	}
}

func TestWebSocketReplacementClosesOldSessionAndKeepsNewSessionAuthorized(t *testing.T) {
	handler := NewHandler(admission.NewStore(), room.NewManager())
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/lobbies", handler.CreateLobby)
	mux.HandleFunc("GET /ws", handler.WebSocket)
	server := httptest.NewServer(mux)
	defer server.Close()

	response, err := http.Post(server.URL+"/api/lobbies", "application/json", strings.NewReader(`{"player_name":"Agent A"}`))
	if err != nil {
		t.Fatal(err)
	}
	var created lobbyResponse
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		response.Body.Close()
		t.Fatal(err)
	}
	response.Body.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?" + url.Values{
		"room_id":   []string{string(created.RoomID)},
		"player_id": []string{string(created.PlayerID)},
	}.Encode()

	oldConnection, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer oldConnection.Close()
	oldProjection := authenticateConnection(t, oldConnection, created.ReconnectToken)

	newConnection, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer newConnection.Close()
	newProjection := authenticateConnection(t, newConnection, created.ReconnectToken)
	if newProjection.Public.Version != oldProjection.Public.Version {
		t.Fatalf("replacement snapshot version = %d, old version = %d", newProjection.Public.Version, oldProjection.Public.Version)
	}

	if err := oldConnection.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := oldConnection.ReadMessage(); err == nil {
		t.Fatal("replaced WebSocket session remained readable")
	}

	if err := newConnection.WriteJSON(map[string]any{
		"kind":             domain.CommandSetReady,
		"request_id":       "replacement-ready",
		"expected_version": newProjection.Public.Version,
	}); err != nil {
		t.Fatal(err)
	}
	readyProjection := readProjection(t, newConnection)
	if !readyProjection.Public.Players[0].Ready {
		t.Fatalf("replacement session could not submit command: %#v", readyProjection)
	}
	var ack map[string]any
	if err := newConnection.ReadJSON(&ack); err != nil {
		t.Fatal(err)
	}
	if ack["type"] != "command.ack" || ack["ok"] != true {
		t.Fatalf("replacement command ack = %#v", ack)
	}
}

func TestWebSocketRejectsInvalidFirstFrame(t *testing.T) {
	handler := NewHandler(admission.NewStore(), room.NewManager())
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/lobbies", handler.CreateLobby)
	mux.HandleFunc("GET /ws", handler.WebSocket)
	server := httptest.NewServer(mux)
	defer server.Close()

	response, err := http.Post(server.URL+"/api/lobbies", "application/json", strings.NewReader(`{"player_name":"Agent A"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var created lobbyResponse
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?" + url.Values{
		"room_id":   []string{string(created.RoomID)},
		"player_id": []string{string(created.PlayerID)},
	}.Encode()
	connection, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if err := connection.WriteJSON(map[string]any{"type": "room.auth", "reconnect_token": "wrong"}); err != nil {
		t.Fatal(err)
	}
	var message map[string]any
	if err := connection.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := connection.ReadJSON(&message); err != nil {
		t.Fatal(err)
	}
	if message["type"] != "session.error" || message["status"] != float64(http.StatusUnauthorized) {
		t.Fatalf("invalid authentication response = %#v", message)
	}
}

func TestJoinLobbySeatsPlayerAndAuthenticates(t *testing.T) {
	store := admission.NewStore()
	rooms := room.NewManager()
	handler := NewHandler(store, rooms)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/lobbies", handler.CreateLobby)
	mux.HandleFunc("POST /api/lobbies/join", handler.JoinLobby)
	server := httptest.NewServer(mux)
	defer server.Close()

	createResponse, err := http.Post(server.URL+"/api/lobbies", "application/json", strings.NewReader(`{"player_name":"Host"}`))
	if err != nil {
		t.Fatal(err)
	}
	var created lobbyResponse
	if err := json.NewDecoder(createResponse.Body).Decode(&created); err != nil {
		createResponse.Body.Close()
		t.Fatal(err)
	}
	createResponse.Body.Close()

	body := fmt.Sprintf(`{"join_code":"%s","player_name":"Agent B"}`, created.JoinCode)
	joinResponse, err := http.Post(server.URL+"/api/lobbies/join", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	var joined lobbyResponse
	if err := json.NewDecoder(joinResponse.Body).Decode(&joined); err != nil {
		joinResponse.Body.Close()
		t.Fatal(err)
	}
	joinResponse.Body.Close()

	if joined.PlayerID == "" || joined.ReconnectToken == "" {
		t.Fatalf("invalid join response: %#v", joined)
	}
	active, ok := rooms.Get(string(created.RoomID))
	if !ok {
		t.Fatal("created room is not active")
	}
	projection, err := active.Snapshot(string(joined.PlayerID))
	if err != nil {
		t.Fatal(err)
	}
	if len(projection.Public.Players) != 2 {
		t.Fatalf("expected 2 players in room, got: %#v", projection.Public.Players)
	}
	if err := store.ValidateReconnectToken(string(created.RoomID), string(joined.PlayerID), joined.ReconnectToken); err != nil {
		t.Fatal("server token did not authenticate:", err)
	}
}

type websocketTestClient struct {
	connection *websocket.Conn
	player     lobbyResponse
}

func TestFivePlayerWebSocketMatchAndRematch(t *testing.T) {
	store := admission.NewStore()
	handler := NewHandler(store, room.NewManager())
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/lobbies", handler.CreateLobby)
	mux.HandleFunc("POST /api/lobbies/join", handler.JoinLobby)
	mux.HandleFunc("GET /ws", handler.WebSocket)
	server := httptest.NewServer(mux)
	defer server.Close()

	response, err := http.Post(server.URL+"/api/lobbies", "application/json", strings.NewReader(`{"player_name":"Agent A"}`))
	if err != nil {
		t.Fatal(err)
	}
	var host lobbyResponse
	if err := json.NewDecoder(response.Body).Decode(&host); err != nil {
		response.Body.Close()
		t.Fatal(err)
	}
	response.Body.Close()

	players := []websocketTestClient{{player: host}}
	for _, name := range []string{"Agent B", "Agent C", "Agent D", "Agent E"} {
		joinBody := strings.NewReader(`{"join_code":"` + host.JoinCode + `","player_name":"` + name + `"}`)
		joinResponse, joinErr := http.Post(server.URL+"/api/lobbies/join", "application/json", joinBody)
		if joinErr != nil {
			t.Fatal(joinErr)
		}
		var joined lobbyResponse
		if err := json.NewDecoder(joinResponse.Body).Decode(&joined); err != nil {
			joinResponse.Body.Close()
			t.Fatal(err)
		}
		joinResponse.Body.Close()
		players = append(players, websocketTestClient{player: joined})
	}

	var projection domain.Projection
	for index := range players {
		player := players[index].player
		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?" + url.Values{
			"room_id":   []string{string(player.RoomID)},
			"player_id": []string{string(player.PlayerID)},
		}.Encode()
		connection, _, dialErr := websocket.DefaultDialer.Dial(wsURL, nil)
		if dialErr != nil {
			t.Fatal(dialErr)
		}
		players[index].connection = connection
		t.Cleanup(func() { _ = connection.Close() })
		if index == 0 {
			projection = authenticateConnection(t, connection, player.ReconnectToken)
		} else {
			_ = authenticateConnection(t, connection, player.ReconnectToken)
		}
		for existing := 0; existing < index; existing++ {
			current := readProjection(t, players[existing].connection)
			if existing == 0 {
				projection = current
			}
		}
	}

	clients := make([]*websocket.Conn, len(players))
	for index := range players {
		clients[index] = players[index].connection
	}

	projection = sendCommandAndCollect(t, clients, 0, projection.Public.Version, domain.CommandSetOperationEnabled, "enable-swap", nil, map[string]any{"operation_kind": "Swap", "operation_enabled": true})
	for index := range clients {
		projection = sendCommandAndCollect(t, clients, index, projection.Public.Version, domain.CommandSetReady, "ready-"+string(rune('a'+index)), nil, nil)
	}
	projection = sendCommandAndCollect(t, clients, 0, projection.Public.Version, domain.CommandStartMatch, "start", nil, map[string]any{"operation_kind": "Swap"})
	if projection.Public.Phase != domain.PhaseRoleReveal {
		t.Fatalf("start phase = %s, want %s", projection.Public.Phase, domain.PhaseRoleReveal)
	}
	// A started match refuses new seats. The live room is the only authority on
	// that now, so the rejection surfaces as a join conflict rather than a
	// vanished join code.
	startedJoin, err := http.Post(server.URL+"/api/lobbies/join", "application/json", strings.NewReader(fmt.Sprintf(`{"join_code":%q,"player_name":"Latecomer"}`, host.JoinCode)))
	if err != nil {
		t.Fatal(err)
	}
	startedJoin.Body.Close()
	if startedJoin.StatusCode != http.StatusConflict {
		t.Fatalf("join after start status = %d, want %d", startedJoin.StatusCode, http.StatusConflict)
	}
	for index := range clients {
		projection = sendCommandAndCollect(t, clients, index, projection.Public.Version, domain.CommandAcknowledgeRole, "role-"+string(rune('a'+index)), nil, nil)
	}
	indexOf := func(id string) int {
		for index, entry := range players {
			if string(entry.player.PlayerID) == id {
				return index
			}
		}
		t.Fatalf("no client for player %s", id)
		return 0
	}
	active0 := indexOf(projection.Public.ActivePlayerID)
	if err := clients[active0].WriteJSON(map[string]any{"kind": "room.resync"}); err != nil {
		t.Fatal(err)
	}
	projection = readProjection(t, clients[active0])
	if projection.Public.Phase != domain.PhaseOperationInput || projection.Private.PlayerID != string(players[active0].player.PlayerID) {
		t.Fatalf("operation input projection = %#v", projection)
	}
	if len(projection.Private.LegalTargetIDs) != 4 {
		t.Fatalf("legal operation targets = %#v, want four other players", projection.Private.LegalTargetIDs)
	}

	target0 := players[1].player.PlayerID
	if target0 == players[active0].player.PlayerID {
		target0 = players[0].player.PlayerID
	}
	projection = sendCommandAndCollect(t, clients, active0, projection.Public.Version, domain.CommandResolveOperation, "operation", []string{string(target0)}, nil)
	if projection.Public.Phase != domain.PhaseOperationResult || projection.Private.OperationResult == nil {
		t.Fatalf("operation result projection = %#v", projection)
	}
	projection = sendCommandAndCollect(t, clients, active0, projection.Public.Version, domain.CommandOperationExplainDone, "explain", nil, nil)
	// The rest of the table each receive an operation, separated by the timed
	// interlude, before the room reaches its final discussion.
	for step := 0; projection.Public.Phase != domain.PhaseDiscussion; step++ {
		if step > 4*len(clients)+8 {
			t.Fatalf("match never reached the final discussion, stuck in %s", projection.Public.Phase)
		}
		switch projection.Public.Phase {
		case domain.PhaseOperationInterlude:
			projection = sendCommandAndCollect(t, clients, 0, projection.Public.Version, domain.CommandAdvanceInterlude, fmt.Sprintf("interlude-%d", step), nil, nil)
		case domain.PhaseOperationResult:
			active := indexOf(projection.Public.ActivePlayerID)
			projection = sendCommandAndCollect(t, clients, active, projection.Public.Version, domain.CommandOperationExplainDone, fmt.Sprintf("explain-%d", step), nil, nil)
		case domain.PhaseOperationInput:
			active := indexOf(projection.Public.ActivePlayerID)
			targets := projection.Public.Operation.TargetCount
			if targets == 0 {
				targets = 1
			}
			chosen := make([]string, 0, targets)
			for _, entry := range players {
				if string(entry.player.PlayerID) != projection.Public.ActivePlayerID && len(chosen) < targets {
					chosen = append(chosen, string(entry.player.PlayerID))
				}
			}
			projection = sendCommandAndCollect(t, clients, active, projection.Public.Version, domain.CommandResolveOperation, fmt.Sprintf("operation-%d", step), chosen, map[string]any{"choice": "STAY"})
		default:
			t.Fatalf("unexpected phase %s while advancing operations", projection.Public.Phase)
		}
	}
	for index := range clients {
		projection = sendCommandAndCollect(t, clients, index, projection.Public.Version, domain.CommandAdvanceDiscussion, fmt.Sprintf("vote-open-%d", index), nil, nil)
	}
	if projection.Public.Phase != domain.PhaseVoteInput {
		t.Fatalf("vote phase = %s, want %s", projection.Public.Phase, domain.PhaseVoteInput)
	}
	for index := range clients {
		target := players[0].player.PlayerID
		if index == 0 {
			target = players[1].player.PlayerID
		}
		projection = sendCommandAndCollect(t, clients, index, projection.Public.Version, domain.CommandSubmitVote, "vote-"+string(rune('a'+index)), nil, map[string]any{"target_id": string(target)})
	}
	if projection.Public.Phase != domain.PhaseResultsIntro {
		t.Fatalf("results phase = %s, want %s", projection.Public.Phase, domain.PhaseResultsIntro)
	}

	expectedPhases := []domain.Phase{domain.PhaseVoteResults, domain.PhaseImprisonment, domain.PhaseAgencyReveal, domain.PhaseOutcomeReveal, domain.PhaseLeaderboard, domain.PhaseOutOfLoop, domain.PhaseEnd}
	for index, expected := range expectedPhases {
		projection = sendCommandAndCollect(t, clients, 0, projection.Public.Version, domain.CommandContinueResults, "results-"+string(rune('a'+index)), nil, nil)
		if projection.Public.Phase != expected {
			t.Fatalf("results phase %d = %s, want %s", index, projection.Public.Phase, expected)
		}
	}
	if len(projection.Public.Leaderboard) != 5 {
		t.Fatalf("leaderboard length = %d, want 5", len(projection.Public.Leaderboard))
	}

	projection = sendCommandAndCollect(t, clients, 0, projection.Public.Version, domain.CommandRematch, "rematch", nil, nil)
	if projection.Public.Phase != domain.PhaseLobby || projection.Public.Version == 0 {
		t.Fatalf("rematch projection = %#v", projection.Public)
	}
	for _, player := range projection.Public.Players {
		if player.Ready || !player.Connected {
			t.Fatalf("rematch player projection = %#v", player)
		}
	}
}

func readProjection(t *testing.T, connection *websocket.Conn) domain.Projection {
	t.Helper()
	if err := connection.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	var projection domain.Projection
	if err := connection.ReadJSON(&projection); err != nil {
		t.Fatal(err)
	}
	return projection
}

func authenticateConnection(t *testing.T, connection *websocket.Conn, token string) domain.Projection {
	t.Helper()
	if err := connection.WriteJSON(map[string]any{"type": "room.auth", "reconnect_token": token}); err != nil {
		t.Fatal(err)
	}
	if err := connection.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	var authenticated map[string]any
	if err := connection.ReadJSON(&authenticated); err != nil {
		t.Fatal(err)
	}
	if authenticated["type"] != "session.authenticated" {
		t.Fatalf("authentication response = %#v", authenticated)
	}
	return readProjection(t, connection)
}

func sendCommandAndCollect(t *testing.T, clients []*websocket.Conn, actorIndex int, version uint64, kind domain.CommandKind, requestID string, targetIDs []string, extra map[string]any) domain.Projection {
	t.Helper()
	message := map[string]any{
		"kind":             kind,
		"request_id":       requestID,
		"expected_version": version,
		"target_ids":       targetIDs,
	}
	for key, value := range extra {
		message[key] = value
	}
	if err := clients[actorIndex].WriteJSON(message); err != nil {
		t.Fatal(err)
	}
	projections := make([]domain.Projection, len(clients))
	for index, client := range clients {
		projections[index] = readProjection(t, client)
	}
	var ack map[string]any
	if err := clients[actorIndex].ReadJSON(&ack); err != nil {
		t.Fatal(err)
	}
	if ok, exists := ack["ok"].(bool); !exists || !ok {
		t.Fatalf("command %s was rejected: %#v", kind, ack)
	}
	return projections[actorIndex]
}

func TestLobbyEndpointsRequirePlayerName(t *testing.T) {
	handler := NewHandler(admission.NewStore(), room.NewManager())
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/lobbies", handler.CreateLobby)
	mux.HandleFunc("POST /api/lobbies/join", handler.JoinLobby)
	server := httptest.NewServer(mux)
	defer server.Close()

	createResp, err := http.Post(server.URL+"/api/lobbies", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	createResp.Body.Close()
	if createResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("create status = %d, want %d", createResp.StatusCode, http.StatusBadRequest)
	}

	createResp, err = http.Post(server.URL+"/api/lobbies", "application/json", strings.NewReader(`{"player_name":"Host"}`))
	if err != nil {
		t.Fatal(err)
	}
	var created lobbyResponse
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		createResp.Body.Close()
		t.Fatal(err)
	}
	createResp.Body.Close()

	joinResp, err := http.Post(server.URL+"/api/lobbies/join", "application/json", strings.NewReader(fmt.Sprintf(`{"join_code":%q}`, created.JoinCode)))
	if err != nil {
		t.Fatal(err)
	}
	joinResp.Body.Close()
	if joinResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("join status = %d, want %d", joinResp.StatusCode, http.StatusBadRequest)
	}
}

func TestWebSocketDiscussionTimerConfiguration(t *testing.T) {
	handler := NewHandler(admission.NewStore(), room.NewManager())
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/lobbies", handler.CreateLobby)
	mux.HandleFunc("GET /ws", handler.WebSocket)
	server := httptest.NewServer(mux)
	defer server.Close()

	response, err := http.Post(server.URL+"/api/lobbies", "application/json", strings.NewReader(`{"player_name":"Host"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var lobbyResponse lobbyResponse
	if err := json.NewDecoder(response.Body).Decode(&lobbyResponse); err != nil {
		t.Fatal(err)
	}

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?" + url.Values{
		"room_id":   []string{string(lobbyResponse.RoomID)},
		"player_id": []string{string(lobbyResponse.PlayerID)},
	}.Encode()
	connection, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()

	initial := authenticateConnection(t, connection, lobbyResponse.ReconnectToken)

	// Configure discussion timer to 420 seconds
	if err := connection.WriteJSON(map[string]any{
		"kind":                     domain.CommandSetDiscussionTimer,
		"request_id":               "req-timer-1",
		"expected_version":         initial.Public.Version,
		"discussion_timer_enabled": true,
		"discussion_seconds":       420,
	}); err != nil {
		t.Fatal(err)
	}
	var projection domain.Projection
	if err := connection.ReadJSON(&projection); err != nil {
		t.Fatal(err)
	}
	var ack map[string]any
	if err := connection.ReadJSON(&ack); err != nil {
		t.Fatal(err)
	}
	if ack["type"] != "command.ack" || ack["ok"] != true {
		t.Fatalf("ack = %#v", ack)
	}
	if projection.Public.Settings.DiscussionSeconds != 420 || !projection.Public.Settings.DiscussionTimerEnabled {
		t.Fatalf("projection settings = %#v, want discussion_seconds=420 enabled=true", projection.Public.Settings)
	}
}
