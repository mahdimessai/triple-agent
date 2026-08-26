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
