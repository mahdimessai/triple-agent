package domain

import (
	"fmt"
	"testing"
	"time"
)

// advanceToDiscussion plays out the remaining operations — each one resolved by
// its recipient and followed by the interlude beat — until the room reaches the
// final discussion before the accusation.
func advanceToDiscussion(t *testing.T, state GameState, now time.Time) GameState {
	t.Helper()
	for range make([]struct{}, 4*len(state.PlayerOrder)+8) {
		switch state.Phase {
		case PhaseDiscussion:
			return state
		case PhaseOperationInterlude:
			transition, err := Apply(state, Command{ActorID: state.HostID, ExpectedVersion: state.Version, Kind: CommandAdvanceInterlude}, now)
			if err != nil {
				t.Fatalf("advance interlude: %v", err)
			}
			state = transition.State
		case PhaseOperationResult:
			transition, err := Apply(state, Command{ActorID: state.ActivePlayerID, ExpectedVersion: state.Version, Kind: CommandOperationExplainDone}, now)
			if err != nil {
				t.Fatalf("explain done: %v", err)
			}
			state = transition.State
		case PhaseOperationInput:
			operation := state.Operation
			resolver, resolverErr := operationResolverFor(operation.Kind)
			if resolverErr != nil {
				t.Fatalf("resolver for %s: %v", operation.Kind, resolverErr)
			}
			if operation.Kind == "ChooseVoteShield" && operation.Step == 2 {
				transition, err := Apply(state, Command{ActorID: operation.InputOwnerID, ExpectedVersion: state.Version, Kind: CommandResolveOperation, Choice: "VOTE_SHIELD"}, now)
				if err != nil {
					t.Fatalf("resolve operation %s step 2: %v", operation.Kind, err)
				}
				state = transition.State
				continue
			}
			count := resolver.Definition().TargetCount
			if count == 0 {
				count = 1
			}
			targets := legalOperationTargets(state, state.ActivePlayerID, count)
			command := Command{ActorID: state.ActivePlayerID, ExpectedVersion: state.Version, Kind: CommandResolveOperation, TargetIDs: targets[:min(count, len(targets))]}
			if operation.InputKind == OperationInputChoice {
				command.Choice = "STAY"
			}
			transition, err := Apply(state, command, now)
			if err != nil {
				t.Fatalf("resolve operation: %v", err)
			}
			state = transition.State
		default:
			t.Fatalf("unexpected phase %s while advancing to discussion", state.Phase)
		}
	}
	t.Fatal("operations never reached the final discussion")
	return state
}

func operationKindsUntilDiscussion(t *testing.T, state GameState, now time.Time) ([]string, GameState) {
	t.Helper()
	kinds := make([]string, 0)
	lastOperationID := ""
	for range make([]struct{}, 8*len(state.PlayerOrder)+16) {
		if state.Operation != nil && state.Operation.ID != lastOperationID && (state.Phase == PhaseOperationInput || state.Phase == PhaseOperationResult) {
			kinds = append(kinds, state.Operation.Kind)
			lastOperationID = state.Operation.ID
		}
		switch state.Phase {
		case PhaseDiscussion:
			return kinds, state
		case PhaseOperationInterlude:
			transition, err := Apply(state, Command{ActorID: state.HostID, ExpectedVersion: state.Version, Kind: CommandAdvanceInterlude}, now)
			if err != nil {
				t.Fatalf("advance interlude: %v", err)
			}
			state = transition.State
		case PhaseOperationResult:
			transition, err := Apply(state, Command{ActorID: state.ActivePlayerID, ExpectedVersion: state.Version, Kind: CommandOperationExplainDone}, now)
			if err != nil {
				t.Fatalf("explain done: %v", err)
			}
			state = transition.State
		case PhaseOperationInput:
			operation := state.Operation
			resolver, resolverErr := operationResolverFor(operation.Kind)
			if resolverErr != nil {
				t.Fatalf("resolver for %s: %v", operation.Kind, resolverErr)
			}
			if operation.Kind == "ChooseVoteShield" && operation.Step == 2 {
				transition, err := Apply(state, Command{ActorID: operation.InputOwnerID, ExpectedVersion: state.Version, Kind: CommandResolveOperation, Choice: "VOTE_SHIELD"}, now)
				if err != nil {
					t.Fatalf("resolve operation %s step 2: %v", operation.Kind, err)
				}
				state = transition.State
				continue
			}
			count := resolver.Definition().TargetCount
			if count == 0 {
				count = 1
			}
			targets := legalOperationTargets(state, state.ActivePlayerID, count)
			command := Command{ActorID: state.ActivePlayerID, ExpectedVersion: state.Version, Kind: CommandResolveOperation, TargetIDs: targets[:min(count, len(targets))]}
			if operation.InputKind == OperationInputChoice {
				command.Choice = "STAY"
			}
			transition, err := Apply(state, command, now)
			if err != nil {
				t.Fatalf("resolve operation %s: %v", operation.Kind, err)
			}
			state = transition.State
		default:
			t.Fatalf("unexpected phase %s while collecting operations", state.Phase)
		}
	}
	t.Fatal("operations never reached the final discussion")
	return kinds, state
}

