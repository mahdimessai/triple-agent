package room

import (
	"errors"
	"sync"
	"testing"
	"time"

	"tripleagent/server/internal/domain"
)

func TestStaleDetachCannotDisconnectReplacementSession(t *testing.T) {
	state := domain.NewLobby("room", "host", "Host", domain.DefaultRoomSettings())
	active := newRoom(state, time.Hour, time.Hour, nil)
	defer active.Close()

	oldClosed := make(chan struct{})
	if err := active.Attach("host", "old", func(domain.Projection) error { return nil }, func() { close(oldClosed) }); err != nil {
		t.Fatal(err)
	}
	if err := active.Attach("host", "new", func(domain.Projection) error { return nil }, func() {}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-oldClosed:
	case <-time.After(time.Second):
		t.Fatal("replacement did not close the old session")
	}

	active.Detach("host", "old")
	projection, err := active.Snapshot("host")
	if err != nil {
		t.Fatal(err)
	}
	if !projection.Public.Players[0].Connected {
		t.Fatal("stale detach marked the replacement session offline")
	}
	if _, _, err := active.SubmitForSession("old", domain.Command{ActorID: "host", ExpectedVersion: projection.Public.Version, Kind: domain.CommandSetReady}); err == nil {
		t.Fatal("stale session submitted a command after replacement")
	}
	if _, _, err := active.SubmitForSession("new", domain.Command{ActorID: "host", ExpectedVersion: projection.Public.Version, Kind: domain.CommandSetReady}); err != nil {
		t.Fatal("current session could not submit a command:", err)
	}
}

func TestFailedProjectionDeliveryAppliesDisconnectPolicy(t *testing.T) {
	state := domain.NewLobby("room", "host", "Host", domain.DefaultRoomSettings())
	if err := state.AddPlayer("player", "Player"); err != nil {
		t.Fatal(err)
	}
	active := newRoom(state, time.Hour, time.Hour, nil)
	defer active.Close()
	if err := active.Attach("host", "host-session", func(domain.Projection) error { return nil }, func() {}); err != nil {
		t.Fatal(err)
	}
	deliveryCount := 0
	if err := active.Attach("player", "player-session", func(domain.Projection) error {
		deliveryCount++
		if deliveryCount > 1 {
			return errors.New("slow client")
		}
		return nil
	}, func() {}); err != nil {
		t.Fatal(err)
	}
	if err := active.Submit(domain.Command{ActorID: "host", ExpectedVersion: 0, Kind: domain.CommandSetReady}); err != nil {
		t.Fatal(err)
	}
	projection, err := active.Snapshot("host")
	if err != nil {
		t.Fatal(err)
	}
	for _, player := range projection.Public.Players {
		if player.ID == "player" && player.Connected {
			t.Fatal("failed projection delivery left the player connected")
		}
	}
}

func TestConcurrentCloseIsSafe(t *testing.T) {
	state := domain.NewLobby("room", "host", "Host", domain.DefaultRoomSettings())
	active := newRoom(state, time.Hour, time.Hour, nil)
	var group sync.WaitGroup
	for i := 0; i < 32; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			active.Close()
		}()
	}
	group.Wait()
}

func TestDuplicateRequestReplaysProjectionWithoutReapplying(t *testing.T) {
	state := domain.NewLobby("room", "host", "Host", domain.DefaultRoomSettings())
	active := newRoom(state, time.Hour, time.Hour, nil)
	defer active.Close()

	command := domain.Command{RequestID: "ready-once", ActorID: "host", ExpectedVersion: 0, Kind: domain.CommandSetReady}
	if err := active.Submit(command); err != nil {
		t.Fatal(err)
	}
	if err := active.Submit(domain.Command{ActorID: "host", ExpectedVersion: 1, Kind: domain.CommandSetDiscussionTimer, DiscussionTimerEnabled: false}); err != nil {
		t.Fatal(err)
	}
	replayed, wasReplay, err := active.SubmitForSession("", command)
	if err != nil {
		t.Fatal(err)
	}
	if !wasReplay || replayed.Public.Version != 2 || replayed.Private.CanSubmit {
		t.Fatalf("replay = %#v, wasReplay = %t", replayed, wasReplay)
	}
	snapshot, err := active.Snapshot("host")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Public.Version != 2 {
		t.Fatalf("duplicate request advanced version to %d", snapshot.Public.Version)
	}
}

