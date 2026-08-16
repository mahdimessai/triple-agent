package domain

import (
	"testing"
	"time"
)

func TestApplyLeaveRemovesPlayerAndTransfersHost(t *testing.T) {
	state := NewLobby("room", "host", "Host", DefaultRoomSettings())
	if err := state.AddPlayer("player", "Player"); err != nil {
		t.Fatal(err)
	}
	state.Players["player"] = PlayerState{ID: "player", Name: "Player", Seat: 2, Connected: true}

	transition, err := ApplyLeave(state, "host", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if transition.State.HostID != "player" {
		t.Fatalf("host after leave = %q, want player", transition.State.HostID)
	}
	if len(transition.State.PlayerOrder) != 1 || transition.State.PlayerOrder[0] != "player" {
		t.Fatalf("player order after leave = %#v", transition.State.PlayerOrder)
	}
	if transition.State.Players["player"].Seat != 1 {
		t.Fatalf("remaining player seat = %d, want 1", transition.State.Players["player"].Seat)
	}
	if transition.Event.Kind != "PLAYER_LEFT" || !transition.Changed {
		t.Fatalf("leave transition = %#v", transition)
	}
}

func TestApplyLeaveRejectsStartedRoom(t *testing.T) {
	state := NewLobby("room", "host", "Host", DefaultRoomSettings())
	state.Phase = PhaseRoleReveal
	if _, err := ApplyLeave(state, "host", time.Now()); err != ErrNotAllowed {
		t.Fatalf("leave error = %v, want %v", err, ErrNotAllowed)
	}
}

func TestApplyDisconnectLobbyKeepsSeatAndTransfersHost(t *testing.T) {
	state := NewLobby("room", "host", "Host", DefaultRoomSettings())
	if err := state.AddPlayer("player", "Player"); err != nil {
		t.Fatal(err)
	}
	state.Players["host"] = PlayerState{ID: "host", Name: "Host", Seat: 1, Ready: true, Connected: true}
	state.Players["player"] = PlayerState{ID: "player", Name: "Player", Seat: 2, Connected: true}

	transition, err := ApplyDisconnect(state, "host", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(transition.State.PlayerOrder) != 2 || transition.State.HostID != "player" {
		t.Fatalf("lobby after host disconnect = %#v", transition.State)
	}
	disconnected := transition.State.Players["host"]
	if disconnected.Connected || disconnected.Ready {
		t.Fatalf("disconnected lobby host = %#v, want offline and not ready", disconnected)
	}
}
