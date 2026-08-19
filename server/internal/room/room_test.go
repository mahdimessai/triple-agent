package room

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"tripleagent/server/internal/game"
)

func waitUntil(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition did not become true")
}

func TestRegistryCreateJoinAndAuthentication(t *testing.T) {
	fixture := newRegistryFixture(t, "Host", "Guest")
	joined := fixture.Identity(1)
	active := fixture.room
	if err := active.Attach(joined.PlayerID, "wrong", "s1", func(game.Projection) error { return nil }, nil); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("wrong token: got %v", err)
	}
	if err := active.Attach(joined.PlayerID, joined.ReconnectToken, "s1", func(game.Projection) error { return nil }, nil); err != nil {
		t.Fatalf("correct token rejected: %v", err)
	}
	projection, err := active.Snapshot(joined.PlayerID)
	if err != nil {
		t.Fatal(err)
	}
	if !projection.Public.Players[1].Connected {
		t.Fatalf("joined player not connected: %+v", projection.Public.Players)
	}
}

func TestSoleHostLobbyDisconnectRetainsRoomForReconnect(t *testing.T) {
	fixture := newRegistryFixture(t, "Host")
	created := fixture.Identity(0)
	active := fixture.room
	if err := active.Attach(created.PlayerID, created.ReconnectToken, "host-session", func(game.Projection) error { return nil }, nil); err != nil {
		t.Fatal(err)
	}
	active.Detach(created.PlayerID, "host-session")
	projection, err := active.Snapshot(created.PlayerID)
	if err != nil {
		t.Fatal(err)
	}
	if projection.Public.Players[0].Connected {
		t.Fatal("lobby disconnect removed the reconnectable player state")
	}
	if err := active.Attach(created.PlayerID, created.ReconnectToken, "reconnected-session", func(game.Projection) error { return nil }, nil); err != nil {
		t.Fatalf("lobby player could not reconnect: %v", err)
	}
	projection, err = active.Snapshot(created.PlayerID)
	if err != nil {
		t.Fatal(err)
	}
	if !projection.Public.Players[0].Connected {
		t.Fatal("reconnected lobby player was not marked connected")
	}
}

func TestExplicitLeaveRemovesSoleLobbyPlayerAndJoinCode(t *testing.T) {
	fixture := newRegistryFixture(t, "Host")
	created := fixture.Identity(0)
	if err := fixture.room.Leave(created.PlayerID, created.ReconnectToken); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, func() bool {
		_, ok := fixture.registry.Get(created.RoomID)
		return !ok
	})
	if _, err := fixture.registry.Join(created.JoinCode, "Latecomer"); !errors.Is(err, ErrRoomNotFound) {
		t.Fatalf("old join code remained valid: %v", err)
	}
}

func TestStaleDetachDoesNotDisconnectReplacementSession(t *testing.T) {
	state := game.NewLobby("p1", "Host", game.DefaultSettings())
	active := newRoom("room", state, map[string]string{"p1": "token"}, nil)
	defer active.Close()
	var oldClosed atomic.Bool
	if err := active.Attach("p1", "token", "old", func(game.Projection) error { return nil }, func() { oldClosed.Store(true) }); err != nil {
		t.Fatal(err)
	}
	if err := active.Attach("p1", "token", "new", func(game.Projection) error { return nil }, nil); err != nil {
		t.Fatal(err)
	}
	if !oldClosed.Load() {
		t.Fatal("replacement did not close old session")
	}
	active.Detach("p1", "old")
	projection, err := active.Snapshot("p1")
	if err != nil {
		t.Fatal(err)
	}
	if !projection.Public.Players[0].Connected {
		t.Fatal("stale detach disconnected replacement session")
	}
}

func TestCommandRejectsStaleVersionAndStaleSession(t *testing.T) {
	state := game.NewLobby("p1", "Host", game.DefaultSettings())
	active := newRoom("room", state, map[string]string{"p1": "token"}, nil)
	defer active.Close()
	if err := active.Attach("p1", "token", "s1", func(game.Projection) error { return nil }, nil); err != nil {
		t.Fatal(err)
	}
	projection, err := active.Snapshot("p1")
	if err != nil {
		t.Fatal(err)
	}
	if err := active.Command("p1", "old", projection.Public.Version, game.Command{Kind: game.CommandSetReady}); !errors.Is(err, ErrSessionGone) {
		t.Fatalf("stale session got %v", err)
	}
	if err := active.Command("p1", "s1", projection.Public.Version+1, game.Command{Kind: game.CommandSetReady}); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale version got %v", err)
	}
	if err := active.Command("p1", "s1", projection.Public.Version, game.Command{Kind: game.CommandSetReady}); err != nil {
		t.Fatal(err)
	}
	after, err := active.Snapshot("p1")
	if err != nil {
		t.Fatal(err)
	}
	if !after.Public.Players[0].Ready || after.Public.Version != projection.Public.Version+1 {
		t.Fatalf("command did not apply once: before=%d after=%+v", projection.Public.Version, after.Public)
	}
	if err := active.Command("p1", "s1", projection.Public.Version, game.Command{Kind: game.CommandSetReady}); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("second identical expected version should be stale, got %v", err)
	}
}