func TestLateJoinRefreshesRoomActorLifetime(t *testing.T) {
	state := domain.NewLobby("room_lifetime", "host", "Host", domain.DefaultRoomSettings())
	active := newRoom(state, 300*time.Millisecond, time.Second, nil)
	defer active.Close()
	time.Sleep(180 * time.Millisecond)
	if err := active.AddPlayer("p2", "Agent B"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(180 * time.Millisecond)
	if _, err := active.Snapshot("host"); err != nil {
		t.Fatal("late join did not refresh room lifetime:", err)
	}
	time.Sleep(180 * time.Millisecond)
	if _, err := active.Snapshot("host"); err == nil {
		t.Fatal("room remained active beyond refreshed lifetime")
	}
}

func TestDiscussionDeadlineAdvancesRoomWithoutClientCommand(t *testing.T) {
	state := domain.NewLobby("room", "host", "Host", domain.DefaultRoomSettings())
	deadline := time.Now().Add(30 * time.Millisecond)
	state.Phase = domain.PhaseDiscussion
	state.DiscussionDeadline = &deadline
	active := newRoom(state, time.Hour, time.Hour, nil)
	defer active.Close()

	deadlineAt := time.Now().Add(time.Second)
	for time.Now().Before(deadlineAt) {
		projection, err := active.Snapshot("host")
		if err != nil {
			t.Fatal(err)
		}
		if projection.Public.Phase == domain.PhaseVoteInput {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("discussion deadline did not advance the room to voting")
}

func TestManagerCreateClosesOldRoomOnCollision(t *testing.T) {
	mgr := NewManager()
	state1 := domain.NewLobby("collision_room", "host", "Host", domain.DefaultRoomSettings())
	room1 := mgr.Create(state1)

	// Verify room1 is open
	if err := room1.AddPlayer("p2", "Agent B"); err != nil {
		t.Fatal(err)
	}

	state2 := domain.NewLobby("collision_room", "host2", "Host 2", domain.DefaultRoomSettings())
	room2 := mgr.Create(state2)
	defer room2.Close()

	// room1 should now be closed
	if err := room1.AddPlayer("p3", "Agent C"); err == nil {
		t.Fatal("expected old room to be closed on collision, but it accepted add_player")
	}

	// room2 should be active in manager
	active, ok := mgr.Get("collision_room")
	if !ok || active != room2 {
		t.Fatal("expected room2 to be active in manager")
	}
}

func TestRematchPreservesFullRoomLifetime(t *testing.T) {
	state := domain.NewLobby("room_rematch", "host", "Host", domain.DefaultRoomSettings())
	state.Phase = domain.PhaseEnd
	state.Winner = domain.FactionService
	state.Players["host"] = domain.PlayerState{ID: "host", Name: "Host", Ready: true}

	expired := false
	active := newRoom(state, 500*time.Millisecond, 20*time.Millisecond, func() {
		expired = true
	})
	defer active.Close()

	// Execute rematch - should reset to lifetime (500ms), not endedAfter (20ms)
	rematchCmd := domain.Command{ActorID: "host", ExpectedVersion: 0, Kind: domain.CommandRematch}
	if err := active.Submit(rematchCmd); err != nil {
		t.Fatal(err)
	}

	// Wait 50ms (longer than 20ms endedAfter)
	time.Sleep(50 * time.Millisecond)

	snapshot, err := active.Snapshot("host")
	if err != nil {
		t.Fatal("room was closed prematurely after rematch:", err)
	}
	if snapshot.Public.Phase != domain.PhaseLobby {
		t.Fatalf("phase after rematch = %s, want LOBBY", snapshot.Public.Phase)
	}
	if expired {
		t.Fatal("room expired prematurely within endedAfter window instead of full room lifetime")
	}
}
