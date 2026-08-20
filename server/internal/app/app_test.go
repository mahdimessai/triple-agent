package app

import (
	"errors"
	"testing"
	"time"

	"tripleagent/server/internal/admission"
	"tripleagent/server/internal/domain"
	"tripleagent/server/internal/room"
)

func TestJoinLobbySeatsThePlayerInTheLiveRoom(t *testing.T) {
	store := admission.NewStore()
	rooms := room.NewManager()
	lobbies := NewLobbies(store, rooms)

	created, err := lobbies.Create(CreateInput{PlayerName: "Host"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rooms.Remove(created.RoomID) })

	joined, err := lobbies.Join(JoinInput{JoinCode: created.JoinCode, PlayerName: "Agent B"})
	if err != nil {
		t.Fatal(err)
	}

	active, ok := rooms.Get(created.RoomID)
	if !ok {
		t.Fatal("created room is not active")
	}
	if err := active.Authenticate(joined.PlayerID, joined.ReconnectToken); err != nil {
		t.Fatal("admitted player has no usable credential:", err)
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
	for _, id := range []string{"actor-2", "actor-3", "actor-4", "actor-5", "actor-6", "actor-7", "actor-8", "actor-9"} {
		if err := active.AddPlayer(id, id); err != nil {
			t.Fatal(err)
		}
	}

	_, err = lobbies.Join(JoinInput{JoinCode: created.JoinCode, PlayerName: "Late Agent"})
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
	sessions := NewSessions(rooms)
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
	if err := active.Authenticate(joined.PlayerID, joined.ReconnectToken); err != nil {
		t.Fatal("attached player lost their credential:", err)
	}

	sessions.Detach(session)
	if err := active.Authenticate(joined.PlayerID, joined.ReconnectToken); !errors.Is(err, room.ErrInvalidCredential) {
		t.Fatalf("credential after lobby disconnect = %v, want %v", err, room.ErrInvalidCredential)
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
}

func TestLastLobbySocketCloseRetiresRoomAndReleasesCode(t *testing.T) {
	store := admission.NewStore()
	rooms := room.NewManager()
	lobbies := NewLobbies(store, rooms)
	sessions := NewSessions(rooms)

	created, err := lobbies.Create(CreateInput{PlayerName: "Host"})
	if err != nil {
		t.Fatal(err)
	}
	active, ok := rooms.Get(created.RoomID)
	if !ok {
		t.Fatal("created room is not active")
	}
	session, err := sessions.Attach(active, created.RoomID, created.PlayerID, "host-session", func(domain.Projection) error { return nil }, func() {})
	if err != nil {
		t.Fatal(err)
	}

	sessions.Detach(session)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		_, active := rooms.Get(created.RoomID)
		_, _, codeErr := store.ResolveCode(created.JoinCode)
		if !active && errors.Is(codeErr, admission.ErrRoomNotFound) {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("last lobby disconnect did not retire the room and release its join code")
}

func TestInGameSocketCloseKeepsSeatAndCredentialForReconnect(t *testing.T) {
	rooms := room.NewManager()
	sessions := NewSessions(rooms)
	state := domain.NewLobby("room_in_game_reconnect", "host", "Host", domain.DefaultRoomSettings())
	if err := state.AddPlayer("player", "Player"); err != nil {
		t.Fatal(err)
	}
	state.Phase = domain.PhaseDiscussion
	active := rooms.CreateWithCredentials(state, map[string]string{
		"host":   "host-token",
		"player": "reconnect-token",
	}, nil)
	t.Cleanup(func() { rooms.Remove(state.RoomID) })

	session, err := sessions.Attach(active, state.RoomID, "player", "session-player", func(domain.Projection) error { return nil }, func() {})
	if err != nil {
		t.Fatal(err)
	}
	sessions.Detach(session)
	if err := active.Authenticate("player", "reconnect-token"); err != nil {
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
	if err := active.Authenticate(joined.PlayerID, joined.ReconnectToken); !errors.Is(err, room.ErrInvalidCredential) {
		t.Fatalf("left player credential = %v, want %v", err, room.ErrInvalidCredential)
	}

	if err := lobbies.Leave(LeaveInput{RoomID: created.RoomID, PlayerID: created.PlayerID, ReconnectToken: created.ReconnectToken}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		_, active := rooms.Get(created.RoomID)
		_, _, codeErr := store.ResolveCode(created.JoinCode)
		if !active && errors.Is(codeErr, admission.ErrRoomNotFound) {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("empty lobby was not retired")
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
