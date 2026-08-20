package game

import "testing"

func TestProjectionNeverExposesOtherPlayerSecretState(t *testing.T) {
	state := testLobby(t, 3)
	p1 := state.Players["p1"]
	p1.InitialFaction, p1.Faction, p1.Role = FactionVirus, FactionVirus, RoleLoyalRed
	state.Players["p1"] = p1
	p2 := state.Players["p2"]
	p2.InitialFaction, p2.Faction, p2.Role = FactionService, FactionService, RoleFakeRed
	p2.ObjectiveKind = "IMPRISON_TARGET"
	p2.ObjectiveTarget = "p3"
	state.Players["p2"] = p2

	view := Project("room", state, "p1")
	if view.Private.PlayerID != "p1" {
		t.Fatalf("private projection belongs to %q", view.Private.PlayerID)
	}
	for _, publicPlayer := range view.Public.Players {
		if publicPlayer.ID == "p2" {
			// The public type has no faction, role, or objective fields by construction.
			if publicPlayer.Name != p2.Name {
				t.Fatal("public player malformed")
			}
		}
	}
	if view.Private.Role == p2.Role && p1.Role != p2.Role {
		t.Fatal("another player's role leaked into private projection")
	}
}

func TestResultsProjectionRevealsInformationOnlyAtItsPhase(t *testing.T) {
	state := testLobby(t, 3)
	state.Vote.Totals = map[string]int{"p1": 1, "p2": 2}
	state.Vote.ImprisonedPlayerID = "p2"
	state.Winner = FactionService
	p2 := state.Players["p2"]
	p2.InitialFaction = FactionVirus
	p2.Faction = FactionVirus
	p2.Role = RoleNormalRed
	state.Players["p2"] = p2

	tests := []struct {
		phase       Phase
		votes       bool
		imprisoned  bool
		agency      bool
		winner      bool
		leaderboard bool
	}{
		{phase: PhaseResultsIntro},
		{phase: PhaseVoteResults, votes: true},
		{phase: PhaseImprisonment, votes: true, imprisoned: true},
		{phase: PhaseAgencyReveal, votes: true, imprisoned: true, agency: true},
		{phase: PhaseOutcomeReveal, votes: true, imprisoned: true, agency: true, winner: true},
		{phase: PhaseLeaderboard, votes: true, imprisoned: true, agency: true, winner: true, leaderboard: true},
		{phase: PhaseOutOfLoop, votes: true, imprisoned: true, agency: true, winner: true, leaderboard: true},
		{phase: PhaseEnd, votes: true, imprisoned: true, agency: true, winner: true, leaderboard: true},
	}

	for _, tc := range tests {
		t.Run(string(tc.phase), func(t *testing.T) {
			state.Phase = tc.phase
			view := PublicProjectionFor("room", state)

			if got := len(view.VoteTotals) > 0; got != tc.votes {
				t.Fatalf("vote totals visible=%v, want %v: %+v", got, tc.votes, view.VoteTotals)
			}
			if got := view.ImprisonedPlayerID != ""; got != tc.imprisoned {
				t.Fatalf("imprisoned player visible=%v, want %v: %q", got, tc.imprisoned, view.ImprisonedPlayerID)
			}
			if got := view.RevealedFaction != ""; got != tc.agency {
				t.Fatalf("agency visible=%v, want %v: %q", got, tc.agency, view.RevealedFaction)
			}
			if got := view.Winner != ""; got != tc.winner {
				t.Fatalf("winner visible=%v, want %v: %q", got, tc.winner, view.Winner)
			}
			if got := len(view.Leaderboard) > 0; got != tc.leaderboard {
				t.Fatalf("leaderboard visible=%v, want %v: %+v", got, tc.leaderboard, view.Leaderboard)
			}
		})
	}
}
