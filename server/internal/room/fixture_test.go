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
	identities []PlayerCredentials
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
	fixture.room, _ = registry.GetByCode(created.JoinCode)
	if fixture.room == nil {
		t.Fatalf("created fixture room for join code %q is missing", created.JoinCode)
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

func (f *registryFixture) Identity(index int) PlayerCredentials {
	f.t.Helper()
	if index < 0 || index >= len(f.identities) {
		f.t.Fatalf("fixture identity index %d out of range", index)
	}
	return f.identities[index]
}

func (f *registryFixture) PlayerID(index int) string {
	f.t.Helper()
	identity := f.Identity(index)
	playerID, err := f.room.PlayerIDForToken(identity.ReconnectToken)
	if err != nil {
		f.t.Fatalf("resolve fixture player: %v", err)
	}
	return playerID
}

func (f *registryFixture) Snapshot(index int) game.Projection {
	f.t.Helper()
	playerID := f.PlayerID(index)
	projection, err := f.room.Snapshot(playerID)
	if err != nil {
		f.t.Fatalf("snapshot fixture player %q: %v", playerID, err)
	}
	return projection
}

func (f *registryFixture) Attach(index int, sessionID string, send func(game.Projection) error) {
	f.t.Helper()
	identity := f.Identity(index)
	playerID := f.PlayerID(index)
	if err := f.room.Attach(playerID, identity.ReconnectToken, sessionID, send, nil); err != nil {
		f.t.Fatalf("attach fixture player %q: %v", playerID, err)
	}
}

func fixturePlayerNames(count int) []string {
	names := make([]string, 0, count)
	for index := 1; index <= count; index++ {
		names = append(names, fmt.Sprintf("P%d", index))
	}
	return names
}
