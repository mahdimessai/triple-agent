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

func TestLeaderboardPublishesObjectiveAndWinReason(t *testing.T) {
	state := GameState{
		PlayerOrder: []string{"p_scapegoat", "p_grudge", "p_infatuation", "p_virus", "p_service"},
		Players: map[string]PlayerState{
			"p_scapegoat": {
				ID: "p_scapegoat", Name: "Alice", Faction: FactionService,
				ObjectiveKind: "IMPRISON_SELF", Statuses: []string{"STRAIN"},
			},
			"p_grudge": {
				ID: "p_grudge", Name: "Bob", Faction: FactionVirus,
				ObjectiveKind: "IMPRISON_TARGET", ObjectiveTarget: "p_scapegoat", Statuses: []string{"GRUDGE"},
			},
			"p_infatuation": {
				ID: "p_infatuation", Name: "Charlie", Faction: FactionService,
				ObjectiveKind: "TARGET_WINS", ObjectiveTarget: "p_grudge", Statuses: []string{"INFATUATION"},
			},
			"p_virus": {
				ID: "p_virus", Name: "Dave", Faction: FactionVirus,
			},
			"p_service": {
				ID: "p_service", Name: "Eve", Faction: FactionService,
			},
		},
		Vote: VoteState{
			ImprisonedPlayerID: "p_scapegoat",
			Totals:             map[string]int{"p_scapegoat": 3, "p_virus": 1},
		},
		Winner: FactionNone,
	}

	entries := buildLeaderboard(state)
	if len(entries) != 5 {
		t.Fatalf("leaderboard length = %d, want 5", len(entries))
	}

	// Scapegoat was imprisoned -> WINNER
	if entries[0].Result != "WINNER" || entries[0].ObjectiveKind != "IMPRISON_SELF" || entries[0].WinReason != "Imprisoned and achieved solo victory" {
		t.Fatalf("scapegoat entry = %#v", entries[0])
	}

	// Grudge had Scapegoat as target -> WINNER because target was imprisoned
	if entries[1].Result != "WINNER" || entries[1].ObjectiveKind != "IMPRISON_TARGET" || entries[1].ObjectiveTargetName != "Alice" || entries[1].WinReason != "Succeeded in getting Alice imprisoned" {
		t.Fatalf("grudge entry = %#v", entries[1])
	}

	// Infatuation targeted Grudge (who won) -> WINNER
	if entries[2].Result != "WINNER" || entries[2].ObjectiveKind != "TARGET_WINS" || entries[2].ObjectiveTargetName != "Bob" || entries[2].WinReason != "Felt the love: Bob won the match" {
		t.Fatalf("infatuation entry = %#v", entries[2])
	}

	// Standard virus player -> LOSER (Scapegoat took the round)
	if entries[3].Result != "LOSER" || entries[3].WinReason != "Lost: Operation Scapegoat took the round" {
		t.Fatalf("virus entry = %#v", entries[3])
	}

	// Standard service player -> LOSER (Scapegoat took the round)
	if entries[4].Result != "LOSER" || entries[4].WinReason != "Lost: Operation Scapegoat took the round" {
		t.Fatalf("service entry = %#v", entries[4])
	}
}
