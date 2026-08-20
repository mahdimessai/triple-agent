package game

import "testing"

func TestNewLobbyWithSeedPreservesDeterministicRandomState(t *testing.T) {
	const seed = uint64(0x1234567890abcdef)
	state := NewLobbyWithSeed("p1", "Host", DefaultSettings(), seed)
	if state.RandomState != seed {
		t.Fatalf("random state = %x, want %x", state.RandomState, seed)
	}

	left := state
	right := state
	for i := 0; i < 10; i++ {
		if got, want := nextRandom(&left, 1000), nextRandom(&right, 1000); got != want {
			t.Fatalf("deterministic sequence diverged at %d: got %d want %d", i, got, want)
		}
	}
}
