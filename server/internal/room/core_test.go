package room

import (
	"errors"
	"testing"
	"time"

	"tripleagent/server/internal/game"
)

func TestHandleAttachRejectsWhenSeatAlreadyLive(t *testing.T) {
	tokens := TokensFromMap(map[string]string{"p1": "tok"})
	core := NewRoomCore("room_1", game.NewLobby("p1", "Host"), tokens)

	if _, _, err := core.HandleAttach("tok", CallbackSession{SessionID: "s1"}); err != nil {
		t.Fatalf("initial attach: %v", err)
	}
	_, _, err := core.HandleAttach("tok", CallbackSession{SessionID: "s2"})
	if !errors.Is(err, ErrSessionActive) {
		t.Fatalf("second attach got %v, want ErrSessionActive", err)
	}
	if current := core.Sessions["p1"]; current == nil || current.ID() != "s1" {
		t.Fatalf("incumbent session disturbed: %#v", current)
	}
	if !core.State.Players["p1"].Connected {
		t.Fatal("incumbent should remain connected after a rejected attach")
	}
}

func TestHandleDetachPreservesSeatAndIgnoresStaleSessions(t *testing.T) {
	tokens := TokensFromMap(map[string]string{"p1": "tok"})
	core := NewRoomCore("room_1", game.NewLobby("p1", "Host"), tokens)
	now := time.Now()

	if _, _, err := core.HandleAttach("tok", CallbackSession{SessionID: "s1"}); err != nil {
		t.Fatalf("initial attach: %v", err)
	}

	if changed, err := core.HandleDetach("p1", "wrong-session", now); err != nil || changed {
		t.Fatalf("unrelated detach changed=%v err=%v, want no-op", changed, err)
	}
	changed, err := core.HandleDetach("p1", "s1", now)
	if err != nil || !changed {
		t.Fatalf("detach changed=%v err=%v", changed, err)
	}
	if len(core.Sessions) != 0 {
		t.Fatalf("sessions not cleared: %#v", core.Sessions)
	}
	if core.State.Players["p1"].Connected {
		t.Fatal("detached player still marked connected")
	}

	if changed, err := core.HandleDetach("p1", "s1", now); err != nil || changed {
		t.Fatalf("repeat detach changed=%v err=%v, want no-op", changed, err)
	}
	if _, _, err := core.HandleAttach("tok", CallbackSession{SessionID: "s2"}); err != nil {
		t.Fatalf("reattach after detach: %v", err)
	}
	if changed, err := core.HandleDetach("p1", "s1", now); err != nil || changed {
		t.Fatalf("stale detach changed=%v err=%v, want no-op", changed, err)
	}
	if core.Sessions["p1"].ID() != "s2" || !core.State.Players["p1"].Connected {
		t.Fatal("stale detach disturbed the replacement session")
	}
}
