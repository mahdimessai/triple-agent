package room

import (
	"fmt"
	"testing"

	"tripleagent/server/internal/game"
)

type registryFixture struct {
	t          *testing.T
	registry   *Registry
	room       *Room
	identities []Identity
}

func newRegistryFixture(t *testing.T, playerNames ...string) *registryFixture {
	t.Helper()
	if len(playerNames) == 0 {
		t.Fatal("registry fixture requires at least one player")
	}

	registry := NewRegistry()
	fixture := &registryFixture{t: t, registry: registry}
	t.Cleanup(registry.Close)

	created, err := registry.Create(playerNames[0])
	if err != nil {
		t.Fatalf("create fixture room: %v", err)
	}
	fixture.identities = append(fixture.identities, created)
	fixture.room, _ = registry.Get(created.RoomID)
	if fixture.room == nil {
		t.Fatalf("created fixture room %q is missing", created.RoomID)
	}

	for _, name := range playerNames[1:] {
		joined, err := registry.Join(created.JoinCode, name)
		if err != nil {
			t.Fatalf("join fixture room as %q: %v", name, err)
		}
		fixture.identities = append(fixture.identities, joined)
	}
	return fixture
}

func (f *registryFixture) Identity(index int) Identity {
	f.t.Helper()
	if index < 0 || index >= len(f.identities) {
		f.t.Fatalf("fixture identity index %d out of range", index)
	}
	return f.identities[index]
}

func (f *registryFixture) Snapshot(index int) game.Projection {
	f.t.Helper()
	identity := f.Identity(index)
	projection, err := f.room.Snapshot(identity.PlayerID)
	if err != nil {
		f.t.Fatalf("snapshot fixture player %q: %v", identity.PlayerID, err)
	}
	return projection
}

func (f *registryFixture) Attach(index int, sessionID string, send func(game.Projection) error) {
	f.t.Helper()
	identity := f.Identity(index)
	if err := f.room.Attach(identity.PlayerID, identity.ReconnectToken, sessionID, send, nil); err != nil {
		f.t.Fatalf("attach fixture player %q: %v", identity.PlayerID, err)
	}
}

func fixturePlayerNames(count int) []string {
	names := make([]string, 0, count)
	for index := 1; index <= count; index++ {
		names = append(names, fmt.Sprintf("P%d", index))
	}
	return names
}
