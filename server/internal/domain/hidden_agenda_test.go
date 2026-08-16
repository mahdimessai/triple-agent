package domain

import (
	"testing"
	"time"
)

// Hidden Agenda is one operation with several possible contents, so it must win
// the draw about as often as any single named operation, not once per envelope.
func TestHiddenAgendaDrawsAsOneUnit(t *testing.T) {
	state := &GameState{}
	eligible := []OperationResolver{
		anonymousTipResolver{},
		detectorResolver{},
		strainResolver{},
		grudgeResolver{},
		infatuationResolver{},
		flipResolver{},
		hiddenTipResolver{},
	}

	const draws = 9000
	counts := make(map[string]int)
	for i := 0; i < draws; i++ {
		chosen := drawOperation(state, eligible)
		if chosen == nil {
			t.Fatal("drawOperation returned nil for a non-empty pool")
		}
		counts[chosen.Definition().ID]++
	}

	hidden := 0
	for _, id := range []string{"Strain", "Grudge", "Infatuation", "Flip", "HiddenOneRandom"} {
		if counts[id] == 0 {
			t.Errorf("envelope %s was never dealt", id)
		}
		hidden += counts[id]
	}

	// Two named operations plus the cover means three slots, so the cover should
	// land near a third of the draws rather than the five sevenths it would get
	// if every envelope held its own slot.
	share := float64(hidden) / float64(draws)
	if share < 0.28 || share > 0.39 {
		t.Fatalf("hidden agenda share = %.3f, want about one third (counts %#v)", share, counts)
	}
}

// A player can only ever open one envelope, because the members share a
// category and the category is what the deal records.
func TestHiddenAgendaDealtOncePerPlayer(t *testing.T) {
	state := fivePlayerMatch(t)
	recordDealtOperation(&state, "p1", grudgeResolver{}.Definition())

	for i := 0; i < 50; i++ {
		resolver, err := randomEligibleOperation(&state, "p1", 1)
		if err != nil {
			t.Fatal(err)
		}
		if resolver.Definition().Hidden {
			t.Fatalf("player drew a second hidden agenda: %s", resolver.Definition().ID)
		}
		// Roll the recipient back to "one hidden agenda, nothing else" so each
		// pass is a fresh draw against the same history instead of walking the
		// pool down to the repeat-allowed fallback.
		player := state.Players["p1"]
		player.DealtOperations = []string{grudgeResolver{}.Definition().ID}
		player.DealtCategories = []int{grudgeResolver{}.Definition().Category}
		state.Players["p1"] = player
	}
}

// Requesting the cover asks for Hidden Agenda, not for a specific envelope, so
// the server still decides which one the recipient opens.
func TestStartMatchWithHiddenAgendaPicksAnEnvelope(t *testing.T) {
	state := fivePlayerMatch(t)
	resolver, err := operationForStart(&state, HiddenAgendaKind)
	if err != nil {
		t.Fatal(err)
	}
	if !resolver.Definition().Hidden {
		t.Fatalf("planned operation = %s, want a hidden agenda envelope", resolver.Definition().ID)
	}
}

// The host toggles the cover once and every envelope follows it.
func TestSetHiddenAgendaEnabledTogglesTheWholeGroup(t *testing.T) {
	state := NewLobby("room_test", "p1", "Agent A", DefaultRoomSettings())
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	transition, err := Apply(state, Command{ActorID: "p1", ExpectedVersion: state.Version, Kind: CommandSetOperationEnabled, OperationKind: HiddenAgendaKind, OperationEnabled: false}, now)
	if err != nil {
		t.Fatal(err)
	}
	state = transition.State
	for _, id := range HiddenAgendaMemberIDs() {
		if state.Settings.EnabledOperations[id] {
			t.Fatalf("envelope %s stayed enabled after the cover was switched off", id)
		}
	}
	for _, id := range []string{"Detector", "OneRandom", "OneOfTwo", "TwoFriends"} {
		if !state.Settings.EnabledOperations[id] {
			t.Fatalf("named operation %s was switched off with the cover", id)
		}
	}

	transition, err = Apply(state, Command{ActorID: "p1", ExpectedVersion: state.Version, Kind: CommandSetOperationEnabled, OperationKind: HiddenAgendaKind, OperationEnabled: true}, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range HiddenAgendaMemberIDs() {
		if !transition.State.Settings.EnabledOperations[id] {
			t.Fatalf("envelope %s stayed disabled after the cover was switched back on", id)
		}
	}
}

// Switching the cover off may not empty the deck.
func TestSetHiddenAgendaEnabledKeepsOneOperationInThePool(t *testing.T) {
	state := NewLobby("room_test", "p1", "Agent A", DefaultRoomSettings())
	pool := cloneOperationPool(state.Settings.EnabledOperations)
	for id := range pool {
		pool[id] = false
	}
	pool["Grudge"] = true
	state.Settings.EnabledOperations = pool
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	if _, err := Apply(state, Command{ActorID: "p1", ExpectedVersion: state.Version, Kind: CommandSetOperationEnabled, OperationKind: HiddenAgendaKind, OperationEnabled: false}, now); err != ErrNotAllowed {
		t.Fatalf("error = %v, want ErrNotAllowed", err)
	}
}
