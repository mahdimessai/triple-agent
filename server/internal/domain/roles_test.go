package domain

import "testing"

func lobbyWithPlayers(t *testing.T, count int) GameState {
	t.Helper()
	state := NewLobby("room_roles", "p1", "Agent A", DefaultRoomSettings())
	for i := 2; i <= count; i++ {
		id := string("p" + string(rune('0'+i)))
		if err := state.AddPlayer(id, "Agent "+string(rune('A'+i-1))); err != nil {
			t.Fatal(err)
		}
	}
	return state
}

func setRole(state *GameState, id string, role RoleKind, faction Faction) {
	player := state.Players[id]
	player.Role = role
	player.Faction = faction
	player.InitialFaction = faction
	switch role {
	case RoleLyingRed:
		service := FactionService
		player.ApparentFaction = &service
	case RoleLyingBlue:
		virus := FactionVirus
		player.ApparentFaction = &virus
	}
	state.Players[id] = player
}

func TestLyingRolesInvertEveryCheck(t *testing.T) {
	state := lobbyWithPlayers(t, 5)
	assignRoles(&state)
	setRole(&state, "p1", RoleLyingRed, FactionVirus)
	setRole(&state, "p2", RoleLyingBlue, FactionService)

	if got := checkFaction(state.Players["p1"]); got != FactionService {
		t.Fatalf("deep cover agent checks as %s, want SERVICE", got)
	}
	if got := checkFaction(state.Players["p2"]); got != FactionVirus {
		t.Fatalf("suspicious agent checks as %s, want VIRUS", got)
	}
	// The lie is only skin deep: scoring still uses the real agency.
	if state.Players["p1"].Faction != FactionVirus || state.Players["p2"].Faction != FactionService {
		t.Fatalf("a lying role changed the real agency: %#v %#v", state.Players["p1"], state.Players["p2"])
	}
}

func TestLyingRolesAlsoRewriteTheStartingAgency(t *testing.T) {
	state := lobbyWithPlayers(t, 5)
	assignRoles(&state)
	setRole(&state, "p1", RoleLyingRed, FactionVirus)
	setRole(&state, "p2", RoleLyingBlue, FactionService)

	// Old Photographs digs into the past, so it has to meet the same lie the
	// live checks do, or it becomes a way to see straight through both roles.
	if got := checkInitialFaction(state.Players["p1"]); got != FactionService {
		t.Fatalf("deep cover agent's starting agency reads as %s, want SERVICE", got)
	}
	if got := checkInitialFaction(state.Players["p2"]); got != FactionVirus {
		t.Fatalf("suspicious agent's starting agency reads as %s, want VIRUS", got)
	}
	if state.Players["p1"].InitialFaction != FactionVirus || state.Players["p2"].InitialFaction != FactionService {
		t.Fatal("a lying role changed the real starting agency")
	}

	// The photograph record uses the same apparent starting agency as every
	// other check, so a matching apparent pair can be shown together.
	setRole(&state, "p3", RoleNormalBlue, FactionService)
	if checkInitialFaction(state.Players["p1"]) != checkInitialFaction(state.Players["p3"]) {
		t.Fatal("old photographs failed to pair players with the same visible starting agency")
	}
}

func TestOldPhotographsAlwaysSelectsTwoPlayersFromOneStartingAgency(t *testing.T) {
	state := lobbyWithPlayers(t, 5)
	setRole(&state, "p1", RoleNormalBlue, FactionService)
	setRole(&state, "p2", RoleNormalRed, FactionVirus)
	setRole(&state, "p3", RoleNormalRed, FactionVirus)
	setRole(&state, "p4", RoleNormalBlue, FactionService)
	setRole(&state, "p5", RoleNormalBlue, FactionService)
	state.ActivePlayerID = "p1"

	for seed := uint64(1); seed <= 40; seed++ {
		state.RandomState = seed
		state.Operation = newOperationState(&state, twoFriendsResolver{}.Definition())
		if err := (twoFriendsResolver{}).Resolve(&state, Command{}); err != nil {
			t.Fatal(err)
		}
		result := state.Operation.PrivateResults[state.ActivePlayerID]
		if result.Code != "SAME_INITIAL_AGENCY" {
			t.Fatalf("seed %d result code = %q, want SAME_INITIAL_AGENCY", seed, result.Code)
		}
		if len(result.TargetPlayerIDs) != 2 || result.TargetPlayerIDs[0] == state.ActivePlayerID || result.TargetPlayerIDs[1] == state.ActivePlayerID {
			t.Fatalf("seed %d targets = %#v, want two other players", seed, result.TargetPlayerIDs)
		}
		if checkInitialFaction(state.Players[result.TargetPlayerIDs[0]]) != checkInitialFaction(state.Players[result.TargetPlayerIDs[1]]) {
			t.Fatalf("seed %d targets have different starting agencies: %#v", seed, result.TargetPlayerIDs)
		}
	}
}