func readyFivePlayerMatch(t *testing.T, settings RoomSettings, now time.Time) GameState {
	t.Helper()
	settings.MaxPlayers = 5
	state := NewLobby("room_test", "p1", "Agent A", settings)
	for i := 2; i <= 5; i++ {
		id := string("p" + string(rune('0'+i)))
		if err := state.AddPlayer(id, "Agent "+string(rune('A'+i-1))); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range state.PlayerOrder {
		transition, err := Apply(state, Command{ActorID: id, ExpectedVersion: state.Version, Kind: CommandSetReady}, now)
		if err != nil {
			t.Fatalf("ready %s: %v", id, err)
		}
		state = transition.State
	}
	transition, err := Apply(state, Command{ActorID: state.HostID, ExpectedVersion: state.Version, Kind: CommandStartMatch}, now)
	if err != nil {
		t.Fatalf("start match: %v", err)
	}
	state = transition.State
	for _, id := range state.PlayerOrder {
		transition, err = Apply(state, Command{ActorID: id, ExpectedVersion: state.Version, Kind: CommandAcknowledgeRole}, now)
		if err != nil {
			t.Fatalf("role ack %s: %v", id, err)
		}
		state = transition.State
	}
	return state
}

func TestOperationPoolServesEverySlotBeforeRepeating(t *testing.T) {
	settings := DefaultRoomSettings()
	settings.EnabledOperations = map[string]bool{
		"Share":      true,
		"Detector":   true,
		"OneRandom":  true,
		"OneOfTwo":   true,
		"TwoFriends": true,
		"Swap":       true,
	}
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	state := readyFivePlayerMatch(t, settings, now)

	kinds, finalState := operationKindsUntilDiscussion(t, state, now)
	if len(kinds) != 6 {
		t.Fatalf("served operations = %#v, want six deck slots", kinds)
	}
	seen := make(map[string]bool, len(kinds))
	for _, kind := range kinds {
		if seen[kind] {
			t.Fatalf("operation %s repeated before the full deck was served: %#v", kind, kinds)
		}
		seen[kind] = true
	}
	for kind := range settings.EnabledOperations {
		if !seen[kind] {
			t.Fatalf("enabled operation %s was never served: %#v", kind, kinds)
		}
	}
	if finalState.Phase != PhaseDiscussion {
		t.Fatalf("phase = %s, want discussion", finalState.Phase)
	}
}

func TestOperationPoolReshufflesOnlyAfterDeckExhaustion(t *testing.T) {
	settings := DefaultRoomSettings()
	settings.EnabledOperations = map[string]bool{"Detector": true, "Share": true}
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	state := readyFivePlayerMatch(t, settings, now)

	kinds, _ := operationKindsUntilDiscussion(t, state, now)
	if len(kinds) != len(state.PlayerOrder) {
		t.Fatalf("served operations = %#v, want one turn per player", kinds)
	}
	if kinds[0] == kinds[1] {
		t.Fatalf("operation repeated before the two-card deck was exhausted: %#v", kinds)
	}
	for _, kind := range kinds {
		if kind != "Detector" && kind != "Share" {
			t.Fatalf("unexpected operation %s in %#v", kind, kinds)
		}
	}
}

func TestOperationPoolDoesNotDealFutureEventOperationFirst(t *testing.T) {
	settings := DefaultRoomSettings()
	settings.EnabledOperations = map[string]bool{"ChooseVoteShield": true}
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	settings.MaxPlayers = 5
	state := NewLobby("room_test", "p1", "Agent A", settings)
	for i := 2; i <= 5; i++ {
		id := string("p" + string(rune('0'+i)))
		if err := state.AddPlayer(id, "Agent "+string(rune('A'+i-1))); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range state.PlayerOrder {
		transition, err := Apply(state, Command{ActorID: id, ExpectedVersion: state.Version, Kind: CommandSetReady}, now)
		if err != nil {
			t.Fatalf("ready %s: %v", id, err)
		}
		state = transition.State
	}
	if _, err := Apply(state, Command{ActorID: state.HostID, ExpectedVersion: state.Version, Kind: CommandStartMatch}, now); err != ErrNoEligibleOperations {
		t.Fatalf("future-only operation start err = %v, want %v", err, ErrNoEligibleOperations)
	}
}

func TestOperationPoolReleasesFutureEventOperationAfterItsGate(t *testing.T) {
	settings := DefaultRoomSettings()
	settings.EnabledOperations = map[string]bool{"Detector": true, "ChooseVoteShield": true}
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	state := readyFivePlayerMatch(t, settings, now)
	kinds, _ := operationKindsUntilDiscussion(t, state, now)
	if len(kinds) != len(state.PlayerOrder) {
		t.Fatalf("served operations = %#v, want one turn per player", kinds)
	}
	if kinds[0] == "ChooseVoteShield" {
		t.Fatalf("future-event operation was served first: %#v", kinds)
	}
	seenFuture := false
	for _, kind := range kinds {
		if kind == "ChooseVoteShield" {
			seenFuture = true
		}
	}
	if !seenFuture {
		t.Fatalf("future-event operation was never released: %#v", kinds)
	}
}

func TestAnonymousTipFlowAndVoteResolution(t *testing.T) {
	settings := DefaultRoomSettings()
	settings.MaxPlayers = 5
	settings.EnabledOperations = map[string]bool{"OneRandom": true}
	state := NewLobby("room_test", "p1", "Agent A", settings)
	for i := 2; i <= 5; i++ {
		id := string("p" + string(rune('0'+i)))
		if err := state.AddPlayer(id, "Agent "+string(rune('A'+i-1))); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	for _, id := range state.PlayerOrder {
		transition, err := Apply(state, Command{ActorID: id, ExpectedVersion: state.Version, Kind: CommandSetReady}, now)
		if err != nil {
			t.Fatalf("ready %s: %v", id, err)
		}
		state = transition.State
	}
	transition, err := Apply(state, Command{ActorID: state.HostID, ExpectedVersion: state.Version, Kind: CommandStartMatch}, now)
	if err != nil {
		t.Fatal(err)
	}
	state = transition.State
	if state.Phase != PhaseRoleReveal {
		t.Fatalf("phase = %s, want %s", state.Phase, PhaseRoleReveal)
	}
	for _, id := range state.PlayerOrder {
		transition, err = Apply(state, Command{ActorID: id, ExpectedVersion: state.Version, Kind: CommandAcknowledgeRole}, now)
		if err != nil {
			t.Fatalf("role ack %s: %v", id, err)
		}
		state = transition.State
	}
	if state.Phase != PhaseOperationResult || state.Operation == nil || state.Operation.Name != "Anonymous Tip" {
		t.Fatalf("operation state = %#v, want Anonymous Tip result", state.Operation)
	}
	active := state.ActivePlayerID
	private := Project(state, active).Private
	if private.OperationResult == nil || private.OperationResult.TargetPlayerID == "" {
		t.Fatal("active player did not receive the private Anonymous Tip result")
	}
	otherID := state.PlayerOrder[1]
	if otherID == active {
		otherID = state.PlayerOrder[0]
	}
	if other := Project(state, otherID).Private; other.OperationResult != nil {
		t.Fatal("another player received the private Anonymous Tip result")
	}
	transition, err = Apply(state, Command{ActorID: active, ExpectedVersion: state.Version, Kind: CommandOperationExplainDone}, now)
	if err != nil {
		t.Fatal(err)
	}
	state = transition.State
	// Every player now receives an operation, separated by a timed interlude,
	// before the room reaches its final discussion.
	state = advanceToDiscussion(t, state, now)
	for _, id := range state.PlayerOrder {
		transition, err = Apply(state, Command{ActorID: id, ExpectedVersion: state.Version, Kind: CommandAdvanceDiscussion}, now)
		if err != nil {
			t.Fatal(err)
		}
		state = transition.State
	}
	for _, id := range state.PlayerOrder {
		target := state.PlayerOrder[0]
		if target == id {
			target = state.PlayerOrder[1]
		}
		transition, err = Apply(state, Command{ActorID: id, ExpectedVersion: state.Version, Kind: CommandSubmitVote, TargetID: target}, now)
		if err != nil {
			t.Fatalf("vote %s: %v", id, err)
		}
		state = transition.State
	}
	if state.Phase != PhaseResultsIntro {
		t.Fatalf("phase = %s, want %s", state.Phase, PhaseResultsIntro)
	}
	if state.Winner == "" {
		t.Fatal("vote results did not calculate a winner")
	}
	intro := Project(state, state.HostID).Public
	if len(intro.VoteTotals) == 0 || intro.ImprisonedPlayerID == "" || intro.Winner == "" || len(intro.Leaderboard) == 0 {
		t.Fatalf("results intro missing resolved data: %#v", intro)
	}
	expectedPhases := []Phase{PhaseVoteResults, PhaseImprisonment, PhaseAgencyReveal, PhaseOutcomeReveal, PhaseLeaderboard, PhaseOutOfLoop, PhaseEnd}
	for index, expected := range expectedPhases {
		transition, err = Apply(state, Command{ActorID: state.HostID, ExpectedVersion: state.Version, Kind: CommandContinueResults}, now)
		if err != nil {
			t.Fatalf("results advance %d: %v", index, err)
		}
		state = transition.State
		if state.Phase != expected {
			t.Fatalf("results phase %d = %s, want %s", index, state.Phase, expected)
		}
		public := Project(state, state.HostID).Public
		if len(public.VoteTotals) == 0 || public.ImprisonedPlayerID == "" || public.Winner == "" || len(public.Leaderboard) == 0 {
			t.Fatalf("results phase %d (%s) missing data: %#v", index, state.Phase, public)
		}
	}
}

func TestStaleVersionIsRejected(t *testing.T) {
	state := NewLobby("room_test", "p1", "Agent A", DefaultRoomSettings())
	_, err := Apply(state, Command{ActorID: "p1", ExpectedVersion: 42, Kind: CommandSetReady}, time.Now())
	if err != ErrStaleVersion {
		t.Fatalf("err = %v, want %v", err, ErrStaleVersion)
	}
}

func TestHostControlsOperationPoolAndStartRandomizesFromIt(t *testing.T) {
	settings := DefaultRoomSettings()
	settings.MaxPlayers = 5
	settings.EnabledOperations = map[string]bool{"Swap": true, "Detector": true}
	state := NewLobby("room_test", "p1", "Agent A", settings)
	for i := 2; i <= 5; i++ {
		id := string("p" + string(rune('0'+i)))
		if err := state.AddPlayer(id, "Agent "+string(rune('A'+i-1))); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	if _, err := Apply(state, Command{ActorID: "p2", ExpectedVersion: state.Version, Kind: CommandSetOperationEnabled, OperationKind: "Swap", OperationEnabled: false}, now); err != ErrNotAllowed {
		t.Fatalf("non-host operation setting err = %v, want %v", err, ErrNotAllowed)
	}
	transition, err := Apply(state, Command{ActorID: state.HostID, ExpectedVersion: state.Version, Kind: CommandSetOperationEnabled, OperationKind: "Swap", OperationEnabled: false}, now)
	if err != nil {
		t.Fatal(err)
	}
	state = transition.State
	if state.Settings.EnabledOperations["Swap"] || !state.Settings.EnabledOperations["Detector"] {
		t.Fatalf("operation pool = %#v", state.Settings.EnabledOperations)
	}
	if _, err := Apply(state, Command{ActorID: state.HostID, ExpectedVersion: state.Version, Kind: CommandSetOperationEnabled, OperationKind: "Ambassador", OperationEnabled: true}, now); err != ErrNotAllowed {
		t.Fatalf("recovered operation setting err = %v, want %v", err, ErrNotAllowed)
	}

	for _, id := range state.PlayerOrder {
		transition, err = Apply(state, Command{ActorID: id, ExpectedVersion: state.Version, Kind: CommandSetReady}, now)
		if err != nil {
			t.Fatal(err)
		}
		state = transition.State
	}
	if _, err = Apply(state, Command{ActorID: state.HostID, ExpectedVersion: state.Version, Kind: CommandStartMatch, OperationKind: "Swap"}, now); err != ErrNotAllowed {
		t.Fatalf("disabled requested operation err = %v, want %v", err, ErrNotAllowed)
	}
	transition, err = Apply(state, Command{ActorID: state.HostID, ExpectedVersion: state.Version, Kind: CommandStartMatch, OperationKind: "Detector"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if transition.State.PlannedOperation != "Detector" {
		t.Fatalf("planned operation = %s, want requested operation Detector", transition.State.PlannedOperation)
	}
	projection := Project(transition.State, state.HostID)
	if len(projection.Public.Settings.EnabledOperations) != 1 || projection.Public.Settings.EnabledOperations[0] != "Detector" {
		t.Fatalf("public operation settings = %#v", projection.Public.Settings)
	}
}

func TestLobbySettingsNoOpsDoNotAdvanceVersion(t *testing.T) {
	state := NewLobby("room_test", "host", "Host", DefaultRoomSettings())
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	transition, err := Apply(state, Command{ActorID: state.HostID, ExpectedVersion: state.Version, Kind: CommandSetOperationEnabled, OperationKind: "Detector", OperationEnabled: true}, now)
	if err != nil || transition.Changed {
		t.Fatalf("same operation setting transition = %#v, err = %v", transition, err)
	}
	if transition.State.Version != 0 {
		t.Fatalf("same operation setting mutated state = version %d", transition.State.Version)
	}
	transition, err = Apply(state, Command{ActorID: state.HostID, ExpectedVersion: state.Version, Kind: CommandSetDiscussionTimer, DiscussionTimerEnabled: true}, now)
	if err != nil || transition.Changed || transition.State.Version != 0 {
		t.Fatalf("same timer setting transition = %#v, err = %v", transition, err)
	}
}

func TestHostCanResetAnEndedMatchForRematch(t *testing.T) {
	settings := DefaultRoomSettings()
	settings.MaxPlayers = 5
	state := NewLobby("room_test", "p1", "Agent A", settings)
	for i := 2; i <= 5; i++ {
		id := string("p" + string(rune('0'+i)))
		if err := state.AddPlayer(id, "Agent "+string(rune('A'+i-1))); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range state.PlayerOrder {
		player := state.Players[id]
		player.Ready = true
		player.InitialFaction = FactionService
		player.Faction = FactionService
		player.Role = RoleNormalBlue
		player.Statuses = []string{"EXAMPLE_STATUS"}
		state.Players[id] = player
	}
	state.Phase = PhaseEnd
	state.Version = 12
	state.Winner = FactionService
	state.Operation = &OperationState{ID: "op_1", Kind: "Swap", Name: "Spy Transfer"}
	state.Vote = VoteState{Submitted: map[string]string{"p1": "p2"}, Totals: map[string]int{"p2": 1}, ImprisonedPlayerID: "p2"}

	transition, err := Apply(state, Command{ActorID: state.HostID, ExpectedVersion: state.Version, Kind: CommandRematch}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	state = transition.State
	if state.Phase != PhaseLobby || state.Version != 13 {
		t.Fatalf("rematch state = phase %s version %d", state.Phase, state.Version)
	}
	if state.Winner != FactionNone || state.Operation != nil || state.Vote.ImprisonedPlayerID != "" || len(state.Vote.Submitted) != 0 || len(state.Vote.Totals) != 0 {
		t.Fatalf("rematch did not clear match state: %#v", state)
	}
	for _, id := range state.PlayerOrder {
		player := state.Players[id]
		if player.Ready || player.Role != "" || player.InitialFaction != "" || player.Faction != "" || len(player.Statuses) != 0 {
			t.Fatalf("player %s was not reset: %#v", id, player)
		}
	}
}

func TestOperationRegistryExposesEveryCatalogOperation(t *testing.T) {
	want := []string{
		"Swap", "Injection", "Share", "Detector", "Strain", "Grudge", "Infatuation", "Flip", "HiddenOneRandom", "OneRandom",
		"OneOfTwo", "TwoFriends", "Undercover", "InfoForTwo", "ChooseVoteShield", "Defect", "Power", "Vote", "Confirm", "NegativeVote",
		"Ambassador", "Brig", "EarlyVote", "Hunter", "LastEvent",
	}
	definitions := OperationDefinitions()
	if len(definitions) != len(want) {
		t.Fatalf("operation definition count = %d, want %d", len(definitions), len(want))
	}
	seen := make(map[string]bool, len(definitions))
	for _, definition := range definitions {
		seen[definition.ID] = true
		if definition.Name == "" || definition.PublicInstruction == "" || definition.PrivateInstruction == "" {
			t.Fatalf("incomplete operation definition: %#v", definition)
		}
	}
	for _, id := range want {
		if !seen[id] {
			t.Fatalf("operation %s is missing from registry", id)
		}
	}
}

func TestDefaultSettingsExposeOnlyVerifiedLiveOperations(t *testing.T) {
	settings := DefaultRoomSettings()
	defaultOps := []string{
		"Grudge", "Infatuation", "Share", "Detector", "Strain",
		"Flip", "HiddenOneRandom", "TwoFriends", "OneOfTwo", "OneRandom",
	}
	for _, opID := range defaultOps {
		if !settings.EnabledOperations[opID] {
			t.Fatalf("default pool omitted %s", opID)
		}
	}
	packOps := []string{"Swap", "Undercover", "InfoForTwo", "ChooseVoteShield", "Defect"}
	for _, opID := range packOps {
		if settings.EnabledOperations[opID] {
			t.Fatalf("default pool should not enable pack operation %s", opID)
		}
	}
	if _, ok := OperationDefinitionFor(""); ok {
		t.Fatal("empty operation ID should not resolve as a default alias")
	}
	if CanonicalOperationKind("SpyTransfer") != "Swap" {
		t.Fatal("legacy SpyTransfer alias was not normalized")
	}
}

func TestEveryOperationResolverCanExecuteItsDeclaredContract(t *testing.T) {
	for _, definition := range OperationDefinitions() {
		definition := definition
		if definition.MinEventOrder > 1 {
			continue
		}
		t.Run(definition.ID, func(t *testing.T) {
			settings := DefaultRoomSettings()
			settings.MaxPlayers = 5
			settings.EnabledOperations = map[string]bool{definition.ID: true}
			state := NewLobby("room_test", "p1", "Agent A", settings)
			for i := 2; i <= 5; i++ {
				id := string("p" + string(rune('0'+i)))
				if err := state.AddPlayer(id, "Agent "+string(rune('A'+i-1))); err != nil {
					t.Fatal(err)
				}
			}
			assignRoles(&state)
			resolver, err := operationResolverFor(definition.ID)
			if err != nil {
				t.Fatal(err)
			}
			if err := resolver.Begin(&state); err != nil {
				t.Fatal(err)
			}
			if state.Operation == nil {
				t.Fatal("resolver did not create operation state")
			}
			if definition.InputKind == OperationInputPrivateInfo || definition.InputKind == OperationInputNone {
				if len(state.Operation.PrivateResults) == 0 {
					t.Fatal("private operation did not produce a private result")
				}
				return
			}

			if definition.ID == "ChooseVoteShield" {
				if err := resolver.Resolve(&state, Command{ActorID: state.ActivePlayerID, TargetIDs: []string{state.PlayerOrder[1]}}); err != nil {
					t.Fatal(err)
				}
				if err := resolver.Resolve(&state, Command{ActorID: state.PlayerOrder[1], Choice: "VOTE_SHIELD"}); err != nil {
					t.Fatal(err)
				}
				if len(state.Operation.PrivateResults) == 0 {
					t.Fatal("ChooseVoteShield did not produce private results on step 2")
				}
				return
			}

			command := Command{ActorID: state.ActivePlayerID}
			switch definition.InputKind {
			case OperationInputOneTarget:
				command.TargetIDs = []string{state.PlayerOrder[1]}
			case OperationInputTwoTargets:
				command.TargetIDs = []string{state.PlayerOrder[1], state.PlayerOrder[2]}
			case OperationInputChoice:
				command.Choice = "STAY"
			}
			if err := resolver.Resolve(&state, command); err != nil {
				t.Fatal(err)
			}
			if len(state.Operation.PrivateResults) == 0 {
				t.Fatal("operation did not produce a private result")
			}
		})
	}
}

func TestEnabledOperationDefinitionsCanResolveThroughTheGenericInputContract(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	for _, definition := range OperationDefinitions() {
		if !definition.Enabled || !IsLiveOperation(definition.ID) {
			continue
		}
		if definition.MinEventOrder > 1 {
			continue
		}
		t.Run(definition.ID, func(t *testing.T) {
			settings := DefaultRoomSettings()
			settings.MaxPlayers = 5
			settings.EnabledOperations = map[string]bool{definition.ID: true}
			state := NewLobby("room_test", "p1", "Agent A", settings)
			for i := 2; i <= 5; i++ {
				id := string("p" + string(rune('0'+i)))
				if err := state.AddPlayer(id, "Agent "+string(rune('A'+i-1))); err != nil {
					t.Fatal(err)
				}
			}
			for _, id := range state.PlayerOrder {
				transition, err := Apply(state, Command{ActorID: id, ExpectedVersion: state.Version, Kind: CommandSetReady}, now)
				if err != nil {
					t.Fatal(err)
				}
				state = transition.State
			}
			transition, err := Apply(state, Command{ActorID: state.HostID, ExpectedVersion: state.Version, Kind: CommandStartMatch, OperationKind: definition.ID}, now)
			if err != nil {
				t.Fatal(err)
			}
			state = transition.State
			for _, id := range state.PlayerOrder {
				transition, err = Apply(state, Command{ActorID: id, ExpectedVersion: state.Version, Kind: CommandAcknowledgeRole}, now)
				if err != nil {
					t.Fatal(err)
				}
				state = transition.State
			}
			if state.Operation == nil {
				t.Fatal("operation state was not created")
			}
			if state.Phase == PhaseOperationInput {
				if definition.ID == "ChooseVoteShield" {
					targetID := state.PlayerOrder[1]
					if state.ActivePlayerID == targetID {
						targetID = state.PlayerOrder[0]
					}
					transition, err = Apply(state, Command{ActorID: state.ActivePlayerID, ExpectedVersion: state.Version, Kind: CommandResolveOperation, TargetIDs: []string{targetID}}, now)
					if err != nil {
						t.Fatal(err)
					}
					state = transition.State
					transition, err = Apply(state, Command{ActorID: targetID, ExpectedVersion: state.Version, Kind: CommandResolveOperation, Choice: "VOTE_SHIELD"}, now)
					if err != nil {
						t.Fatal(err)
					}
					state = transition.State
				} else {
					command := Command{ActorID: state.ActivePlayerID, ExpectedVersion: state.Version, Kind: CommandResolveOperation}
					switch definition.InputKind {
					case OperationInputOneTarget:
						targetID := state.PlayerOrder[1]
						if targetID == state.ActivePlayerID {
							targetID = state.PlayerOrder[0]
						}
						command.TargetIDs = []string{targetID}
					case OperationInputTwoTargets:
						targets := make([]string, 0, 2)
						for _, id := range state.PlayerOrder {
							if id != state.ActivePlayerID && len(targets) < 2 {
								targets = append(targets, id)
							}
						}
						command.TargetIDs = targets
					case OperationInputChoice:
						command.Choice = "STAY"
					}
					transition, err = Apply(state, command, now)
					if err != nil {
						t.Fatal(err)
					}
					state = transition.State
				}
			}
			if state.Phase != PhaseOperationResult || len(state.Operation.PrivateResults) == 0 {
				t.Fatalf("operation did not resolve: phase=%s operation=%#v", state.Phase, state.Operation)
			}
		})
	}
}

func TestGenericResolversMatchTheirDeclaredMutationAndPrivacyContracts(t *testing.T) {
	type operationContract struct {
		id         string
		recipients func(activeID, targetID string) map[string]bool
		assert     func(*testing.T, PlayerState, PlayerState, GameState, string, string, map[string]OperationResult)
	}
	activeOnly := func(activeID, _ string) map[string]bool { return map[string]bool{activeID: true} }
	targetOnly := func(_, targetID string) map[string]bool { return map[string]bool{targetID: true} }
	activeAndTarget := func(activeID, targetID string) map[string]bool {
		return map[string]bool{activeID: true, targetID: true}
	}
	pairResult := func(t *testing.T, results map[string]OperationResult, activeID string) {
		result := results[activeID]
		if len(result.TargetPlayerIDs) != 2 || result.TargetPlayerIDs[0] == result.TargetPlayerIDs[1] {
			t.Fatalf("pair result = %#v", result)
		}
	}
	contracts := []operationContract{
		{id: "OneRandom", recipients: activeOnly, assert: func(t *testing.T, _, _ PlayerState, _ GameState, activeID, _ string, results map[string]OperationResult) {
			result := results[activeID]
			if result.Code != "FACTION_REVEALED" || result.TargetPlayerID == "" || result.TargetPlayerID == activeID {
				t.Fatalf("anonymous tip result = %#v", result)
			}
		}},
		{id: "Swap", recipients: activeAndTarget, assert: func(t *testing.T, beforeActive, beforeTarget PlayerState, state GameState, activeID, targetID string, results map[string]OperationResult) {
			if state.Players[activeID].Faction != beforeTarget.Faction || state.Players[targetID].Faction != beforeActive.Faction {
				t.Fatalf("swap mutation before=(%#v, %#v) after=(%#v, %#v)", beforeActive, beforeTarget, state.Players[activeID], state.Players[targetID])
			}
			if state.Players[activeID].InitialFaction != beforeActive.InitialFaction || state.Players[targetID].InitialFaction != beforeTarget.InitialFaction {
				t.Fatalf("swap changed initial faction")
			}
			if results[activeID].Code != "FACTIONS_EXCHANGED" || results[targetID].Code != "FACTIONS_EXCHANGED" {
				t.Fatalf("swap results = %#v", results)
			}
		}},
		{id: "Detector", recipients: activeOnly, assert: func(t *testing.T, _, _ PlayerState, _ GameState, activeID, _ string, results map[string]OperationResult) {
			pairResult(t, results, activeID)
		}},
		{id: "Share", recipients: targetOnly, assert: func(t *testing.T, _, _ PlayerState, _ GameState, activeID, targetID string, results map[string]OperationResult) {
			if results[targetID].Code != "AGENCY_SHARED" || results[targetID].OtherPlayerID != activeID || results[targetID].OtherFaction == "" {
				t.Fatalf("share result = %#v", results)
			}
		}},
		{id: "Injection", recipients: targetOnly, assert: func(t *testing.T, beforeActive, _ PlayerState, state GameState, _, targetID string, results map[string]OperationResult) {
			if state.Players[targetID].Faction != beforeActive.Faction || results[targetID].Code != "AGENCY_ASSIGNED" {
				t.Fatalf("injection mutation/result = target=%#v results=%#v", state.Players[targetID], results)
			}
		}},
		{id: "Strain", recipients: activeOnly, assert: func(t *testing.T, _, _ PlayerState, state GameState, activeID, _ string, results map[string]OperationResult) {
			if state.Players[activeID].ObjectiveKind != "IMPRISON_SELF" || results[activeID].Code != "OBJECTIVE_ASSIGNED" {
				t.Fatalf("strain mutation/result = player=%#v results=%#v", state.Players[activeID], results)
			}
		}},
		{id: "Grudge", recipients: activeOnly, assert: func(t *testing.T, _, _ PlayerState, state GameState, activeID, _ string, results map[string]OperationResult) {
			if state.Players[activeID].ObjectiveKind != "IMPRISON_TARGET" || state.Players[activeID].ObjectiveTarget == "" || state.Players[activeID].ObjectiveTarget == activeID {
				t.Fatalf("grudge mutation = %#v", state.Players[activeID])
			}
			if results[activeID].Code != "GRUDGE_TARGET_ASSIGNED" {
				t.Fatalf("grudge result = %#v", results[activeID])
			}
		}},
		{id: "Infatuation", recipients: activeOnly, assert: func(t *testing.T, _, _ PlayerState, state GameState, activeID, _ string, results map[string]OperationResult) {
			if state.Players[activeID].ObjectiveKind != "TARGET_WINS" || state.Players[activeID].ObjectiveTarget == "" || state.Players[activeID].ObjectiveTarget == activeID {
				t.Fatalf("infatuation mutation = %#v", state.Players[activeID])
			}
			if results[activeID].Code != "INFATUATION_TARGET_ASSIGNED" {
				t.Fatalf("infatuation result = %#v", results[activeID])
			}
		}},
		{id: "Flip", recipients: activeOnly, assert: func(t *testing.T, beforeActive, _ PlayerState, state GameState, activeID, _ string, _ map[string]OperationResult) {
			if state.Players[activeID].Faction == beforeActive.Faction || len(state.Players[activeID].Statuses) <= len(beforeActive.Statuses) {
				t.Fatalf("flip mutation before=%#v after=%#v", beforeActive, state.Players[activeID])
			}
		}},
		{id: "HiddenOneRandom", recipients: activeOnly, assert: func(t *testing.T, _, _ PlayerState, _ GameState, activeID, _ string, results map[string]OperationResult) {
			result := results[activeID]
			if result.Code != "FACTION_REVEALED" || result.TargetPlayerID == "" || result.TargetPlayerID == activeID {
				t.Fatalf("hidden tip result = %#v", result)
			}
		}},
		{id: "OneOfTwo", recipients: activeOnly, assert: func(t *testing.T, _, _ PlayerState, _ GameState, activeID, _ string, results map[string]OperationResult) {
			pairResult(t, results, activeID)
		}},
		{id: "TwoFriends", recipients: activeOnly, assert: func(t *testing.T, _, _ PlayerState, _ GameState, activeID, _ string, results map[string]OperationResult) {
			pairResult(t, results, activeID)
		}},
		{id: "Undercover", recipients: activeOnly, assert: func(t *testing.T, _, _ PlayerState, _ GameState, activeID, targetID string, results map[string]OperationResult) {
			if results[activeID].TargetPlayerID != targetID || results[activeID].Code == "" {
				t.Fatalf("undercover result = %#v", results[activeID])
			}
		}},
		{id: "InfoForTwo", recipients: activeAndTarget, assert: func(t *testing.T, _, _ PlayerState, _ GameState, activeID, targetID string, results map[string]OperationResult) {
			if results[activeID].TargetPlayerID != targetID || results[targetID].TargetPlayerID != activeID {
				t.Fatalf("shared information results = %#v", results)
			}
		}},
		{id: "ChooseVoteShield", recipients: targetOnly, assert: func(t *testing.T, _, _ PlayerState, state GameState, activeID, _ string, _ map[string]OperationResult) {
			if !containsString(state.Players[activeID].Statuses, "VOTE_SHIELD") {
				t.Fatalf("choose vote shield active = %#v", state.Players[activeID])
			}
		}},
		{id: "Defect", recipients: activeOnly, assert: func(t *testing.T, beforeActive, _ PlayerState, state GameState, activeID, _ string, _ map[string]OperationResult) {
			activeAfter := state.Players[activeID]
			if activeAfter.InitialFaction != beforeActive.InitialFaction {
				t.Fatalf("defect changed initial faction before=%#v after=%#v", beforeActive, activeAfter)
			}
			switch beforeActive.Faction {
			case FactionVirus:
				if activeAfter.Faction != FactionService || activeAfter.ObjectiveKind != "RED_DEFECTOR" || !containsString(activeAfter.Statuses, "RED_DEFECTOR") {
					t.Fatalf("red defect mutation before=%#v after=%#v", beforeActive, activeAfter)
				}
			case FactionService:
				if activeAfter.Faction != FactionVirus || activeAfter.CanVote || !containsString(activeAfter.Statuses, "BLUE_DEFECTOR") {
					t.Fatalf("blue defect mutation before=%#v after=%#v", beforeActive, activeAfter)
				}
			default:
				t.Fatalf("defect started with unsupported faction: %#v", beforeActive)
			}
		}},
		{id: "Power", recipients: activeOnly, assert: func(t *testing.T, _, beforeTarget PlayerState, state GameState, _, targetID string, _ map[string]OperationResult) {
			if state.Players[targetID].VotingPower <= beforeTarget.VotingPower || !containsString(state.Players[targetID].Statuses, "DOUBLE_VOTE") {
				t.Fatalf("double vote target before=%#v after=%#v", beforeTarget, state.Players[targetID])
			}
		}},
		{id: "Vote", recipients: activeOnly, assert: func(t *testing.T, _, _ PlayerState, state GameState, _, targetID string, _ map[string]OperationResult) {
			if !containsString(state.Players[targetID].Statuses, "EXTRA_SUSPICION") {
				t.Fatalf("extra suspicion target = %#v", state.Players[targetID])
			}
		}},
		{id: "Confirm", recipients: activeOnly, assert: func(t *testing.T, _, _ PlayerState, _ GameState, activeID, _ string, results map[string]OperationResult) {
			if results[activeID].Code != "AGENCY_UNCHANGED" && results[activeID].Code != "AGENCY_CHANGED" {
				t.Fatalf("confirmation result = %#v", results[activeID])
			}
		}},
		{id: "NegativeVote", recipients: activeOnly, assert: func(t *testing.T, _, _ PlayerState, state GameState, _, targetID string, _ map[string]OperationResult) {
			if !containsString(state.Players[targetID].Statuses, "VOTE_SHIELD") {
				t.Fatalf("negative vote target = %#v", state.Players[targetID])
			}
		}},
		{id: "Brig", recipients: activeOnly, assert: func(t *testing.T, _, _ PlayerState, state GameState, _, targetID string, _ map[string]OperationResult) {
			if state.Players[targetID].CanVote || !containsString(state.Players[targetID].Statuses, "SILENCED") {
				t.Fatalf("brig target = %#v", state.Players[targetID])
			}
		}},
		{id: "EarlyVote", recipients: activeOnly, assert: func(t *testing.T, _, _ PlayerState, _ GameState, activeID, targetID string, results map[string]OperationResult) {
			if results[activeID].Code != "EARLY-VOTE" || results[activeID].TargetPlayerID != targetID {
				t.Fatalf("early vote result = %#v", results[activeID])
			}
		}},
		{id: "Hunter", recipients: activeOnly, assert: func(t *testing.T, _, _ PlayerState, _ GameState, activeID, targetID string, results map[string]OperationResult) {
			if results[activeID].Code != "HUNTER" || results[activeID].TargetPlayerID != targetID {
				t.Fatalf("hunter result = %#v", results[activeID])
			}
		}},
		{id: "Ambassador", recipients: activeOnly, assert: func(t *testing.T, _, _ PlayerState, _ GameState, activeID, _ string, results map[string]OperationResult) {
			if results[activeID].Code != "AMBASSADOR" {
				t.Fatalf("ambassador result = %#v", results[activeID])
			}
		}},
		{id: "LastEvent", recipients: activeOnly, assert: func(t *testing.T, _, _ PlayerState, _ GameState, activeID, _ string, results map[string]OperationResult) {
			if results[activeID].Code != "LAST-EVENT" {
				t.Fatalf("last event result = %#v", results[activeID])
			}
		}},
	}

	contractByID := make(map[string]operationContract, len(contracts))
	for _, contract := range contracts {
		if _, exists := contractByID[contract.id]; exists {
			t.Fatalf("duplicate operation contract %q", contract.id)
		}
		contractByID[contract.id] = contract
	}
	for _, definition := range OperationDefinitions() {
		if _, ok := contractByID[definition.ID]; !ok {
			t.Errorf("operation %q has no explicit mutation/privacy contract", definition.ID)
		}
	}
	if len(contractByID) != len(OperationDefinitions()) {
		t.Fatalf("contract count = %d, operation definition count = %d", len(contractByID), len(OperationDefinitions()))
	}

	for _, contract := range contracts {
		contract := contract
		t.Run(contract.id, func(t *testing.T) {
			state := NewLobby("room_test", "p1", "Agent A", DefaultRoomSettings())
			for i := 2; i <= 5; i++ {
				id := string("p" + string(rune('0'+i)))
				if err := state.AddPlayer(id, "Agent "+string(rune('A'+i-1))); err != nil {
					t.Fatal(err)
				}
			}
			assignRoles(&state)
			resolver, err := operationResolverFor(contract.id)
			if err != nil {
				t.Fatal(err)
			}
			activeID := activePlayerID(state)
			activeBefore := state.Players[activeID]
			activeBefore.Statuses = append([]string(nil), activeBefore.Statuses...)
			if err := resolver.Begin(&state); err != nil {
				t.Fatal(err)
			}
			if state.Operation == nil {
				t.Fatal("resolver did not create operation state")
			}
			activeID = state.ActivePlayerID
			var targetID string
			for _, candidate := range state.PlayerOrder {
				if candidate != activeID {
					targetID = candidate
					break
				}
			}
			targetBefore := state.Players[targetID]
			targetBefore.Statuses = append([]string(nil), targetBefore.Statuses...)
			if contract.id == "ChooseVoteShield" {
				if err := resolver.Resolve(&state, Command{ActorID: activeID, TargetIDs: []string{targetID}}); err != nil {
					t.Fatal(err)
				}
				if err := resolver.Resolve(&state, Command{ActorID: targetID, Choice: "VOTE_SHIELD"}); err != nil {
					t.Fatal(err)
				}
			} else if state.Operation.InputKind != OperationInputPrivateInfo && state.Operation.InputKind != OperationInputNone {
				command := Command{ActorID: activeID, TargetIDs: []string{targetID}}
				switch state.Operation.InputKind {
				case OperationInputTwoTargets:
					command.TargetIDs = []string{targetID, state.PlayerOrder[2]}
				case OperationInputChoice:
					command.TargetIDs = nil
					command.Choice = "DEFECT"
				}
				if err := resolver.Resolve(&state, command); err != nil {
					t.Fatal(err)
				}
			}
			results := state.Operation.PrivateResults
			if len(results) == 0 {
				t.Fatal("resolver produced no private result")
			}
			wantRecipients := contract.recipients(activeID, targetID)
			for recipient := range results {
				if !wantRecipients[recipient] {
					t.Fatalf("private result leaked to %s: %#v", recipient, results)
				}
			}
			if len(results) != len(wantRecipients) {
				t.Fatalf("private recipients = %#v, want exactly %#v", results, wantRecipients)
			}
			for _, playerID := range state.PlayerOrder {
				gotPrivateResult := Project(state, playerID).Private.OperationResult != nil
				if gotPrivateResult != wantRecipients[playerID] {
					t.Fatalf("projected private result for %s = %v, want %v", playerID, gotPrivateResult, wantRecipients[playerID])
				}
			}
			contract.assert(t, activeBefore, targetBefore, state, activeID, targetID, results)
		})
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestSpyTransferSwapsCurrentFactionAndSharesPrivateResult(t *testing.T) {
	settings := DefaultRoomSettings()
	settings.MaxPlayers = 5
	settings.EnabledOperations = map[string]bool{"Swap": true}
	state := NewLobby("room_test", "p1", "Agent A", settings)
	for i := 2; i <= 5; i++ {
		id := string("p" + string(rune('0'+i)))
		if err := state.AddPlayer(id, "Agent "+string(rune('A'+i-1))); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	for _, id := range state.PlayerOrder {
		transition, err := Apply(state, Command{ActorID: id, ExpectedVersion: state.Version, Kind: CommandSetReady}, now)
		if err != nil {
			t.Fatal(err)
		}
		state = transition.State
	}
	transition, err := Apply(state, Command{ActorID: state.HostID, ExpectedVersion: state.Version, Kind: CommandStartMatch, OperationKind: "Swap"}, now)
	if err != nil {
		t.Fatal(err)
	}
	state = transition.State
	for _, id := range state.PlayerOrder {
		transition, err = Apply(state, Command{ActorID: id, ExpectedVersion: state.Version, Kind: CommandAcknowledgeRole}, now)
		if err != nil {
			t.Fatal(err)
		}
		state = transition.State
	}
	if state.Phase != PhaseOperationInput || state.Operation == nil || state.Operation.Kind != "Swap" {
		t.Fatalf("operation state = %#v", state.Operation)
	}
	activeID := state.ActivePlayerID
	targetID := state.PlayerOrder[1]
	if targetID == activeID {
		targetID = state.PlayerOrder[0]
	}
	activeBefore := state.Players[activeID]
	targetBefore := state.Players[targetID]
	transition, err = Apply(state, Command{ActorID: activeID, ExpectedVersion: state.Version, Kind: CommandSelectOperationTarget, TargetID: targetID}, now)
	if err != nil {
		t.Fatal(err)
	}
	state = transition.State
	if state.Phase != PhaseOperationResult {
		t.Fatalf("phase = %s, want %s", state.Phase, PhaseOperationResult)
	}
	if state.Players[activeID].Faction != targetBefore.Faction || state.Players[targetID].Faction != activeBefore.Faction {
		t.Fatal("Spy Transfer did not exchange current factions")
	}
	if state.Players[activeID].InitialFaction != activeBefore.InitialFaction || state.Players[targetID].InitialFaction != targetBefore.InitialFaction {
		t.Fatal("Spy Transfer changed an initial faction")
	}
	if Project(state, activeID).Private.OperationResult == nil || Project(state, targetID).Private.OperationResult == nil {
		t.Fatal("Spy Transfer did not deliver private results to both players")
	}
	var bystanderID string
	for _, id := range state.PlayerOrder {
		if id != activeID && id != targetID {
			bystanderID = id
			break
		}
	}
	if bystanderID != "" && Project(state, bystanderID).Private.OperationResult != nil {
		t.Fatal("Spy Transfer leaked a private result to a bystander")
	}
}

func TestOperationRegistryResolvesSecretIntelWithTwoPrivateTargets(t *testing.T) {
	settings := DefaultRoomSettings()
	settings.MaxPlayers = 5
	settings.EnabledOperations = map[string]bool{"Detector": true}
	state := NewLobby("room_test", "p1", "Agent A", settings)
	for i := 2; i <= 5; i++ {
		id := string("p" + string(rune('0'+i)))
		if err := state.AddPlayer(id, "Agent "+string(rune('A'+i-1))); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	for _, id := range state.PlayerOrder {
		transition, err := Apply(state, Command{ActorID: id, ExpectedVersion: state.Version, Kind: CommandSetReady}, now)
		if err != nil {
			t.Fatalf("ready %s: %v", id, err)
		}
		state = transition.State
	}
	transition, err := Apply(state, Command{ActorID: state.HostID, ExpectedVersion: state.Version, Kind: CommandStartMatch, OperationKind: "Detector"}, now)
	if err != nil {
		t.Fatal(err)
	}
	state = transition.State
	for _, id := range state.PlayerOrder {
		transition, err = Apply(state, Command{ActorID: id, ExpectedVersion: state.Version, Kind: CommandAcknowledgeRole}, now)
		if err != nil {
			t.Fatalf("role ack %s: %v", id, err)
		}
		state = transition.State
	}
	if state.Operation == nil || state.Operation.InputKind != OperationInputTwoTargets || state.Phase != PhaseOperationInput {
		t.Fatalf("operation state = %#v, phase = %s", state.Operation, state.Phase)
	}

	activeID := state.ActivePlayerID
	var virusID, serviceID string
	for _, id := range state.PlayerOrder {
		if id == activeID {
			continue
		}
		switch state.Players[id].Faction {
		case FactionVirus:
			if virusID == "" {
				virusID = id
			}
		case FactionService:
			if serviceID == "" {
				serviceID = id
			}
		}
	}
	if virusID == "" || serviceID == "" {
		t.Fatal("role assignment did not provide both factions for Secret Intel")
	}
	inputPrivate := Project(state, activeID).Private
	if inputPrivate.OperationInstruction == "" || len(inputPrivate.LegalTargetIDs) != 4 {
		t.Fatalf("private input contract = %#v", inputPrivate)
	}
	_, err = Apply(state, Command{ActorID: activeID, ExpectedVersion: state.Version, Kind: CommandResolveOperation, TargetIDs: []string{virusID}}, now)
	if err != ErrInvalidTarget {
		t.Fatalf("one-target Secret Intel error = %v, want %v", err, ErrInvalidTarget)
	}
	transition, err = Apply(state, Command{ActorID: activeID, ExpectedVersion: state.Version, Kind: CommandResolveOperation, TargetIDs: []string{virusID, serviceID}}, now)
	if err != nil {
		t.Fatal(err)
	}
	state = transition.State
	if state.Phase != PhaseOperationResult {
		t.Fatalf("phase = %s, want %s", state.Phase, PhaseOperationResult)
	}
	result := Project(state, activeID).Private.OperationResult
	if result == nil || result.Code != "AT_LEAST_ONE_VIRUS" || len(result.TargetPlayerIDs) != 2 {
		t.Fatalf("Secret Intel result = %#v", result)
	}
	var intelBystanderID string
	for _, id := range state.PlayerOrder {
		if id != activeID && id != virusID && id != serviceID {
			intelBystanderID = id
			break
		}
	}
	if intelBystanderID != "" && Project(state, intelBystanderID).Private.OperationResult != nil {
		t.Fatal("Secret Intel leaked its private result to a bystander")
	}
	public := Project(state, activeID).Public.Operation
	if public == nil || public.TargetCount != 2 {
		t.Fatalf("public operation = %#v, want target count 2 without target IDs", public)
	}
	if public.PublicInstruction == "" || public.ActivePlayerName == "" {
		t.Fatalf("public operation contract = %#v", public)
	}
	private := Project(state, activeID).Private
	if private.OperationInstruction == "" {
		t.Fatalf("private operation contract = %#v", private)
	}
}

func TestRedDefectorLosesWhenAnotherVirusVotesForThem(t *testing.T) {
	state := GameState{
		Winner: FactionService,
		Players: map[string]PlayerState{
			"red":   {ID: "red", Faction: FactionService, ObjectiveKind: "RED_DEFECTOR"},
			"virus": {ID: "virus", Faction: FactionVirus},
		},
		Vote: VoteState{Submitted: map[string]string{}, Totals: map[string]int{}},
	}
	if !playerWins(state, "red") {
		t.Fatal("red defector should win with the service faction when unaccused")
	}
	state.Vote.Submitted["virus"] = "red"
	if playerWins(state, "red") {
		t.Fatal("red defector should lose when another virus votes for them")
	}
}

func TestConfessionSharesOnlyTheActiveAgencyWithTheChosenRecipient(t *testing.T) {
	settings := DefaultRoomSettings()
	settings.MaxPlayers = 5
	settings.EnabledOperations = map[string]bool{"Share": true}
	state := NewLobby("room_test", "p1", "Agent A", settings)
	for i := 2; i <= 5; i++ {
		id := string("p" + string(rune('0'+i)))
		if err := state.AddPlayer(id, "Agent "+string(rune('A'+i-1))); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	for _, id := range state.PlayerOrder {
		transition, err := Apply(state, Command{ActorID: id, ExpectedVersion: state.Version, Kind: CommandSetReady}, now)
		if err != nil {
			t.Fatalf("ready %s: %v", id, err)
		}
		state = transition.State
	}
	transition, err := Apply(state, Command{ActorID: state.HostID, ExpectedVersion: state.Version, Kind: CommandStartMatch, OperationKind: "Share"}, now)
	if err != nil {
		t.Fatal(err)
	}
	state = transition.State
	for _, id := range state.PlayerOrder {
		transition, err = Apply(state, Command{ActorID: id, ExpectedVersion: state.Version, Kind: CommandAcknowledgeRole}, now)
		if err != nil {
			t.Fatalf("role ack %s: %v", id, err)
		}
		state = transition.State
	}
	if state.Operation == nil || state.Operation.Kind != "Share" || state.Phase != PhaseOperationInput {
		t.Fatalf("operation state = %#v, phase = %s", state.Operation, state.Phase)
	}
	activeID := state.ActivePlayerID
	targetID := state.PlayerOrder[1]
	if targetID == activeID {
		targetID = state.PlayerOrder[0]
	}
	activeFaction := state.Players[activeID].Faction
	targetFaction := state.Players[targetID].Faction
	_, err = Apply(state, Command{ActorID: activeID, ExpectedVersion: state.Version, Kind: CommandResolveOperation, TargetIDs: []string{activeID}}, now)
	if err != ErrInvalidTarget {
		t.Fatalf("self-target Confession error = %v, want %v", err, ErrInvalidTarget)
	}
	transition, err = Apply(state, Command{ActorID: activeID, ExpectedVersion: state.Version, Kind: CommandResolveOperation, TargetIDs: []string{targetID}}, now)
	if err != nil {
		t.Fatal(err)
	}
	state = transition.State
	if state.Phase != PhaseOperationResult {
		t.Fatalf("phase = %s, want %s", state.Phase, PhaseOperationResult)
	}
	if state.Players[activeID].Faction != activeFaction || state.Players[targetID].Faction != targetFaction {
		t.Fatal("Confession changed faction state")
	}
	recipientResult := Project(state, targetID).Private.OperationResult
	if recipientResult == nil || recipientResult.Code != "AGENCY_SHARED" || recipientResult.OtherPlayerID != activeID || recipientResult.OtherFaction != activeFaction {
		t.Fatalf("recipient result = %#v", recipientResult)
	}
	if Project(state, activeID).Private.OperationResult != nil {
		t.Fatal("Confession delivered an unnecessary private result to the active player")
	}
	if Project(state, state.PlayerOrder[2]).Private.OperationResult != nil {
		t.Fatal("Confession leaked its private result to a bystander")
	}
	public := Project(state, activeID).Public.Operation
	if public == nil || public.TargetCount != 1 {
		t.Fatalf("public operation = %#v, want one private target", public)
	}
}

func TestVoteResolutionIsDeterministicWithShieldAndDoubleVote(t *testing.T) {
	for i := 0; i < 50; i++ {
		state := GameState{
			PlayerOrder: []string{"p1", "p2", "p3", "p4", "p5"},
			Players: map[string]PlayerState{
				"p1": {ID: "p1", Name: "A", Faction: FactionService, CanVote: true, VotingPower: 1},
				"p2": {ID: "p2", Name: "B", Faction: FactionService, CanVote: true, VotingPower: 2, Statuses: []string{"DOUBLE_VOTE"}},
				"p3": {ID: "p3", Name: "C", Faction: FactionVirus, CanVote: true, VotingPower: 1, Statuses: []string{"VOTE_SHIELD"}},
				"p4": {ID: "p4", Name: "D", Faction: FactionService, CanVote: true, VotingPower: 1},
				"p5": {ID: "p5", Name: "E", Faction: FactionVirus, CanVote: true, VotingPower: 1},
			},
			Vote: VoteState{
				Submitted: map[string]string{
					"p1": "p3", // p1 (power 1) votes for p3 (has shield)
					"p2": "p3", // p2 (power 2) votes for p3
					"p4": "p5", // p4 votes for p5
					"p5": "p4", // p5 votes for p4
				},
				Totals: map[string]int{},
			},
		}

		resolveVote(&state)

		// p1 is processed first in PlayerOrder: consumes VOTE_SHIELD on p3 (0 votes added)
		// p2 is processed next in PlayerOrder: adds 2 votes to p3
		// p4 adds 1 vote to p5
		// p5 adds 1 vote to p4
		// Totals: p3 has 2, p4 has 1, p5 has 1
		if state.Vote.Totals["p3"] != 2 {
			t.Fatalf("run %d: p3 totals = %d, want 2", i, state.Vote.Totals["p3"])
		}
		if state.Vote.ImprisonedPlayerID != "p3" {
			t.Fatalf("run %d: imprisoned = %s, want p3", i, state.Vote.ImprisonedPlayerID)
		}
		if state.Winner != FactionService {
			t.Fatalf("run %d: winner = %s, want SERVICE", i, state.Winner)
		}
	}
}

func TestSharedInfoUsesRecipientSpecificTargetIDs(t *testing.T) {
	state := GameState{
		PlayerOrder:    []string{"active", "target", "bystander"},
		ActivePlayerID: "active",
		Players: map[string]PlayerState{
			"active":    {ID: "active", Faction: FactionService},
			"target":    {ID: "target", Faction: FactionVirus},
			"bystander": {ID: "bystander", Faction: FactionService},
		},
		Operation: &OperationState{Kind: "InfoForTwo", ActivePlayerID: "active"},
	}
	resolver, err := operationResolverFor("InfoForTwo")
	if err != nil {
		t.Fatal(err)
	}
	if err := resolver.Resolve(&state, Command{ActorID: "active", TargetID: "target"}); err != nil {
		t.Fatal(err)
	}
	activeResult := state.Operation.PrivateResults["active"]
	targetResult := state.Operation.PrivateResults["target"]
	if activeResult.TargetPlayerID != "target" || targetResult.TargetPlayerID != "active" {
		t.Fatalf("recipient targets = active:%#v target:%#v", activeResult, targetResult)
	}
	if state.Operation.PrivateResults["bystander"].Message != "" {
		t.Fatal("shared-info leaked a result to a bystander")
	}
}

func TestVoteShieldConsumesOneBallotBeforeApplyingVotingPower(t *testing.T) {
	state := GameState{
		PlayerOrder: []string{"voter", "shielded", "other"},
		Players: map[string]PlayerState{
			"voter":    {ID: "voter", Faction: FactionService, VotingPower: 2},
			"shielded": {ID: "shielded", Faction: FactionVirus, Statuses: []string{"VOTE_SHIELD"}},
			"other":    {ID: "other", Faction: FactionService, VotingPower: 1},
		},
		Vote: VoteState{Submitted: map[string]string{"voter": "shielded", "other": "voter"}, Totals: map[string]int{}},
	}
	resolveVote(&state)
	if state.Vote.Totals["shielded"] != 0 || state.Vote.Totals["voter"] != 1 {
		t.Fatalf("shielded totals = %#v, want shielded 0 and voter 1", state.Vote.Totals)
	}
	if hasStatus(state.Players["shielded"], "VOTE_SHIELD") {
		t.Fatal("vote shield was not consumed")
	}
}

func TestTargetWinsEvaluatesCustomObjectiveAndStopsCycles(t *testing.T) {
	state := GameState{
		Winner: FactionService,
		Players: map[string]PlayerState{
			"source": {ID: "source", Faction: FactionVirus, ObjectiveKind: "TARGET_WINS", ObjectiveTarget: "target"},
			"target": {ID: "target", Faction: FactionVirus, ObjectiveKind: "IMPRISON_SELF"},
		},
		Vote: VoteState{ImprisonedPlayerID: "target", Submitted: map[string]string{}, Totals: map[string]int{}},
	}
	if !playerWins(state, "source") {
		t.Fatal("TARGET_WINS did not honor the target's custom objective")
	}
	state.Players["source"] = PlayerState{ID: "source", ObjectiveKind: "TARGET_WINS", ObjectiveTarget: "target"}
	state.Players["target"] = PlayerState{ID: "target", ObjectiveKind: "TARGET_WINS", ObjectiveTarget: "source"}
	state.Vote.ImprisonedPlayerID = ""
	if playerWins(state, "source") {
		t.Fatal("cyclic TARGET_WINS objectives should terminate as false")
	}
}

func TestTieVoteFavorsVirus(t *testing.T) {
	state := GameState{
		PlayerOrder: []string{"p1", "p2", "p3", "p4", "p5"},
		Players: map[string]PlayerState{
			"p1": {ID: "p1", Name: "A", Faction: FactionService, CanVote: true, VotingPower: 1},
			"p2": {ID: "p2", Name: "B", Faction: FactionService, CanVote: true, VotingPower: 1},
			"p3": {ID: "p3", Name: "C", Faction: FactionVirus, CanVote: true, VotingPower: 1},
			"p4": {ID: "p4", Name: "D", Faction: FactionVirus, CanVote: true, VotingPower: 1},
			"p5": {ID: "p5", Name: "E", Faction: FactionService, CanVote: true, VotingPower: 1},
		},
		Vote: VoteState{
			Submitted: map[string]string{
				"p1": "p3",
				"p2": "p3",
				"p3": "p1",
				"p4": "p1",
				"p5": "p2",
			},
			Totals: map[string]int{},
		},
	}
	resolveVote(&state)
	if state.Vote.ImprisonedPlayerID != "" {
		t.Fatalf("imprisoned player on tie = %s, want none", state.Vote.ImprisonedPlayerID)
	}
	if state.Winner != FactionVirus {
		t.Fatalf("winner on tie = %s, want VIRUS", state.Winner)
	}
	for _, entry := range buildLeaderboard(state) {
		if entry.Faction == FactionVirus && entry.Result != "WINNER" {
			t.Fatalf("virus player result on tie = %s, want WINNER", entry.Result)
		}
		if entry.Faction == FactionService && entry.Result != "LOSER" {
			t.Fatalf("service player result on tie = %s, want LOSER", entry.Result)
		}
	}
}

func TestStrainVictoryDefeatsBothAgencies(t *testing.T) {
	state := GameState{
		PlayerOrder: []string{"p1", "p2", "p3", "p4", "p5"},
		Players: map[string]PlayerState{
			"p1": {ID: "p1", Name: "A", Faction: FactionService, ObjectiveKind: "IMPRISON_SELF", CanVote: true, VotingPower: 1},
			"p2": {ID: "p2", Name: "B", Faction: FactionService, CanVote: true, VotingPower: 1},
			"p3": {ID: "p3", Name: "C", Faction: FactionVirus, CanVote: true, VotingPower: 1},
			"p4": {ID: "p4", Name: "D", Faction: FactionVirus, CanVote: true, VotingPower: 1},
			"p5": {ID: "p5", Name: "E", Faction: FactionService, CanVote: true, VotingPower: 1},
		},
		Vote: VoteState{
			Submitted: map[string]string{
				"p1": "p3",
				"p2": "p1",
				"p3": "p1",
				"p4": "p1",
				"p5": "p1",
			},
			Totals: map[string]int{},
		},
	}
	resolveVote(&state)
	if state.Vote.ImprisonedPlayerID != "p1" {
		t.Fatalf("imprisoned player = %s, want p1", state.Vote.ImprisonedPlayerID)
	}
	if state.Winner != FactionNone {
		t.Fatalf("winner on strain = %s, want FactionNone", state.Winner)
	}
	for _, entry := range buildLeaderboard(state) {
		if entry.PlayerID == "p1" && entry.Result != "WINNER" {
			t.Fatalf("strain player result = %s, want WINNER", entry.Result)
		}
		if entry.PlayerID != "p1" && entry.Result != "LOSER" {
			t.Fatalf("other player result = %s, want LOSER", entry.Result)
		}
	}
}

func TestDisconnectedPlayersDoNotBlockRoleRevealOrVoting(t *testing.T) {
	state := NewLobby("room", "p1", "A", DefaultRoomSettings())
	state.Settings.EnabledOperations = map[string]bool{"Swap": true}
	for _, id := range []string{"p2", "p3", "p4", "p5"} {
		if err := state.AddPlayer(id, string(id)); err != nil {
			t.Fatal(err)
		}
	}
	state.Phase = PhaseRoleReveal
	state.PlannedOperation = "Swap"
	state.RoleAcks = map[string]bool{"p1": true, "p2": true, "p3": true, "p4": true}
	for _, id := range state.PlayerOrder {
		player := state.Players[id]
		player.Connected = true
		state.Players[id] = player
	}
	transition, err := ApplyDisconnect(state, "p5", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if transition.State.Phase != PhaseOperationInput || transition.State.ActivePlayerID == "p5" {
		t.Fatalf("role reveal after disconnect = phase %s active %s", transition.State.Phase, transition.State.ActivePlayerID)
	}
	state = transition.State
	state.Phase = PhaseVoteInput
	p5 := state.Players["p5"]
	p5.Connected = true
	state.Players["p5"] = p5
	state.Vote = VoteState{Submitted: map[string]string{"p1": "p2", "p2": "p1", "p3": "p1", "p4": "p1"}, Totals: map[string]int{}}
	transition, err = ApplyDisconnect(state, "p5", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if transition.State.Phase != PhaseResultsIntro {
		t.Fatalf("vote phase after disconnected voter = %s, want %s", transition.State.Phase, PhaseResultsIntro)
	}
}

func TestDiscussionAllPlayersReadyAdvancesToVote(t *testing.T) {
	state := NewLobby("room_disc", "p1", "Player 1", DefaultRoomSettings())
	for i := 2; i <= 5; i++ {
		id := string(fmt.Sprintf("p%d", i))
		state.Players[id] = PlayerState{ID: id, Name: fmt.Sprintf("Player %d", i), Seat: i, Ready: true, Connected: true}
		state.PlayerOrder = append(state.PlayerOrder, id)
	}
	state.Phase = PhaseDiscussion
	state.DiscussionAcks = make(map[string]bool)
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	// Non-host players vote to open accusations one by one
	for i := 2; i <= 4; i++ {
		id := string(fmt.Sprintf("p%d", i))
		transition, err := Apply(state, Command{ActorID: id, ExpectedVersion: state.Version, Kind: CommandAdvanceDiscussion}, now)
		if err != nil {
			t.Fatalf("player %s ready: %v", id, err)
		}
		if transition.State.Phase != PhaseDiscussion {
			t.Fatalf("phase prematurely changed to %s", transition.State.Phase)
		}
		state = transition.State
		proj := Project(state, "p1").Public
		if proj.DiscussionReadyCount != i-1 {
			t.Fatalf("discussion ready count = %d, want %d", proj.DiscussionReadyCount, i-1)
		}
	}

	// Final player votes to open accusations, auto-advancing to vote phase
	transition, err := Apply(state, Command{ActorID: "p5", ExpectedVersion: state.Version, Kind: CommandAdvanceDiscussion}, now)
	if err != nil {
		t.Fatalf("final player ready: %v", err)
	}
	if transition.State.Phase == PhaseDiscussion {
		// Still waiting for p1 (host)
		state = transition.State
		transition, err = Apply(state, Command{ActorID: "p1", ExpectedVersion: state.Version, Kind: CommandAdvanceDiscussion}, now)
		if err != nil {
			t.Fatalf("host ready: %v", err)
		}
	}
	if transition.State.Phase != PhaseVoteInput {
		t.Fatalf("phase = %s, want %s", transition.State.Phase, PhaseVoteInput)
	}
}

func TestDiscussionHostCannotOverrideAlone(t *testing.T) {
	state := NewLobby("room_disc_host", "p1", "Player 1", DefaultRoomSettings())
	for i := 2; i <= 5; i++ {
		id := string(fmt.Sprintf("p%d", i))
		state.Players[id] = PlayerState{ID: id, Name: fmt.Sprintf("Player %d", i), Seat: i, Ready: true, Connected: true}
		state.PlayerOrder = append(state.PlayerOrder, id)
	}
	state.Phase = PhaseDiscussion
	state.DiscussionAcks = make(map[string]bool)
	deadline := time.Date(2026, 8, 15, 12, 5, 0, 0, time.UTC)
	state.DiscussionDeadline = &deadline
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	// Host votes alone while timer is active - should NOT advance to vote input
	transition, err := Apply(state, Command{ActorID: "p1", ExpectedVersion: state.Version, Kind: CommandAdvanceDiscussion}, now)
	if err != nil {
		t.Fatalf("host ready error: %v", err)
	}
	if transition.State.Phase != PhaseDiscussion {
		t.Fatalf("host alone forced advance to %s", transition.State.Phase)
	}
	if !transition.State.DiscussionAcks["p1"] {
		t.Fatal("host ack not recorded")
	}

	// Timer expires - advance should succeed automatically
	afterDeadline := time.Date(2026, 8, 15, 12, 5, 1, 0, time.UTC)
	transition, err = Apply(transition.State, Command{ActorID: "p1", ExpectedVersion: transition.State.Version, Kind: CommandAdvanceDiscussion}, afterDeadline)
	if err != nil {
		t.Fatalf("timer expired advance error: %v", err)
	}
	if transition.State.Phase != PhaseVoteInput {
		t.Fatalf("phase = %s after timer expiry, want %s", transition.State.Phase, PhaseVoteInput)
	}
}

func TestConfigureDiscussionTimerSeconds(t *testing.T) {
	state := NewLobby("room_timer_cfg", "p1", "Player 1", DefaultRoomSettings())
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	// Host configures 7 minutes (420 seconds)
	transition, err := Apply(state, Command{ActorID: "p1", ExpectedVersion: state.Version, Kind: CommandSetDiscussionTimer, DiscussionTimerEnabled: true, DiscussionSeconds: 420}, now)
	if err != nil {
		t.Fatalf("configure timer: %v", err)
	}
	if transition.State.Settings.DiscussionSeconds != 420 {
		t.Fatalf("discussion seconds = %d, want 420", transition.State.Settings.DiscussionSeconds)
	}

	// Non-host cannot configure
	_ = transition.State.AddPlayer("p2", "Player 2")
	_, err = Apply(transition.State, Command{ActorID: "p2", ExpectedVersion: transition.State.Version, Kind: CommandSetDiscussionTimer, DiscussionTimerEnabled: true, DiscussionSeconds: 600}, now)
	if err != ErrNotAllowed {
		t.Fatalf("non-host err = %v, want %v", err, ErrNotAllowed)
	}
}

func TestSetVirusCountInSoloLobby(t *testing.T) {
	state := NewLobby("room_virus_cfg", "p1", "Player 1", DefaultRoomSettings())
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	// Host configures 2 VIRUS when only 1 player is in the lobby
	transition, err := Apply(state, Command{ActorID: "p1", ExpectedVersion: state.Version, Kind: CommandSetVirusCount, VirusCount: 2}, now)
	if err != nil {
		t.Fatalf("set virus count failed in solo lobby: %v", err)
	}
	if transition.State.Settings.VirusCount != 2 {
		t.Fatalf("virus count = %d, want 2", transition.State.Settings.VirusCount)
	}

	// Host resets to Auto Standard (0)
	transition, err = Apply(transition.State, Command{ActorID: "p1", ExpectedVersion: transition.State.Version, Kind: CommandSetVirusCount, VirusCount: 0}, now)
	if err != nil {
		t.Fatalf("reset virus count to 0 failed: %v", err)
	}
	if transition.State.Settings.VirusCount != 0 {
		t.Fatalf("virus count = %d, want 0", transition.State.Settings.VirusCount)
	}
}

func fivePlayerMatch(t *testing.T) GameState {
	t.Helper()
	state := NewLobby("room_test", "p1", "Agent A", DefaultRoomSettings())
	for i := 2; i <= 5; i++ {
		id := string("p" + string(rune('0'+i)))
		if err := state.AddPlayer(id, "Agent "+string(rune('A'+i-1))); err != nil {
			t.Fatal(err)
		}
	}
	assignRoles(&state)
	return state
}

func TestChooseVoteShieldTargetCanFinishOperation(t *testing.T) {
	state := fivePlayerMatch(t)
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	// Force ChooseVoteShield operation
	state.PlannedOperation = "ChooseVoteShield"
	state.ActivePlayerID = "p1"
	if err := beginPlannedOperation(&state); err != nil {
		t.Fatalf("beginPlannedOperation error: %v", err)
	}
	if state.Operation.Step != 1 || state.Operation.InputOwnerID != "p1" {
		t.Fatalf("step = %d, inputOwner = %s", state.Operation.Step, state.Operation.InputOwnerID)
	}

	// Step 1: Active player (p1) selects target (p2)
	transition, err := Apply(state, Command{
		ActorID:         "p1",
		ExpectedVersion: state.Version,
		Kind:            CommandResolveOperation,
		TargetIDs:       []string{"p2"},
	}, now)
	if err != nil {
		t.Fatalf("step 1 resolve error: %v", err)
	}
	state = transition.State
	if state.Phase != PhaseOperationInput || state.Operation.Step != 2 || state.Operation.InputOwnerID != "p2" {
		t.Fatalf("after step 1: phase=%s, step=%d, inputOwner=%s", state.Phase, state.Operation.Step, state.Operation.InputOwnerID)
	}

	// Step 2: Target player (p2) chooses EXTRA_SUSPICION
	transition, err = Apply(state, Command{
		ActorID:         "p2",
		ExpectedVersion: state.Version,
		Kind:            CommandResolveOperation,
		Choice:          "EXTRA_SUSPICION",
	}, now)
	if err != nil {
		t.Fatalf("step 2 resolve error: %v", err)
	}
	state = transition.State
	if state.Phase != PhaseOperationResult {
		t.Fatalf("phase after step 2 = %s, want %s", state.Phase, PhaseOperationResult)
	}

	// Verify projection allows target player (p2) to submit Done
	proj := Project(state, "p2")
	if !proj.Private.CanSubmit {
		t.Fatal("target player (p2) projection can_submit is false during PhaseOperationResult")
	}

	// Target player (p2) clicks Done
	transition, err = Apply(state, Command{
		ActorID:         "p2",
		ExpectedVersion: state.Version,
		Kind:            CommandOperationExplainDone,
	}, now)
	if err != nil {
		t.Fatalf("target player CommandOperationExplainDone failed: %v", err)
	}
	if transition.State.Phase != PhaseOperationInterlude {
		t.Fatalf("phase after explain done = %s, want %s", transition.State.Phase, PhaseOperationInterlude)
	}
}

func TestMultipleExtraSuspicionModifiers(t *testing.T) {
	state := fivePlayerMatch(t)
	p1 := state.Players["p1"]
	p1.Statuses = append(p1.Statuses, "EXTRA_SUSPICION", "EXTRA_SUSPICION")
	state.Players["p1"] = p1

	// No one votes for p1, but p1 has 2 EXTRA_SUSPICION statuses
	state.Vote = VoteState{
		Submitted: map[string]string{
			"p1": "p2",
			"p2": "p2",
			"p3": "p2",
			"p4": "p2",
			"p5": "p2",
		},
		Totals: map[string]int{},
	}
	resolveVote(&state)

	if state.Vote.Totals["p1"] != 2 {
		t.Fatalf("p1 totals with 2 extra suspicion = %d, want 2", state.Vote.Totals["p1"])
	}
}
