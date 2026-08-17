package app

import (
	"errors"
	"testing"

	"tripleagent/server/internal/admission"
	"tripleagent/server/internal/domain"
	"tripleagent/server/internal/room"
)

func TestJoinLobbySeatsThePlayerInTheLiveRoom(t *testing.T) {
	store := admission.NewStore()
	rooms := room.NewManager()
	lobbies := NewLobbies(store, rooms)

	created, err := lobbies.Create(CreateInput{
		PlayerName: "Host",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rooms.Remove(created.RoomID) })

	joined, err := lobbies.Join(JoinInput{
		JoinCode:   created.JoinCode,
		PlayerName: "Agent B",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ValidateReconnectToken(created.RoomID, joined.PlayerID, joined.ReconnectToken); err != nil {
		t.Fatal("admitted player has no usable credential:", err)
	}

	active, ok := rooms.Get(created.RoomID)
	if !ok {
		t.Fatal("created room is not active")
	}
	projection, err := active.Snapshot(joined.PlayerID)
	if err != nil {
		t.Fatal(err)
	}
	if len(projection.Public.Players) != 2 || projection.Private.PlayerID != joined.PlayerID {
		t.Fatalf("live join projection = %#v", projection)
	}
}

func TestJoinLobbyFailsWhenRoomIsFull(t *testing.T) {
	store := admission.NewStore()
	rooms := room.NewManager()
	lobbies := NewLobbies(store, rooms)
	created, err := lobbies.Create(CreateInput{PlayerName: "Host"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rooms.Remove(created.RoomID) })

	active, ok := rooms.Get(created.RoomID)
	if !ok {
		t.Fatal("created room is not active")
	}
	// Fill the live room to its limit (9 players max).
	for _, id := range []string{"actor-2", "actor-3", "actor-4", "actor-5", "actor-6", "actor-7", "actor-8", "actor-9"} {
		if err := active.AddPlayer(id, string(id)); err != nil {
			t.Fatal(err)
		}
	}

	_, err = lobbies.Join(JoinInput{
		JoinCode:   created.JoinCode,
		PlayerName: "Late Agent",
	})
	if !errors.Is(err, domain.ErrRoomFull) {
		t.Fatalf("join error = %v, want %v", err, domain.ErrRoomFull)
	}

	if _, err := active.Snapshot(created.PlayerID); err != nil {
		t.Fatal("live room was closed by a refused admission:", err)
	}
}

func TestLobbySocketCloseReleasesSeatAndCannotReconnect(t *testing.T) {
	store := admission.NewStore()
	rooms := room.NewManager()
	lobbies := NewLobbies(store, rooms)
	sessions := NewSessions(store, rooms)
	created, err := lobbies.Create(CreateInput{PlayerName: "Host"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rooms.Remove(created.RoomID) })
	joined, err := lobbies.Join(JoinInput{JoinCode: created.JoinCode, PlayerName: "Player"})
	if err != nil {
		t.Fatal(err)
	}
	active, ok := rooms.Get(created.RoomID)
	if !ok {
		t.Fatal("created room is not active")
	}

	session, err := sessions.Attach(active, created.RoomID, joined.PlayerID, "session-player", func(domain.Projection) error { return nil }, func() {})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ValidateReconnectToken(created.RoomID, joined.PlayerID, joined.ReconnectToken); err != nil {
		t.Fatal("attached player lost their credential:", err)
	}

	// A socket close in the lobby releases the seat immediately.
	sessions.Detach(session)
	if err := store.ValidateReconnectToken(created.RoomID, joined.PlayerID, joined.ReconnectToken); !errors.Is(err, admission.ErrInvalidToken) {
		t.Fatalf("credential after lobby disconnect = %v, want %v", err, admission.ErrInvalidToken)
	}
	projection, err := active.Snapshot(created.PlayerID)
	if err != nil {
		t.Fatal(err)
	}
	if len(projection.Public.Players) != 1 || projection.Public.Players[0].ID != created.PlayerID {
		t.Fatalf("lobby after socket close = %#v, want only the host", projection.Public.Players)
	}
	if _, err := sessions.Attach(active, created.RoomID, joined.PlayerID, "session-player-reconnected", func(domain.Projection) error { return nil }, func() {}); err == nil {
		t.Fatal("a lobby player reclaimed a released seat with the old session")
	}
	if _, err := active.Snapshot(joined.PlayerID); err == nil {
		t.Fatal("released lobby player remained addressable")
	}

	// A new join receives a fresh identity and is appended after the surviving
	// host, so the former seat is not reserved for the old connection.
	rejoined, err := lobbies.Join(JoinInput{JoinCode: created.JoinCode, PlayerName: "Player"})
	if err != nil {
		t.Fatal(err)
	}
	if rejoined.PlayerID == joined.PlayerID {
		t.Fatal("fresh lobby join reused the released player identity")
	}
	projection, err = active.Snapshot(rejoined.PlayerID)
	if err != nil {
		t.Fatal(err)
	}
	if projection.Public.HostID != created.PlayerID || len(projection.Public.Players) != 2 {
		t.Fatalf("fresh lobby join projection = %#v", projection.Public)
	}
	for _, player := range projection.Public.Players {
		if player.ID == rejoined.PlayerID && player.Seat != 2 {
			t.Fatalf("fresh lobby join seat = %d, want 2", player.Seat)
		}
	}
}

func TestInGameSocketCloseKeepsSeatAndCredentialForReconnect(t *testing.T) {
	store := admission.NewStore()
	rooms := room.NewManager()
	sessions := NewSessions(store, rooms)
	state := domain.NewLobby("room_in_game_reconnect", "host", "Host", domain.DefaultRoomSettings())
	if err := state.AddPlayer("player", "Player"); err != nil {
		t.Fatal(err)
	}
	state.Phase = domain.PhaseDiscussion
	active := rooms.Create(state)
	t.Cleanup(func() { rooms.Remove(state.RoomID) })
	if err := store.Reserve(state.RoomID, "INGAME", "host", "host-token"); err != nil {
		t.Fatal(err)
	}
	if err := store.Grant(state.RoomID, "player", "reconnect-token"); err != nil {
		t.Fatal(err)
	}

	session, err := sessions.Attach(active, state.RoomID, "player", "session-player", func(domain.Projection) error { return nil }, func() {})
	if err != nil {
		t.Fatal(err)
	}
	sessions.Detach(session)
	if err := store.ValidateReconnectToken(state.RoomID, "player", "reconnect-token"); err != nil {
		t.Fatalf("in-game credential after disconnect = %v, want usable credential", err)
	}
	if _, err := sessions.Attach(active, state.RoomID, "player", "session-player-reconnected", func(domain.Projection) error { return nil }, func() {}); err != nil {
		t.Fatalf("in-game player could not reconnect: %v", err)
	}
}

func TestLeaveLobbyTransfersHostAndRemovesTheRoomWhenEmpty(t *testing.T) {
	store := admission.NewStore()
	rooms := room.NewManager()
	lobbies := NewLobbies(store, rooms)
	created, err := lobbies.Create(CreateInput{PlayerName: "Host"})
	if err != nil {
		t.Fatal(err)
	}
	joined, err := lobbies.Join(JoinInput{JoinCode: created.JoinCode, PlayerName: "Player"})
	if err != nil {
		t.Fatal(err)
	}

	if err := lobbies.Leave(LeaveInput{RoomID: created.RoomID, PlayerID: joined.PlayerID, ReconnectToken: joined.ReconnectToken}); err != nil {
		t.Fatal(err)
	}
	active, ok := rooms.Get(created.RoomID)
	if !ok {
		t.Fatal("room was removed while the host was still seated")
	}
	projection, err := active.Snapshot(created.PlayerID)
	if err != nil {
		t.Fatal(err)
	}
	if len(projection.Public.Players) != 1 || projection.Public.HostID != created.PlayerID {
		t.Fatalf("after player leave = %#v", projection.Public)
	}
	if err := store.ValidateReconnectToken(created.RoomID, joined.PlayerID, joined.ReconnectToken); !errors.Is(err, admission.ErrInvalidToken) {
		t.Fatalf("left player credential = %v, want %v", err, admission.ErrInvalidToken)
	}

	if err := lobbies.Leave(LeaveInput{RoomID: created.RoomID, PlayerID: created.PlayerID, ReconnectToken: created.ReconnectToken}); err != nil {
		t.Fatal(err)
	}
	if _, ok := rooms.Get(created.RoomID); ok {
		t.Fatal("room manager retained an empty room")
	}
	if _, _, err := store.ResolveCode(created.JoinCode); !errors.Is(err, admission.ErrRoomNotFound) {
		t.Fatalf("empty lobby code lookup = %v, want lobby-not-found", err)
	}
}

func TestFormerHostCanRejoinAsANormalPlayer(t *testing.T) {
	store := admission.NewStore()
	rooms := room.NewManager()
	lobbies := NewLobbies(store, rooms)
	created, err := lobbies.Create(CreateInput{PlayerName: "Host"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rooms.Remove(created.RoomID) })
	joined, err := lobbies.Join(JoinInput{JoinCode: created.JoinCode, PlayerName: "Player"})
	if err != nil {
		t.Fatal(err)
	}
	if err := lobbies.Leave(LeaveInput{RoomID: created.RoomID, PlayerID: created.PlayerID, ReconnectToken: created.ReconnectToken}); err != nil {
		t.Fatal(err)
	}
	rejoined, err := lobbies.Join(JoinInput{JoinCode: created.JoinCode, PlayerName: "Host"})
	if err != nil {
		t.Fatal(err)
	}
	if rejoined.PlayerID == created.PlayerID {
		t.Fatal("former host reused their old player identity")
	}
	active, ok := rooms.Get(created.RoomID)
	if !ok {
		t.Fatal("room was removed after host left with another player seated")
	}
	projection, err := active.Snapshot(rejoined.PlayerID)
	if err != nil {
		t.Fatal(err)
	}
	if projection.Public.HostID != joined.PlayerID || len(projection.Public.Players) != 2 {
		t.Fatalf("former host rejoin projection = %#v", projection.Public)
	}
}
