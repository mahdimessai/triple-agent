package domain

import "testing"

func TestLeaderboardPublishesRoleAndDefection(t *testing.T) {
	state := GameState{
		PlayerOrder: []string{"virus", "service"},
		Players: map[string]PlayerState{
			"virus": {
				ID: "virus", Name: "Virus Player", Faction: FactionVirus,
				Role: RoleFakeRed, Statuses: []string{"BLUE_DEFECTOR"},
			},
			"service": {
				ID: "service", Name: "Service Player", Faction: FactionService,
				Role: RoleLoyalBlue,
			},
		},
		Vote:   VoteState{Totals: map[string]int{"virus": 2, "service": 1}},
		Winner: FactionService,
	}

	entries := buildLeaderboard(state)
	if len(entries) != 2 {
		t.Fatalf("leaderboard length = %d, want 2", len(entries))
	}
	if entries[0].Role != RoleFakeRed || entries[0].Defection != "BLUE_DEFECTOR" {
		t.Fatalf("defector metadata = %#v, want role %q and BLUE_DEFECTOR", entries[0], RoleFakeRed)
	}
	if entries[1].Role != RoleLoyalBlue || entries[1].Defection != "" {
		t.Fatalf("role metadata = %#v, want role %q and no defection", entries[1], RoleLoyalBlue)
	}
}

func TestPublicProjectionRevealsLeaderboardMetadataOnlyInResults(t *testing.T) {
	state := GameState{
		Phase:       PhaseDiscussion,
		PlayerOrder: []string{"player"},
		Players: map[string]PlayerState{
			"player": {
				ID: "player", Name: "Player", Faction: FactionVirus,
				Role: RoleFakeRed, Statuses: []string{"BLUE_DEFECTOR"},
			},
		},
	}

	if public := PublicProjectionFor(state); len(public.Leaderboard) != 0 {
		t.Fatalf("pre-results leaderboard length = %d, want 0", len(public.Leaderboard))
	}

	state.Phase = PhaseLeaderboard
	public := PublicProjectionFor(state)
	if len(public.Leaderboard) != 1 {
		t.Fatalf("results leaderboard length = %d, want 1", len(public.Leaderboard))
	}
	entry := public.Leaderboard[0]
	if entry.Role != RoleFakeRed || entry.Defection != "BLUE_DEFECTOR" {
		t.Fatalf("results metadata = %#v, want role %q and BLUE_DEFECTOR", entry, RoleFakeRed)
	}
}
