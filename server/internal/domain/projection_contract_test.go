package domain

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestProjectKeepsPrivateIdentityAndOperationDataOutOfPublicProjection(t *testing.T) {
	state := NewLobby("room", "host", "Host", DefaultRoomSettings())
	if err := state.AddPlayer("target", "Target"); err != nil {
		t.Fatal(err)
	}
	state.Phase = PhaseOperationResult
	state.Players["host"] = PlayerState{
		ID: "host", Name: "Host", InitialFaction: FactionVirus, Faction: FactionVirus, Role: RoleNormalRed,
	}
	state.Players["target"] = PlayerState{
		ID: "target", Name: "Target", InitialFaction: FactionService, Faction: FactionService, Role: RoleNormalBlue,
	}
	state.Operation = &OperationState{
		Kind: "Swap", ActivePlayerID: "host", TargetPlayerIDs: []string{"target"},
		PrivateResults: map[string]OperationResult{
			"host": {Code: "FACTION_REVEALED", TargetPlayerID: "target", TargetFaction: FactionService, Message: "private result"},
		},
	}

	projection := Project(state, "host")
	if projection.Private.PlayerID != "host" || projection.Private.Faction != FactionVirus || projection.Private.Role != RoleNormalRed {
		t.Fatalf("private identity projection = %#v", projection.Private)
	}
	if projection.Private.OperationResult == nil || projection.Private.OperationResult.TargetPlayerID != "target" {
		t.Fatalf("private operation result = %#v", projection.Private.OperationResult)
	}
	if projection.Public.Operation == nil || projection.Public.Operation.TargetCount != 1 || projection.Public.Operation.ActivePlayerID != "host" {
		t.Fatalf("public operation contract = %#v", projection.Public.Operation)
	}

	encoded, err := json.Marshal(projection.Public)
	if err != nil {
		t.Fatal(err)
	}
	for _, secretField := range []string{
		`"initial_faction"`, `"faction"`, `"role"`, `"target_player_id"`, `"target_player_ids"`,
		`"operation_result"`, `"legal_target_ids"`,
	} {
		if bytes.Contains(encoded, []byte(secretField)) {
			t.Fatalf("public projection leaked %s: %s", secretField, encoded)
		}
	}

	bystander := Project(state, "target")
	if bystander.Private.OperationResult != nil {
		t.Fatalf("private result leaked to bystander: %#v", bystander.Private.OperationResult)
	}
}