func TestFailedSenderDisconnectsPlayerAndBroadcastsConvergence(t *testing.T) {
	settings := game.DefaultSettings()
	state := game.NewLobby("p1", "Host", settings)
	var err error
	state, err = game.AddPlayer(state, "p2", "Guest")
	if err != nil {
		t.Fatal(err)
	}
	state, err = game.Connect(state, "p2")
	if err != nil {
		t.Fatal(err)
	}
	state.Phase = game.PhaseDiscussion
	active := newRoom("room", state, map[string]string{"p1": "t1", "p2": "t2"}, nil)
	defer active.Close()

	var mu sync.Mutex
	var hostViews []game.Projection
	if err := active.Attach("p1", "t1", "s1", func(p game.Projection) error {
		mu.Lock()
		hostViews = append(hostViews, p)
		mu.Unlock()
		return nil
	}, nil); err != nil {
		t.Fatal(err)
	}
	calls := 0
	if err := active.Attach("p2", "t2", "s2", func(game.Projection) error {
		calls++
		if calls > 1 {
			return errors.New("send failed")
		}
		return nil
	}, nil); err != nil {
		t.Fatal(err)
	}

	host, err := active.Snapshot("p1")
	if err != nil {
		t.Fatal(err)
	}
	if err := active.Command("p1", "s1", host.Public.Version, game.Command{Kind: game.CommandAdvanceDiscussion}); err != nil {
		t.Fatal(err)
	}
	final, err := active.Snapshot("p1")
	if err != nil {
		t.Fatal(err)
	}
	var guestConnected bool
	for _, player := range final.Public.Players {
		if player.ID == "p2" {
			guestConnected = player.Connected
		}
	}
	if guestConnected {
		t.Fatal("failed sender remained connected")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(hostViews) < 2 {
		t.Fatalf("host did not receive convergence broadcasts: %d views", len(hostViews))
	}
}

func TestInGameDisconnectRetainsTokenForReconnect(t *testing.T) {
	state := game.NewLobby("p1", "Host", game.DefaultSettings())
	state.Phase = game.PhaseDiscussion
	active := newRoom("room", state, map[string]string{"p1": "token"}, nil)
	defer active.Close()
	if err := active.Attach("p1", "token", "s1", func(game.Projection) error { return nil }, nil); err != nil {
		t.Fatal(err)
	}
	active.Detach("p1", "s1")
	projection, err := active.Snapshot("p1")
	if err != nil {
		t.Fatalf("in-game player was removed: %v", err)
	}
	if projection.Public.Players[0].Connected {
		t.Fatal("detached in-game player still connected")
	}
	if err := active.Attach("p1", "token", "s2", func(game.Projection) error { return nil }, nil); err != nil {
		t.Fatalf("reconnect token was revoked in game: %v", err)
	}
	projection, _ = active.Snapshot("p1")
	if !projection.Public.Players[0].Connected {
		t.Fatal("reconnected player not marked connected")
	}
}

func TestRoomExpiryRunsRegistryCleanup(t *testing.T) {
	state := game.NewLobby("p1", "Host", game.DefaultSettings())
	closed := make(chan struct{})
	active := newRoomWithLifetimes("room", state, map[string]string{"p1": "token"}, func(*Room) { close(closed) }, 5*time.Millisecond, 5*time.Millisecond)
	defer active.Close()
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("room did not expire")
	}
}

func TestConcurrentCloseIsSafe(t *testing.T) {
	state := game.NewLobby("p1", "Host", game.DefaultSettings())
	active := newRoom("room", state, map[string]string{"p1": "token"}, nil)
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			active.Close()
		}()
	}
	wg.Wait()
	waitUntil(t, func() bool {
		_, err := active.Snapshot("p1")
		return errors.Is(err, ErrClosed)
	})
}

func TestJoinCannotCreatePlayerWithoutCredential(t *testing.T) {
	state := game.NewLobby("p1", "Host", game.DefaultSettings())
	active := newRoom("room", state, map[string]string{"p1": "token"}, nil)
	defer active.Close()
	if err := active.Join("p2", "Guest", ""); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("join error = %v, want ErrUnauthorized", err)
	}
	if _, err := active.Snapshot("p2"); !errors.Is(err, game.ErrPlayerNotInRoom) {
		t.Fatalf("credential-less player was committed: %v", err)
	}
}