func TestLoyalistCancelsEveryFactionChange(t *testing.T) {
	cases := []struct {
		name    string
		role    RoleKind
		faction Faction
		run     func(t *testing.T, state *GameState, loyalID string)
	}{
		{name: "Flip", role: RoleLoyalRed, faction: FactionVirus, run: func(t *testing.T, state *GameState, loyalID string) {
			state.ActivePlayerID = loyalID
			state.Operation = newOperationState(state, flipResolver{}.Definition())
			if err := (flipResolver{}).Resolve(state, Command{}); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "Undercover", role: RoleLoyalBlue, faction: FactionService, run: func(t *testing.T, state *GameState, loyalID string) {
			virusID := string("p2")
			setRole(state, virusID, RoleNormalRed, FactionVirus)
			state.ActivePlayerID = loyalID
			state.Operation = newOperationState(state, undercoverResolver{}.Definition())
			if err := (undercoverResolver{}).Resolve(state, Command{TargetIDs: []string{virusID}}); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "Injection", role: RoleLoyalBlue, faction: FactionService, run: func(t *testing.T, state *GameState, loyalID string) {
			virusID := string("p2")
			setRole(state, virusID, RoleNormalRed, FactionVirus)
			state.ActivePlayerID = virusID
			state.Operation = newOperationState(state, recruitmentResolver{}.Definition())
			if err := (recruitmentResolver{}).Resolve(state, Command{TargetIDs: []string{loyalID}}); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "Swap", role: RoleLoyalBlue, faction: FactionService, run: func(t *testing.T, state *GameState, loyalID string) {
			virusID := string("p2")
			setRole(state, virusID, RoleNormalRed, FactionVirus)
			state.ActivePlayerID = virusID
			state.Operation = newOperationState(state, spyTransferResolver{}.Definition())
			if err := (spyTransferResolver{}).Resolve(state, Command{TargetIDs: []string{loyalID}}); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "Defect", role: RoleLoyalRed, faction: FactionVirus, run: func(t *testing.T, state *GameState, loyalID string) {
			state.ActivePlayerID = loyalID
			state.Operation = newOperationState(state, defectorResolver{}.Definition())
			if err := (defectorResolver{}).Resolve(state, Command{Choice: "DEFECT"}); err != nil {
				t.Fatal(err)
			}
		}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			state := lobbyWithPlayers(t, 5)
			assignRoles(&state)
			loyalID := string("p1")
			setRole(&state, loyalID, testCase.role, testCase.faction)
			testCase.run(t, &state, loyalID)
			if got := state.Players[loyalID].Faction; got != testCase.faction {
				t.Fatalf("%s moved a loyalist to %s, want %s", testCase.name, got, testCase.faction)
			}
		})
	}
}

func TestSwapStillMovesTheNonLoyalSideOfTheExchange(t *testing.T) {
	state := lobbyWithPlayers(t, 5)
	assignRoles(&state)
	setRole(&state, "p1", RoleLoyalBlue, FactionService)
	setRole(&state, "p2", RoleNormalRed, FactionVirus)
	state.ActivePlayerID = "p2"
	state.Operation = newOperationState(&state, spyTransferResolver{}.Definition())
	if err := (spyTransferResolver{}).Resolve(&state, Command{TargetIDs: []string{"p1"}}); err != nil {
		t.Fatal(err)
	}
	if state.Players["p2"].Faction != FactionService {
		t.Fatalf("the non-loyal swapper kept %s, want SERVICE", state.Players["p2"].Faction)
	}
	if state.Players["p1"].Faction != FactionService {
		t.Fatalf("the loyalist moved to %s, want SERVICE", state.Players["p1"].Faction)
	}
}

func TestVirusRosterHidesTheRogueAndPlantsTheTripleAgent(t *testing.T) {
	state := lobbyWithPlayers(t, 5)
	assignRoles(&state)
	setRole(&state, "p1", RoleNormalRed, FactionVirus)
	setRole(&state, "p2", RoleFakeBlue, FactionVirus)
	setRole(&state, "p3", RoleFakeRed, FactionService)
	setRole(&state, "p4", RoleNormalBlue, FactionService)
	setRole(&state, "p5", RoleNormalBlue, FactionService)

	roster := map[string]bool{}
	for _, id := range virusRoster(state) {
		roster[id] = true
	}
	if !roster["p1"] {
		t.Fatal("a plain VIRUS agent is missing from the roster")
	}
	if roster["p2"] {
		t.Fatal("the Rogue Agent appears on the roster the other VIRUS agents read")
	}
	if !roster["p3"] {
		t.Fatal("the Triple Agent is missing from the roster VIRUS reads")
	}
	if roster["p4"] || roster["p5"] {
		t.Fatal("a plain Service agent leaked onto the VIRUS roster")
	}

	// The count and the names are supposed to disagree; that mismatch is the
	// only signal either fake role gives off.
	if got := trueVirusCount(state); got != 2 {
		t.Fatalf("true VIRUS count = %d, want 2", got)
	}
	if len(roster) != 2 {
		t.Fatalf("roster size = %d, want 2 (one real agent plus the plant)", len(roster))
	}

	if !seesVirusRoster(state.Players["p2"]) {
		t.Fatal("the Rogue Agent cannot see the roster they are missing from")
	}
	if !seesVirusRoster(state.Players["p3"]) {
		t.Fatal("the Triple Agent cannot see the roster they were planted on")
	}
	if seesVirusRoster(state.Players["p4"]) {
		t.Fatal("a plain Service agent can see the VIRUS roster")
	}
}

func TestRosterReachesOnlyTheVirusSideOfTheProjection(t *testing.T) {
	state := lobbyWithPlayers(t, 5)
	assignRoles(&state)
	setRole(&state, "p1", RoleNormalRed, FactionVirus)
	setRole(&state, "p2", RoleNormalRed, FactionVirus)
	setRole(&state, "p3", RoleNormalBlue, FactionService)
	setRole(&state, "p4", RoleNormalBlue, FactionService)
	setRole(&state, "p5", RoleNormalBlue, FactionService)

	virusView := Project(state, "p1").Private
	if len(virusView.VirusRoster) != 1 || virusView.VirusRoster[0].ID != "p2" {
		t.Fatalf("VIRUS roster for p1 = %#v, want exactly p2", virusView.VirusRoster)
	}
	if virusView.VirusTeamSize != 2 {
		t.Fatalf("VIRUS team size = %d, want 2", virusView.VirusTeamSize)
	}

	serviceView := Project(state, "p3").Private
	if len(serviceView.VirusRoster) != 0 || serviceView.VirusTeamSize != 0 {
		t.Fatalf("a Service agent received VIRUS roster data: %#v", serviceView)
	}
}

func TestSpecialRolesAreDealtOnlyWhenEnabledAndOnlyToTheirOwnFaction(t *testing.T) {
	settings := DefaultRoomSettings()
	settings.EnabledRoles = map[string]bool{}
	for _, definition := range RoleDefinitions() {
		if definition.Special {
			settings.EnabledRoles[definition.ID] = true
		}
	}
	state := NewLobby("room_roles", "p1", "Agent A", settings)
	for i := 2; i <= 9; i++ {
		if err := state.AddPlayer(string("p"+string(rune('0'+i))), "Agent "+string(rune('A'+i-1))); err != nil {
			t.Fatal(err)
		}
	}
	assignRoles(&state)

	special := 0
	for _, id := range state.PlayerOrder {
		player := state.Players[id]
		definition, ok := RoleDefinitionFor(string(player.Role))
		if !ok {
			t.Fatalf("player %s holds unknown role %q", id, player.Role)
		}
		if !definition.Special {
			continue
		}
		special++
		if definition.Faction != player.Faction {
			t.Fatalf("%s was dealt to a %s agent", definition.Name, player.Faction)
		}
	}
	if special == 0 {
		t.Fatal("roles were enabled but nobody was dealt a special role")
	}

	// With the pool switched off, every player falls back to a baseline role.
	off := DefaultRoomSettings()
	off.EnabledRoles = map[string]bool{}
	stateOff := NewLobby("room_roles_off", "p1", "Agent A", off)
	for i := 2; i <= 6; i++ {
		if err := stateOff.AddPlayer(string("p"+string(rune('0'+i))), "Agent "+string(rune('A'+i-1))); err != nil {
			t.Fatal(err)
		}
	}
	assignRoles(&stateOff)
	for _, id := range stateOff.PlayerOrder {
		if role := stateOff.Players[id].Role; role != baselineRole(stateOff.Players[id].Faction) {
			t.Fatalf("player %s holds %q with an empty role pool", id, role)
		}
	}
}

func TestRolesStayOffUnlessTheHostEnablesThem(t *testing.T) {
	state := lobbyWithPlayers(t, 6)
	assignRoles(&state)
	for _, id := range state.PlayerOrder {
		player := state.Players[id]
		if player.Role != baselineRole(player.Faction) {
			t.Fatalf("player %s was dealt %q while roles are disabled", id, player.Role)
		}
		if player.ApparentFaction != nil {
			t.Fatalf("player %s has an apparent agency while roles are disabled", id)
		}
	}
}
