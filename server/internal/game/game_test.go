package game

import (
	"errors"
	"testing"
	"time"
)

func testLobby(t *testing.T, count int) State {
	t.Helper()
	state := NewLobby("p1", "P1")
	state.Settings.MinPlayers = count
	state.RandomState = 1
	for i := 2; i <= count; i++ {
		id := "p" + string(rune('0'+i))
		var err error
		state, err = AddPlayer(state, id, id)
		if err != nil {
			t.Fatal(err)
		}
		state, err = Connect(state, id)
		if err != nil {
			t.Fatal(err)
		}
	}
	return state
}

func readyAll(t *testing.T, state State) State {
	t.Helper()
	var err error
	for _, id := range state.PlayerOrder {
		state, err = Apply(state, id, Command{Kind: CommandSetReady}, time.Unix(100, 0).UTC())
		if err != nil {
			t.Fatal(err)
		}
	}
	return state
}

func TestLiveOperationRegistryContainsOnlyExecutableOperations(t *testing.T) {
	if len(operationOrder) != 15 {
		t.Fatalf("got %d live operations, want 15", len(operationOrder))
	}
	dead := []string{"Injection", "Power", "Vote", "Confirm", "NegativeVote", "Brig", "EarlyVote", "Hunter", "Ambassador", "LastEvent"}
	for _, id := range dead {
		if _, ok := operationDefinitionFor(id); ok {
			t.Fatalf("dead operation %q remains executable", id)
		}
	}
	for _, id := range operationOrder {
		definition := operations[id].definition
		op, ok := operations[definition.ID]
		if !ok || op.begin == nil {
			t.Fatalf("live operation %q cannot begin", definition.ID)
		}
		if definition.InputKind != OperationInputPrivateInfo && definition.InputKind != OperationInputNone && op.resolve == nil {
			t.Fatalf("input operation %q cannot resolve", definition.ID)
		}
	}
}

func TestPlayerSeatsAreDerivedFromOrder(t *testing.T) {
	state := testLobby(t, 3)
	state, err := Leave(state, "p2")
	if err != nil {
		t.Fatal(err)
	}
	projection := PublicProjectionFor("room_test", state)
	if len(projection.Players) != 2 || projection.Players[0].Seat != 1 || projection.Players[1].Seat != 2 {
		t.Fatalf("seats not derived from player order: %+v", projection.Players)
	}
	if projection.Players[1].ID != "p3" {
		t.Fatalf("unexpected remaining order: %+v", projection.Players)
	}
}

func TestDuplicatePlayerIDIsRejected(t *testing.T) {
	state := NewLobby("p1", "Host")
	_, err := AddPlayer(state, "p1", "Again")
	if !errors.Is(err, ErrPlayerExists) {
		t.Fatalf("got %v, want ErrPlayerExists", err)
	}
}

func TestAddPlayerRejectsDuplicateNames(t *testing.T) {
	state := NewLobby("p1", "Host")
	state, err := AddPlayer(state, "p2", "Guest")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AddPlayer(state, "p3", "host"); !errors.Is(err, ErrNameTaken) {
		t.Fatalf("case-insensitive host clash: got %v, want ErrNameTaken", err)
	}
	if _, err := AddPlayer(state, "p3", "  GUEST  "); !errors.Is(err, ErrNameTaken) {
		t.Fatalf("padded-name clash: got %v, want ErrNameTaken", err)
	}

	state, err = Leave(state, "p2")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AddPlayer(state, "p3", "guest"); err != nil {
		t.Fatalf("name should be free after leave: %v", err)
	}
}

func TestReadyDoesNotCreatePresence(t *testing.T) {
	state := NewLobby("p1", "Host")
	state, err := AddPlayer(state, "p2", "P2")
	if err != nil {
		t.Fatal(err)
	}
	before := state.Version
	state, err = Apply(state, "p2", Command{Kind: CommandSetReady}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if state.Players["p2"].Connected {
		t.Fatal("ready command incorrectly marked disconnected player connected")
	}
	if state.Version != before+1 {
		t.Fatalf("version = %d, want %d", state.Version, before+1)
	}
}

func TestConnectAndDisconnectAreVersioned(t *testing.T) {
	state := NewLobby("p1", "Host")
	state, err := AddPlayer(state, "p2", "P2")
	if err != nil {
		t.Fatal(err)
	}
	before := state.Version
	state, err = Connect(state, "p2")
	if err != nil {
		t.Fatal(err)
	}
	if !state.Players["p2"].Connected || state.Version != before+1 {
		t.Fatalf("connect did not produce one visible version: %+v", state.Players["p2"])
	}
	before = state.Version
	state, err = Disconnect(state, "p2", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !HasPlayer(state, "p2") || state.Players["p2"].Connected {
		t.Fatal("lobby disconnect should retain the seat as disconnected")
	}
	if state.Version != before+1 {
		t.Fatalf("disconnect version = %d, want %d", state.Version, before+1)
	}
}

func TestHostDisconnectTransfersHostInGame(t *testing.T) {
	state := testLobby(t, 3)
	state.Phase = PhaseDiscussion
	state.DiscussionAcks = map[string]bool{}
	before := state.Version
	state, err := Disconnect(state, "p1", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if state.HostID != "p2" {
		t.Fatalf("host = %q, want p2", state.HostID)
	}
	if state.Players["p1"].Connected {
		t.Fatal("old host still connected")
	}
	if state.Version != before+1 {
		t.Fatalf("version = %d, want %d", state.Version, before+1)
	}
}

func TestStartAndRoleAcksBeginOperation(t *testing.T) {
	state := readyAll(t, testLobby(t, 5))
	var err error
	state, err = Apply(state, "p1", Command{Kind: CommandStartMatch, OperationKind: "Detector"}, time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	if state.Phase != PhaseRoleReveal || state.PlannedOperation != "Detector" {
		t.Fatalf("unexpected start state: phase=%s op=%s", state.Phase, state.PlannedOperation)
	}
	for _, id := range state.PlayerOrder {
		state, err = Apply(state, id, Command{Kind: CommandAcknowledgeRole}, time.Unix(100, 0))
		if err != nil {
			t.Fatal(err)
		}
	}
	if state.Phase != PhaseOperationInput || state.Operation == nil || state.Operation.Kind != "Detector" {
		t.Fatalf("operation did not begin: phase=%s op=%+v", state.Phase, state.Operation)
	}
}

func TestOperationDealingGivesEveryPlayerExactlyOneOperation(t *testing.T) {
	fixture := newGameFixture(t, fixtureOptions{PlayerCount: 5})
	for index, id := range fixture.state.PlayerOrder {
		if index%2 == 0 {
			fixture.SetFaction(id, FactionService)
		} else {
			fixture.SetFaction(id, FactionVirus)
		}
	}
	fixture.DealAllOperations()
	state := fixture.State()
	if state.OperationDeals != len(state.PlayerOrder) {
		t.Fatalf("dealt %d operations, want %d", state.OperationDeals, len(state.PlayerOrder))
	}
	for _, id := range state.PlayerOrder {
		if got := len(state.Players[id].DealtOperations); got != 1 {
			t.Fatalf("player %s received %d operations, want 1", id, got)
		}
	}
}

func TestHiddenAgendaCountsAsOneOperationSlot(t *testing.T) {
	state := newGameFixture(t, fixtureOptions{PlayerCount: 5}).State()
	hiddenSlots := 0
	for _, slot := range operationDeckSlots(state) {
		if slot == hiddenAgendaKind {
			hiddenSlots++
		}
	}
	if hiddenSlots != 1 {
		t.Fatalf("hidden agenda contributes %d operation slots, want 1", hiddenSlots)
	}
}

func TestOperationDealingRefillsASmallPoolForAllPlayers(t *testing.T) {
	fixture := newGameFixture(t, fixtureOptions{PlayerCount: 5, EnabledOperations: []string{"Share"}})
	fixture.DealAllOperations()
	state := fixture.State()
	if state.OperationDeals != len(state.PlayerOrder) {
		t.Fatalf("dealt %d operations, want %d", state.OperationDeals, len(state.PlayerOrder))
	}
	for _, id := range state.PlayerOrder {
		dealt := state.Players[id].DealtOperations
		if len(dealt) != 1 || dealt[0] != "Share" {
			t.Fatalf("player %s received operations %v, want [Share]", id, dealt)
		}
	}
}

func TestOperationDealingCountsOnlyConnectedPlayers(t *testing.T) {
	fixture := newGameFixture(t, fixtureOptions{PlayerCount: 5, EnabledOperations: []string{"Share"}})
	fixture.state.Players["p5"] = func() Player {
		player := fixture.state.Players["p5"]
		player.Connected = false
		return player
	}()
	fixture.DealAllOperations()
	state := fixture.State()
	if state.OperationDeals != 4 {
		t.Fatalf("dealt %d operations, want 4 connected players", state.OperationDeals)
	}
	if len(state.Players["p5"].DealtOperations) != 0 {
		t.Fatalf("disconnected player received operations: %v", state.Players["p5"].DealtOperations)
	}
	for _, id := range []string{"p1", "p2", "p3", "p4"} {
		if got := len(state.Players[id].DealtOperations); got != 1 {
			t.Fatalf("player %s received %d operations, want 1", id, got)
		}
	}
}

func TestDanishIntelligenceSelectsOneVirusAndOneNonVirus(t *testing.T) {
	fixture := newGameFixture(t, fixtureOptions{PlayerCount: 4, Seed: 2})
	fixture.StartMatch("OneOfTwo")
	for _, id := range []string{"p1", "p4"} {
		fixture.SetFaction(id, FactionService)
	}
	for _, id := range []string{"p2", "p3"} {
		fixture.SetFaction(id, FactionVirus)
	}
	fixture.AcknowledgeRoles()
	state := fixture.State()
	activeID := state.ActivePlayerID
	result := state.Operation.PrivateResults[activeID]
	if result.Code != "ONE_VIRUS_ONE_SERVICE" {
		t.Fatalf("code = %q, want ONE_VIRUS_ONE_SERVICE; result=%+v", result.Code, result)
	}
	if len(result.TargetPlayerIDs) != 2 {
		t.Fatalf("selected %d players, want 2: %v", len(result.TargetPlayerIDs), result.TargetPlayerIDs)
	}

	virusCount := 0
	serviceCount := 0
	for _, id := range result.TargetPlayerIDs {
		switch checkFaction(state.Players[id]) {
		case FactionVirus:
			virusCount++
		case FactionService:
			serviceCount++
		}
	}
	if virusCount != 1 || serviceCount != 1 {
		t.Fatalf("selected factions = VIRUS:%d SERVICE:%d; IDs=%v", virusCount, serviceCount, result.TargetPlayerIDs)
	}
}

func TestGameFixtureResolvesDetectorThroughCommands(t *testing.T) {
	fixture := newGameFixture(t, fixtureOptions{PlayerCount: 5})
	fixture.StartMatch("Detector")
	targets := make([]string, 0, 2)
	for _, id := range fixture.state.PlayerOrder {
		if id == fixture.state.ActivePlayerID {
			fixture.SetFaction(id, FactionService)
		} else {
			fixture.SetFaction(id, FactionService)
			if len(targets) < 2 {
				targets = append(targets, id)
			}
		}
	}
	fixture.SetFaction(targets[0], FactionVirus)
	fixture.AcknowledgeRoles()
	fixture.Resolve(Command{Kind: CommandResolveOperation, TargetIDs: targets})

	state := fixture.State()
	if state.Phase != PhaseOperationResult {
		t.Fatalf("phase = %s, want %s", state.Phase, PhaseOperationResult)
	}
	result := state.Operation.PrivateResults[state.ActivePlayerID]
	if result.Code != "AT_LEAST_ONE_VIRUS" {
		t.Fatalf("code = %q, want AT_LEAST_ONE_VIRUS", result.Code)
	}
}

func TestGameFixtureCanOverrideRoleBeforeOperationBegins(t *testing.T) {
	fixture := newGameFixture(t, fixtureOptions{PlayerCount: 5})
	fixture.StartMatch("Flip")
	activeID := fixture.state.ActivePlayerID
	fixture.SetRole(activeID, RoleLoyalRed, FactionVirus)
	fixture.AcknowledgeRoles()

	state := fixture.State()
	result := state.Operation.PrivateResults[activeID]
	if result.Code != "AGENCY_HELD" {
		t.Fatalf("code = %q, want AGENCY_HELD", result.Code)
	}
}

func TestAdvanceDeadlineDoesNotPretendHostActed(t *testing.T) {
	state := testLobby(t, 3)
	state.Phase = PhaseDiscussion
	deadline := time.Unix(100, 0).UTC()
	state.DiscussionDeadline = &deadline
	state.DiscussionAcks = map[string]bool{}
	before := state.Version
	next, err := AdvanceDeadline(state, deadline.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if next.Phase != PhaseVoteInput {
		t.Fatalf("phase = %s, want vote input", next.Phase)
	}
	if next.Version != before+1 {
		t.Fatalf("version = %d, want %d", next.Version, before+1)
	}
}

func TestProjectionKeepsPrivateResultsPrivate(t *testing.T) {
	state := testLobby(t, 3)
	state.Phase = PhaseOperationResult
	state.ActivePlayerID = "p1"
	state.Operation = &OperationState{Kind: "OneRandom", InputOwnerID: "p1", Step: 1, PrivateResults: map[string]OperationResult{
		"p1": {Code: "FACTION_REVEALED", TargetPlayerID: "p2", TargetFaction: FactionVirus, Message: "secret"},
	}}
	for id, player := range state.Players {
		player.Faction = FactionService
		player.InitialFaction = FactionService
		state.Players[id] = player
	}
	p1 := Project("room", state, "p1")
	p2 := Project("room", state, "p2")
	if p1.Private.OperationResult == nil {
		t.Fatal("owner did not receive private result")
	}
	if p2.Private.OperationResult != nil {
		t.Fatalf("private result leaked to p2: %+v", p2.Private.OperationResult)
	}
	if p2.Public.Operation == nil || p2.Public.Operation.Kind != "OneRandom" {
		t.Fatalf("public operation malformed: %+v", p2.Public.Operation)
	}
}

func TestHiddenAgendaProjectionHidesRealOperation(t *testing.T) {
	state := testLobby(t, 5)
	state.Phase = PhaseOperationResult
	state.ActivePlayerID = "p1"
	state.Operation = &OperationState{Kind: "Grudge", InputOwnerID: "p1", Step: 1}
	public := PublicProjectionFor("room", state)
	if public.Operation == nil || public.Operation.Kind != hiddenAgendaKind || public.Operation.Name != hiddenAgendaName {
		t.Fatalf("hidden operation leaked publicly: %+v", public.Operation)
	}
	private := ProjectWithPublic(state, "p1", public)
	if private.Private.OperationKind != "Grudge" || private.Private.OperationName != "Grudge" {
		t.Fatalf("recipient did not get hidden identity: %+v", private.Private)
	}
}

func TestChooseVoteShieldTransfersInputOwnership(t *testing.T) {
	state := testLobby(t, 5)
	state.Phase = PhaseOperationInput
	state.ActivePlayerID = "p1"
	state.Operation = &OperationState{Kind: "ChooseVoteShield", InputOwnerID: "p1", Step: 1}
	before := state.Version
	next, err := Apply(state, "p1", Command{Kind: CommandResolveOperation, TargetID: "p2"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if next.Phase != PhaseOperationInput || next.Operation.Step != 2 || next.Operation.InputOwnerID != "p2" {
		t.Fatalf("step one did not transfer input: %+v", next.Operation)
	}
	if next.Version != before+1 {
		t.Fatalf("step one version = %d, want %d", next.Version, before+1)
	}
	next, err = Apply(next, "p2", Command{Kind: CommandResolveOperation, Choice: "VOTE_SHIELD"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if next.Phase != PhaseOperationResult || !hasStatus(next.Players["p1"], "VOTE_SHIELD") {
		t.Fatalf("step two did not resolve: phase=%s player=%+v", next.Phase, next.Players["p1"])
	}
}

func TestVoteShieldConsumesWholePoweredBallot(t *testing.T) {
	state := testLobby(t, 3)
	state.Phase = PhaseVoteInput
	state.Vote = VoteState{Submitted: map[string]string{"p1": "p2", "p2": "p1", "p3": "p1"}, Totals: map[string]int{}}
	p1 := state.Players["p1"]
	p1.VotingPower = 2
	state.Players["p1"] = p1
	p2 := state.Players["p2"]
	p2.Statuses = append(p2.Statuses, "VOTE_SHIELD")
	state.Players["p2"] = p2
	state.resolveVote()
	if state.Vote.Totals["p2"] != 0 {
		t.Fatalf("shielded target got %d votes, want 0", state.Vote.Totals["p2"])
	}
	if hasStatus(state.Players["p2"], "VOTE_SHIELD") {
		t.Fatal("vote shield was not consumed")
	}
}

func TestRematchResetsRoundButKeepsPresence(t *testing.T) {
	state := testLobby(t, 3)
	state.Phase = PhaseEnd
	for id, player := range state.Players {
		player.Ready = true
		player.Faction = FactionVirus
		player.InitialFaction = FactionVirus
		player.Statuses = []string{"X"}
		state.Players[id] = player
	}
	next, err := Apply(state, "p1", Command{Kind: CommandRematch}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if next.Phase != PhaseLobby {
		t.Fatalf("phase = %s", next.Phase)
	}
	for _, player := range next.Players {
		if !player.Connected || player.Ready || player.Faction != "" || len(player.Statuses) != 0 {
			t.Fatalf("player not reset correctly: %+v", player)
		}
	}
}

func TestEveryLiveOperationExecutesItsRuntimeContract(t *testing.T) {
	for _, id := range operationOrder {
		t.Run(id, func(t *testing.T) {
			state := testLobby(t, 5)
			for index, playerID := range state.PlayerOrder {
				player := state.Players[playerID]
				if index%2 == 0 {
					player.InitialFaction = FactionService
					player.Faction = FactionService
					player.Role = RoleNormalBlue
				} else {
					player.InitialFaction = FactionVirus
					player.Faction = FactionVirus
					player.Role = RoleNormalRed
				}
				state.Players[playerID] = player
			}
			state.ActivePlayerID = "p1"
			op := operations[id]
			if err := op.begin(&state); err != nil {
				t.Fatalf("begin: %v", err)
			}
			if state.Operation == nil || state.Operation.Kind != id {
				t.Fatalf("begin produced %+v", state.Operation)
			}
			if op.resolve == nil {
				if op.definition.InputKind != OperationInputPrivateInfo && op.definition.InputKind != OperationInputNone {
					t.Fatalf("input operation has no resolver")
				}
				return
			}

			command := Command{}
			switch op.definition.InputKind {
			case OperationInputOneTarget:
				command.TargetID = "p2"
			case OperationInputTwoTargets:
				command.TargetIDs = []string{"p2", "p3"}
			case OperationInputChoice:
				command.Choice = "STAY"
			}
			if err := op.resolve(&state, command); err != nil {
				t.Fatalf("resolve: %v", err)
			}
			if id == "ChooseVoteShield" {
				if state.Operation.Step != 2 || state.Operation.InputOwnerID != "p2" {
					t.Fatalf("step one = %+v", state.Operation)
				}
				if err := op.resolve(&state, Command{Choice: "VOTE_SHIELD"}); err != nil {
					t.Fatalf("resolve step two: %v", err)
				}
			}
		})
	}
}

func TestLobbyHostCommandsAndNoOps(t *testing.T) {
	state := testLobby(t, 5)
	now := time.Unix(100, 0).UTC()

	before := state.Version
	next, err := Apply(state, "p1", Command{Kind: CommandTransferHost, TargetID: "p2"}, now)
	if err != nil || next.HostID != "p2" || next.Version != before+1 {
		t.Fatalf("transfer host: state=%+v err=%v", next, err)
	}
	state = next
	before = state.Version
	state, err = Apply(state, "p2", Command{Kind: CommandTransferHost, TargetID: "p2"}, now)
	if err != nil || state.Version != before {
		t.Fatalf("same-host transfer should be no-op: version=%d err=%v", state.Version, err)
	}

	state, err = Apply(state, "p2", Command{Kind: CommandSetDiscussionTimer, DiscussionTimerEnabled: true, DiscussionSeconds: 1}, now)
	if err != nil || state.Settings.DiscussionSeconds != 60 {
		t.Fatalf("discussion clamp: seconds=%d err=%v", state.Settings.DiscussionSeconds, err)
	}
	state, err = Apply(state, "p2", Command{Kind: CommandSetVirusCount, VirusCount: 2}, now)
	if err != nil || state.Settings.VirusCount != 2 {
		t.Fatalf("virus count: %d err=%v", state.Settings.VirusCount, err)
	}
	state, err = Apply(state, "p2", Command{Kind: CommandSetRoleEnabled, RoleID: string(RoleFakeBlue), RoleEnabled: true}, now)
	if err != nil || !state.Settings.EnabledRoles[string(RoleFakeBlue)] {
		t.Fatalf("role toggle failed: %v", err)
	}
	state, err = Apply(state, "p2", Command{Kind: CommandSetOperationEnabled, OperationKind: "Swap", OperationEnabled: true}, now)
	if err != nil || !state.Settings.EnabledOperations["Swap"] {
		t.Fatalf("operation toggle failed: %v", err)
	}
	state, err = Apply(state, "p2", Command{Kind: CommandSetOperationEnabled, OperationKind: hiddenAgendaKind, OperationEnabled: false}, now)
	if err != nil {
		t.Fatalf("hidden agenda toggle: %v", err)
	}
	for _, id := range hiddenAgendaMemberIDs() {
		if state.Settings.EnabledOperations[id] {
			t.Fatalf("hidden member %s remained enabled", id)
		}
	}

	state, err = Apply(state, "p2", Command{Kind: CommandKickPlayer, TargetID: "p5"}, now)
	if err != nil || HasPlayer(state, "p5") {
		t.Fatalf("kick failed: err=%v players=%v", err, state.PlayerOrder)
	}
}

func TestResultsProgressionAndRematch(t *testing.T) {
	state := testLobby(t, 3)
	state.Phase = PhaseResultsIntro
	phases := []Phase{PhaseVoteResults, PhaseImprisonment, PhaseAgencyReveal, PhaseOutcomeReveal, PhaseLeaderboard, PhaseOutOfLoop, PhaseEnd}
	for _, want := range phases {
		before := state.Version
		next, err := Apply(state, state.HostID, Command{Kind: CommandContinueResults}, time.Now())
		if err != nil {
			t.Fatalf("continue from %s: %v", state.Phase, err)
		}
		if next.Phase != want || next.Version != before+1 {
			t.Fatalf("got phase=%s version=%d want phase=%s version=%d", next.Phase, next.Version, want, before+1)
		}
		state = next
	}
	state.Players["p2"] = func() Player { p := state.Players["p2"]; p.Connected = false; return p }()
	state, err := Apply(state, state.HostID, Command{Kind: CommandRematch}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if state.Phase != PhaseLobby || state.Players["p2"].Connected {
		t.Fatalf("rematch did not preserve presence/reset phase: phase=%s p2=%+v", state.Phase, state.Players["p2"])
	}
}

func TestStartRejectsDisconnectedOrUnreadyPlayers(t *testing.T) {
	state := testLobby(t, 5)
	for _, id := range state.PlayerOrder {
		player := state.Players[id]
		player.Ready = true
		state.Players[id] = player
	}
	player := state.Players["p3"]
	player.Connected = false
	state.Players["p3"] = player
	if _, err := Apply(state, "p1", Command{Kind: CommandStartMatch}, time.Now()); !errors.Is(err, ErrNotReady) {
		t.Fatalf("got %v, want ErrNotReady", err)
	}
}

func TestDisconnectDuringOperationInputAndResultPreservesTurn(t *testing.T) {
	fixture := newGameFixture(t, fixtureOptions{PlayerCount: 5})
	fixture.StartMatch("Detector")
	for _, id := range fixture.state.PlayerOrder {
		fixture.SetFaction(id, FactionService)
	}
	fixture.AcknowledgeRoles()

	state := fixture.State()
	if state.Phase != PhaseOperationInput || state.Operation == nil {
		t.Fatalf("expected PhaseOperationInput with active operation, got phase=%s, op=%+v", state.Phase, state.Operation)
	}
	activeID := state.ActivePlayerID
	if activeID == "" {
		t.Fatal("expected non-empty ActivePlayerID")
	}

	// 1. Disconnect active player during PhaseOperationInput
	var err error
	state, err = Disconnect(state, activeID, time.Now())
	if err != nil {
		t.Fatalf("disconnect failed: %v", err)
	}
	if state.Players[activeID].Connected {
		t.Fatal("expected active player to be disconnected")
	}
	if state.Phase != PhaseOperationInput {
		t.Fatalf("expected PhaseOperationInput to be preserved, got %s", state.Phase)
	}
	if state.ActivePlayerID != activeID {
		t.Fatalf("expected ActivePlayerID %s, got %s", activeID, state.ActivePlayerID)
	}
	if state.Operation == nil || state.Operation.Kind != "Detector" {
		t.Fatalf("expected Operation to be preserved, got %+v", state.Operation)
	}

	// 2. Reconnect active player and resolve operation
	state, err = Connect(state, activeID)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	if !state.Players[activeID].Connected {
		t.Fatal("expected active player to be reconnected")
	}

	targets := make([]string, 0, 2)
	for _, id := range state.PlayerOrder {
		if id != activeID {
			targets = append(targets, id)
			if len(targets) == 2 {
				break
			}
		}
	}
	state, err = Apply(state, activeID, Command{Kind: CommandResolveOperation, TargetIDs: targets}, time.Now())
	if err != nil {
		t.Fatalf("resolve operation failed: %v", err)
	}
	if state.Phase != PhaseOperationResult {
		t.Fatalf("expected PhaseOperationResult, got %s", state.Phase)
	}
	if len(state.Operation.PrivateResults) == 0 {
		t.Fatal("expected private results to be populated")
	}

	// 3. Disconnect active player during PhaseOperationResult
	state, err = Disconnect(state, activeID, time.Now())
	if err != nil {
		t.Fatalf("disconnect during PhaseOperationResult failed: %v", err)
	}
	if state.Phase != PhaseOperationResult {
		t.Fatalf("expected PhaseOperationResult to be preserved on disconnect, got %s", state.Phase)
	}
	if state.ActivePlayerID != activeID {
		t.Fatalf("expected ActivePlayerID %s preserved, got %s", activeID, state.ActivePlayerID)
	}
	if len(state.Operation.PrivateResults) == 0 {
		t.Fatal("expected PrivateResults to be preserved on disconnect")
	}

	// 4. Reconnect active player and finish operation
	state, err = Connect(state, activeID)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	state, err = Apply(state, activeID, Command{Kind: CommandOperationExplainDone}, time.Now())
	if err != nil {
		t.Fatalf("operation explain done failed: %v", err)
	}
	if state.Phase != PhaseOperationInterlude {
		t.Fatalf("expected PhaseOperationInterlude, got %s", state.Phase)
	}
}

func TestDisconnectDuringRoleRevealDoesNotAutoAdvance(t *testing.T) {
	fixture := newGameFixture(t, fixtureOptions{PlayerCount: 5})
	fixture.StartMatch("Detector")
	state := fixture.State()
	if state.Phase != PhaseRoleReveal {
		t.Fatalf("expected PhaseRoleReveal, got %s", state.Phase)
	}

	// Disconnect p5
	var err error
	state, err = Disconnect(state, "p5", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	// All other players acknowledge
	for _, id := range []string{"p1", "p2", "p3", "p4"} {
		state, err = Apply(state, id, Command{Kind: CommandAcknowledgeRole}, time.Now())
		if err != nil {
			t.Fatal(err)
		}
	}

	// Phase should STILL be PhaseRoleReveal because p5 has not acknowledged
	if state.Phase != PhaseRoleReveal {
		t.Fatalf("phase advanced prematurely to %s without p5 ack", state.Phase)
	}

	// Reconnect p5 and ack
	state, err = Connect(state, "p5")
	if err != nil {
		t.Fatal(err)
	}
	state, err = Apply(state, "p5", Command{Kind: CommandAcknowledgeRole}, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	if state.Phase != PhaseOperationInput {
		t.Fatalf("expected phase to advance to PhaseOperationInput after all acks, got %s", state.Phase)
	}
}

func TestDisconnectDuringVoteInputDoesNotAutoAdvance(t *testing.T) {
	state := testLobby(t, 3)
	state.Phase = PhaseVoteInput
	state.Vote = VoteState{Submitted: map[string]string{}, Totals: map[string]int{}}

	// Disconnect p3
	var err error
	state, err = Disconnect(state, "p3", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	// p1 and p2 vote
	state, err = Apply(state, "p1", Command{Kind: CommandSubmitVote, TargetID: "p2"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	state, err = Apply(state, "p2", Command{Kind: CommandSubmitVote, TargetID: "p1"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	// Phase should STILL be PhaseVoteInput
	if state.Phase != PhaseVoteInput {
		t.Fatalf("phase advanced prematurely to %s without p3 vote", state.Phase)
	}

	// Reconnect p3 and vote
	state, err = Connect(state, "p3")
	if err != nil {
		t.Fatal(err)
	}
	state, err = Apply(state, "p3", Command{Kind: CommandSubmitVote, TargetID: "p1"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	if state.Phase != PhaseEnd {
		t.Fatalf("expected PhaseEnd after all votes submitted, got %s", state.Phase)
	}
}

func TestDisconnectDuringDiscussionDoesNotAutoAdvance(t *testing.T) {
	state := testLobby(t, 3)
	state.Phase = PhaseDiscussion
	state.DiscussionAcks = map[string]bool{}

	// Disconnect p3
	var err error
	state, err = Disconnect(state, "p3", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	// p1 and p2 advance discussion
	state, err = Apply(state, "p1", Command{Kind: CommandAdvanceDiscussion}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	state, err = Apply(state, "p2", Command{Kind: CommandAdvanceDiscussion}, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	// Phase should STILL be PhaseDiscussion
	if state.Phase != PhaseDiscussion {
		t.Fatalf("phase advanced prematurely to %s without p3 discussion ack", state.Phase)
	}

	// Reconnect p3 and advance discussion
	state, err = Connect(state, "p3")
	if err != nil {
		t.Fatal(err)
	}
	state, err = Apply(state, "p3", Command{Kind: CommandAdvanceDiscussion}, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	if state.Phase != PhaseVoteInput {
		t.Fatalf("expected PhaseVoteInput after all discussion acks, got %s", state.Phase)
	}
}

func TestVoteKickRequiresUnanimousConnectedVotes(t *testing.T) {
	state := testLobby(t, 4) // p1, p2, p3, p4
	var err error

	// Disconnect p4
	state, err = Disconnect(state, "p4", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	// Connected players: p1, p2, p3 (total 3)
	// Check projection has vote kicks
	proj := PublicProjectionFor("r1", state)
	if len(proj.VoteKicks) != 1 || proj.VoteKicks[0].TargetID != "p4" || proj.VoteKicks[0].Required != 3 || proj.VoteKicks[0].Votes != 0 {
		t.Fatalf("unexpected projection vote kicks: %+v", proj.VoteKicks)
	}

	// 1. p1 votes to kick p4 (1/3)
	state, err = Apply(state, "p1", Command{Kind: CommandVoteKick, TargetID: "p4"}, time.Now())
	if err != nil {
		t.Fatalf("p1 vote kick failed: %v", err)
	}
	if !HasPlayer(state, "p4") {
		t.Fatal("p4 was kicked prematurely with 1/3 votes")
	}

	p1Proj := Project("r1", state, "p1")
	p2Proj := Project("r1", state, "p2")
	if len(p1Proj.Public.VoteKicks) != 1 || p1Proj.Public.VoteKicks[0].Votes != 1 || !p1Proj.Public.VoteKicks[0].HasVoted {
		t.Fatalf("expected p1 to have voted with 1 vote: %+v", p1Proj.Public.VoteKicks)
	}
	if len(p2Proj.Public.VoteKicks) != 1 || p2Proj.Public.VoteKicks[0].Votes != 1 || p2Proj.Public.VoteKicks[0].HasVoted {
		t.Fatalf("expected p2 has not voted with 1 vote: %+v", p2Proj.Public.VoteKicks)
	}

	// 2. p2 votes to kick p4 (2/3)
	state, err = Apply(state, "p2", Command{Kind: CommandVoteKick, TargetID: "p4"}, time.Now())
	if err != nil {
		t.Fatalf("p2 vote kick failed: %v", err)
	}
	if !HasPlayer(state, "p4") {
		t.Fatal("p4 was kicked prematurely with 2/3 votes")
	}

	// 3. p3 votes to kick p4 (3/3 -> unanimous!)
	state, err = Apply(state, "p3", Command{Kind: CommandVoteKick, TargetID: "p4"}, time.Now())
	if err != nil {
		t.Fatalf("p3 vote kick failed: %v", err)
	}
	if HasPlayer(state, "p4") {
		t.Fatal("p4 should be removed after unanimous vote kick")
	}
	if len(state.VoteKicks["p4"]) != 0 {
		t.Fatalf("p4 vote kicks should be cleaned up: %+v", state.VoteKicks)
	}
}

func TestVoteKickRemovesTargetAndAdvancesPhaseWhenActivePlayer(t *testing.T) {
	fixture := newGameFixture(t, fixtureOptions{PlayerCount: 5})
	fixture.StartMatch("Detector")
	for _, id := range fixture.state.PlayerOrder {
		fixture.SetFaction(id, FactionService)
	}
	fixture.AcknowledgeRoles()

	state := fixture.State()
	if state.Phase != PhaseOperationInput {
		t.Fatalf("expected PhaseOperationInput, got %s", state.Phase)
	}
	activeID := state.ActivePlayerID
	if activeID == "" {
		t.Fatal("expected active player ID")
	}

	// Disconnect the active player
	var err error
	state, err = Disconnect(state, activeID, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	// All remaining connected players vote to kick active player
	for _, id := range state.PlayerOrder {
		if id == activeID {
			continue
		}
		state, err = Apply(state, id, Command{Kind: CommandVoteKick, TargetID: activeID}, time.Now())
		if err != nil {
			t.Fatalf("player %s vote kick failed: %v", id, err)
		}
	}

	// Target player should be removed
	if HasPlayer(state, activeID) {
		t.Fatalf("target %s still in room after vote kick", activeID)
	}

	// Game should have advanced to either next operation or PhaseDiscussion
	if state.Phase != PhaseOperationInput && state.Phase != PhaseDiscussion && state.Phase != PhaseOperationResult {
		t.Fatalf("expected active turn kick to advance phase, got %s", state.Phase)
	}
	if state.ActivePlayerID == activeID {
		t.Fatalf("active player ID should not be the kicked player: %s", state.ActivePlayerID)
	}
}

func TestVoteKickReconnectingCancelsVoteKick(t *testing.T) {
	state := testLobby(t, 4)
	var err error

	// Disconnect p4
	state, err = Disconnect(state, "p4", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	// p1 and p2 vote to kick p4
	state, err = Apply(state, "p1", Command{Kind: CommandVoteKick, TargetID: "p4"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	state, err = Apply(state, "p2", Command{Kind: CommandVoteKick, TargetID: "p4"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if state.VoteKicks["p4"]["p1"] != true || state.VoteKicks["p4"]["p2"] != true {
		t.Fatalf("votes not registered: %+v", state.VoteKicks)
	}

	// p4 reconnects
	state, err = Connect(state, "p4")
	if err != nil {
		t.Fatal(err)
	}

	// Vote kicks against p4 should be deleted
	if len(state.VoteKicks["p4"]) != 0 {
		t.Fatalf("vote kicks against p4 should be deleted on reconnect: %+v", state.VoteKicks)
	}

	// Public projection should have no active vote kicks for p4
	proj := PublicProjectionFor("r1", state)
	if len(proj.VoteKicks) != 0 {
		t.Fatalf("expected 0 vote kicks in projection, got: %+v", proj.VoteKicks)
	}
}

func TestVoteKickInRoleRevealDiscussionAndVotePhases(t *testing.T) {
	// 1. Role reveal: p5 disconnects without acknowledging role
	fixture := newGameFixture(t, fixtureOptions{PlayerCount: 5})
	fixture.StartMatch("Detector")
	state := fixture.State()

	var err error
	state, err = Disconnect(state, "p5", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	// p1..p4 ack role
	for _, id := range []string{"p1", "p2", "p3", "p4"} {
		state, err = Apply(state, id, Command{Kind: CommandAcknowledgeRole}, time.Now())
		if err != nil {
			t.Fatal(err)
		}
	}
	if state.Phase != PhaseRoleReveal {
		t.Fatalf("phase should be PhaseRoleReveal, got %s", state.Phase)
	}

	// Vote kick p5
	for _, id := range []string{"p1", "p2", "p3", "p4"} {
		state, err = Apply(state, id, Command{Kind: CommandVoteKick, TargetID: "p5"}, time.Now())
		if err != nil {
			t.Fatal(err)
		}
	}

	// After kicking p5, all remaining players have acked -> should advance to operation
	if state.Phase != PhaseOperationInput && state.Phase != PhaseOperationResult {
		t.Fatalf("expected advance to operation phase after kicking unacked p5, got %s", state.Phase)
	}

	// 2. Vote input phase: p4 disconnects without voting
	state = testLobby(t, 4)
	state.Phase = PhaseVoteInput
	state.Vote = VoteState{Submitted: map[string]string{}, Totals: map[string]int{}}
	state, err = Disconnect(state, "p4", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	// p1, p2, p3 vote
	state, err = Apply(state, "p1", Command{Kind: CommandSubmitVote, TargetID: "p2"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	state, err = Apply(state, "p2", Command{Kind: CommandSubmitVote, TargetID: "p1"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	state, err = Apply(state, "p3", Command{Kind: CommandSubmitVote, TargetID: "p1"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if state.Phase != PhaseVoteInput {
		t.Fatalf("phase should be PhaseVoteInput, got %s", state.Phase)
	}

	// Vote kick p4
	for _, id := range []string{"p1", "p2", "p3"} {
		state, err = Apply(state, id, Command{Kind: CommandVoteKick, TargetID: "p4"}, time.Now())
		if err != nil {
			t.Fatal(err)
		}
	}

	// After kicking p4, all remaining players have voted -> should advance to results
	if state.Phase != PhaseEnd {
		t.Fatalf("expected advance to PhaseEnd after kicking non-voter p4, got %s", state.Phase)
	}
}

func TestVoteKickValidationErrors(t *testing.T) {
	state := testLobby(t, 3)
	var err error

	// Cannot vote kick a connected player
	_, err = Apply(state, "p1", Command{Kind: CommandVoteKick, TargetID: "p2"}, time.Now())
	if !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("got %v, want ErrNotAllowed", err)
	}

	// Cannot vote kick non-existent player
	_, err = Apply(state, "p1", Command{Kind: CommandVoteKick, TargetID: "nonexistent"}, time.Now())
	if !errors.Is(err, ErrInvalidTarget) {
		t.Fatalf("got %v, want ErrInvalidTarget", err)
	}

	// Disconnected actor cannot vote kick
	state, err = Disconnect(state, "p1", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	state, err = Disconnect(state, "p3", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	_, err = Apply(state, "p1", Command{Kind: CommandVoteKick, TargetID: "p3"}, time.Now())
	if !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("got %v, want ErrNotAllowed", err)
	}
}

func TestShareDualAcknowledgement(t *testing.T) {
	fixture := newGameFixture(t, fixtureOptions{PlayerCount: 5})
	fixture.StartMatch("Share")
	state := fixture.State()
	for _, id := range state.PlayerOrder {
		var err error
		state, err = Apply(state, id, Command{Kind: CommandAcknowledgeRole}, time.Now())
		if err != nil {
			t.Fatal(err)
		}
	}
	if state.Phase != PhaseOperationInput {
		t.Fatalf("expected PhaseOperationInput, got %s", state.Phase)
	}
	activeID := state.ActivePlayerID
	targetID := "p2"
	if activeID == targetID {
		targetID = "p3"
	}
	bystanderID := "p4"
	if activeID == bystanderID || targetID == bystanderID {
		bystanderID = "p5"
	}

	var err error
	state, err = Apply(state, activeID, Command{Kind: CommandResolveOperation, TargetID: targetID}, time.Now())
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if state.Phase != PhaseOperationResult {
		t.Fatalf("expected PhaseOperationResult, got %s", state.Phase)
	}

	// Verify projections before any ack
	projActive := Project("r1", state, activeID)
	projTarget := Project("r1", state, targetID)
	projBystander := Project("r1", state, bystanderID)

	if !projActive.Private.CanSubmit || projActive.Private.OperationAcknowledged {
		t.Fatalf("active player unexpected projection: can_submit=%v acked=%v", projActive.Private.CanSubmit, projActive.Private.OperationAcknowledged)
	}
	if !projTarget.Private.CanSubmit || projTarget.Private.OperationAcknowledged || projTarget.Private.OperationResult == nil {
		t.Fatalf("target player unexpected projection: can_submit=%v acked=%v result=%+v", projTarget.Private.CanSubmit, projTarget.Private.OperationAcknowledged, projTarget.Private.OperationResult)
	}
	if projBystander.Private.CanSubmit || projBystander.Private.OperationAcknowledged {
		t.Fatalf("bystander player unexpected projection: can_submit=%v acked=%v", projBystander.Private.CanSubmit, projBystander.Private.OperationAcknowledged)
	}

	// Bystander cannot ack
	_, err = Apply(state, bystanderID, Command{Kind: CommandOperationExplainDone}, time.Now())
	if !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("bystander explain done expected ErrNotAllowed, got %v", err)
	}

	// Target acks first
	state, err = Apply(state, targetID, Command{Kind: CommandOperationExplainDone}, time.Now())
	if err != nil {
		t.Fatalf("target explain done failed: %v", err)
	}
	if state.Phase != PhaseOperationResult {
		t.Fatalf("expected phase to remain PhaseOperationResult after target ack, got %s", state.Phase)
	}

	projActive = Project("r1", state, activeID)
	projTarget = Project("r1", state, targetID)
	if !projActive.Private.CanSubmit || projActive.Private.OperationAcknowledged {
		t.Fatalf("active player should still be able to submit after target ack")
	}
	if projTarget.Private.CanSubmit || !projTarget.Private.OperationAcknowledged {
		t.Fatalf("target player should have acknowledged: can_submit=%v acked=%v", projTarget.Private.CanSubmit, projTarget.Private.OperationAcknowledged)
	}
	if len(projActive.Public.Operation.AcknowledgedPlayerIDs) != 1 || projActive.Public.Operation.AcknowledgedPlayerIDs[0] != targetID {
		t.Fatalf("public operation acknowledged_player_ids mismatch: %+v", projActive.Public.Operation.AcknowledgedPlayerIDs)
	}

	// Active player acks second -> advances to PhaseOperationInterlude
	state, err = Apply(state, activeID, Command{Kind: CommandOperationExplainDone}, time.Now())
	if err != nil {
		t.Fatalf("active explain done failed: %v", err)
	}
	if state.Phase != PhaseOperationInterlude {
		t.Fatalf("expected PhaseOperationInterlude after both ack, got %s", state.Phase)
	}
	if state.DiscussionDeadline == nil {
		t.Fatal("expected discussion deadline to be set in interlude")
	}
}

func TestInfoForTwoDualAcknowledgement(t *testing.T) {
	fixture := newGameFixture(t, fixtureOptions{PlayerCount: 5, EnabledOperations: []string{"InfoForTwo"}})
	fixture.StartMatch("InfoForTwo")
	state := fixture.State()
	for _, id := range state.PlayerOrder {
		var err error
		state, err = Apply(state, id, Command{Kind: CommandAcknowledgeRole}, time.Now())
		if err != nil {
			t.Fatal(err)
		}
	}
	if state.Phase != PhaseOperationInput {
		t.Fatalf("expected PhaseOperationInput, got %s", state.Phase)
	}
	activeID := state.ActivePlayerID
	targetID := "p2"
	if activeID == targetID {
		targetID = "p3"
	}

	var err error
	state, err = Apply(state, activeID, Command{Kind: CommandResolveOperation, TargetID: targetID}, time.Now())
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if state.Phase != PhaseOperationResult {
		t.Fatalf("expected PhaseOperationResult, got %s", state.Phase)
	}

	// Active acks first
	state, err = Apply(state, activeID, Command{Kind: CommandOperationExplainDone}, time.Now())
	if err != nil {
		t.Fatalf("active explain done failed: %v", err)
	}
	if state.Phase != PhaseOperationResult {
		t.Fatalf("expected phase to remain PhaseOperationResult after 1st ack, got %s", state.Phase)
	}

	// Target acks second
	state, err = Apply(state, targetID, Command{Kind: CommandOperationExplainDone}, time.Now())
	if err != nil {
		t.Fatalf("target explain done failed: %v", err)
	}
	if state.Phase != PhaseOperationInterlude {
		t.Fatalf("expected PhaseOperationInterlude after 2nd ack, got %s", state.Phase)
	}
}

func TestSinglePlayerOperationSingleAcknowledgement(t *testing.T) {
	fixture := newGameFixture(t, fixtureOptions{PlayerCount: 5})
	fixture.StartMatch("Detector")
	state := fixture.State()
	for _, id := range state.PlayerOrder {
		var err error
		state, err = Apply(state, id, Command{Kind: CommandAcknowledgeRole}, time.Now())
		if err != nil {
			t.Fatal(err)
		}
	}
	if state.Phase != PhaseOperationInput {
		t.Fatalf("expected PhaseOperationInput, got %s", state.Phase)
	}
	activeID := state.ActivePlayerID
	var targets []string
	for _, id := range state.PlayerOrder {
		if id != activeID {
			targets = append(targets, id)
			if len(targets) == 2 {
				break
			}
		}
	}

	var err error
	state, err = Apply(state, activeID, Command{Kind: CommandResolveOperation, TargetIDs: targets}, time.Now())
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if state.Phase != PhaseOperationResult {
		t.Fatalf("expected PhaseOperationResult, got %s", state.Phase)
	}

	// Single ack advances immediately
	state, err = Apply(state, activeID, Command{Kind: CommandOperationExplainDone}, time.Now())
	if err != nil {
		t.Fatalf("active explain done failed: %v", err)
	}
	if state.Phase != PhaseOperationInterlude {
		t.Fatalf("expected PhaseOperationInterlude after single ack, got %s", state.Phase)
	}
}

func TestVotingResolvesDirectlyToPhaseEnd(t *testing.T) {
	state := testLobby(t, 3)
	state.Phase = PhaseVoteInput
	now := time.Now()

	p1 := state.Players["p1"]
	p1.Faction = FactionService
	state.Players["p1"] = p1

	p2 := state.Players["p2"]
	p2.Faction = FactionVirus
	state.Players["p2"] = p2

	p3 := state.Players["p3"]
	p3.Faction = FactionService
	state.Players["p3"] = p3

	var err error
	state, err = Apply(state, "p1", Command{Kind: CommandSubmitVote, TargetID: "p2"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if state.Phase != PhaseVoteInput {
		t.Fatalf("expected PhaseVoteInput while waiting for other votes, got %s", state.Phase)
	}

	state, err = Apply(state, "p2", Command{Kind: CommandSubmitVote, TargetID: "p1"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if state.Phase != PhaseVoteInput {
		t.Fatalf("expected PhaseVoteInput while waiting for other votes, got %s", state.Phase)
	}

	state, err = Apply(state, "p3", Command{Kind: CommandSubmitVote, TargetID: "p2"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if state.Phase != PhaseEnd {
		t.Fatalf("expected PhaseEnd immediately after all votes submitted, got %s", state.Phase)
	}

	if state.Vote.ImprisonedPlayerID != "p2" {
		t.Fatalf("expected p2 imprisoned, got %s", state.Vote.ImprisonedPlayerID)
	}
	if state.Winner != FactionService {
		t.Fatalf("expected Service winner, got %s", state.Winner)
	}

	proj := Project("room1", state, "p1")
	if len(proj.Public.Leaderboard) != 3 {
		t.Fatalf("expected 3 leaderboard entries, got %d", len(proj.Public.Leaderboard))
	}
	for _, entry := range proj.Public.Leaderboard {
		if entry.PlayerID == "p1" || entry.PlayerID == "p3" {
			if entry.Result != "WINNER" {
				t.Fatalf("expected %s to be WINNER, got %s", entry.PlayerID, entry.Result)
			}
		} else if entry.PlayerID == "p2" {
			if entry.Result != "LOSER" {
				t.Fatalf("expected p2 to be LOSER, got %s", entry.Result)
			}
		}
	}
}

func TestLeaderboardHiddenAgendas(t *testing.T) {
	// 1. Scapegoat Win: Scapegoat is imprisoned -> only Scapegoat wins, Winner is FactionNone
	t.Run("ScapegoatWin", func(t *testing.T) {
		state := testLobby(t, 3)
		p1 := state.Players["p1"]
		p1.Faction = FactionService
		p1.ObjectiveKind = "IMPRISON_SELF"
		p1.Statuses = []string{"STRAIN"}
		state.Players["p1"] = p1

		p2 := state.Players["p2"]
		p2.Faction = FactionVirus
		state.Players["p2"] = p2

		p3 := state.Players["p3"]
		p3.Faction = FactionService
		state.Players["p3"] = p3

		state.Vote.Submitted = map[string]string{
			"p1": "p2",
			"p2": "p1",
			"p3": "p1",
		}
		state.resolveVote()

		if state.Vote.ImprisonedPlayerID != "p1" {
			t.Fatalf("expected p1 imprisoned, got %s", state.Vote.ImprisonedPlayerID)
		}
		if state.Winner != FactionNone {
			t.Fatalf("expected FactionNone when scapegoat imprisoned, got %s", state.Winner)
		}

		lb := buildLeaderboard(state)
		for _, entry := range lb {
			if entry.PlayerID == "p1" {
				if entry.Result != "WINNER" {
					t.Fatalf("expected Scapegoat p1 to WIN, got %s", entry.Result)
				}
				if entry.ObjectiveKind != "IMPRISON_SELF" {
					t.Fatalf("expected ObjectiveKind IMPRISON_SELF, got %s", entry.ObjectiveKind)
				}
			} else {
				if entry.Result != "LOSER" {
					t.Fatalf("expected other player %s to LOSE, got %s", entry.PlayerID, entry.Result)
				}
			}
		}
	})

	// 2. Scapegoat Loss: Scapegoat is NOT imprisoned -> Scapegoat loses
	t.Run("ScapegoatLoss", func(t *testing.T) {
		state := testLobby(t, 3)
		p1 := state.Players["p1"]
		p1.Faction = FactionService
		p1.ObjectiveKind = "IMPRISON_SELF"
		state.Players["p1"] = p1

		p2 := state.Players["p2"]
		p2.Faction = FactionVirus
		state.Players["p2"] = p2

		p3 := state.Players["p3"]
		p3.Faction = FactionService
		state.Players["p3"] = p3

		state.Vote.Submitted = map[string]string{
			"p1": "p2",
			"p2": "p3",
			"p3": "p2",
		}
		state.resolveVote()

		if state.Vote.ImprisonedPlayerID != "p2" {
			t.Fatalf("expected p2 imprisoned, got %s", state.Vote.ImprisonedPlayerID)
		}
		if state.Winner != FactionService {
			t.Fatalf("expected FactionService, got %s", state.Winner)
		}

		lb := buildLeaderboard(state)
		for _, entry := range lb {
			if entry.PlayerID == "p1" {
				if entry.Result != "LOSER" {
					t.Fatalf("expected Scapegoat p1 to LOSE when not imprisoned, got %s", entry.Result)
				}
			} else if entry.PlayerID == "p3" {
				if entry.Result != "WINNER" {
					t.Fatalf("expected Service p3 to WIN, got %s", entry.Result)
				}
			}
		}
	})

	// 3. Grudge Win vs Loss
	t.Run("GrudgeWinAndLoss", func(t *testing.T) {
		state := testLobby(t, 3)
		p1 := state.Players["p1"]
		p1.Faction = FactionService
		p1.ObjectiveKind = "IMPRISON_TARGET"
		p1.ObjectiveTarget = "p2"
		state.Players["p1"] = p1

		p2 := state.Players["p2"]
		p2.Faction = FactionVirus
		state.Players["p2"] = p2

		p3 := state.Players["p3"]
		p3.Faction = FactionService
		p3.ObjectiveKind = "IMPRISON_TARGET"
		p3.ObjectiveTarget = "p1"
		state.Players["p3"] = p3

		state.Vote.Submitted = map[string]string{
			"p1": "p2",
			"p2": "p2",
			"p3": "p2",
		}
		state.resolveVote()

		lb := buildLeaderboard(state)
		for _, entry := range lb {
			if entry.PlayerID == "p1" {
				if entry.Result != "WINNER" {
					t.Fatalf("expected Grudge holder p1 targeting imprisoned p2 to WIN, got %s", entry.Result)
				}
				if entry.ObjectiveName != p2.Name {
					t.Fatalf("expected ObjectiveName %s, got %s", p2.Name, entry.ObjectiveName)
				}
			}
			if entry.PlayerID == "p3" {
				if entry.Result != "LOSER" {
					t.Fatalf("expected Grudge holder p3 targeting p1 (not imprisoned) to LOSE, got %s", entry.Result)
				}
			}
		}
	})

	// 4. Infatuation Win vs Loss
	t.Run("InfatuationWinAndLoss", func(t *testing.T) {
		state := testLobby(t, 3)
		p1 := state.Players["p1"]
		p1.Faction = FactionVirus
		p1.ObjectiveKind = "TARGET_WINS"
		p1.ObjectiveTarget = "p3" // targets p3 (Service)
		state.Players["p1"] = p1

		p2 := state.Players["p2"]
		p2.Faction = FactionVirus
		p2.ObjectiveKind = "TARGET_WINS"
		p2.ObjectiveTarget = "p2" // targets self / virus
		state.Players["p2"] = p2

		p3 := state.Players["p3"]
		p3.Faction = FactionService
		state.Players["p3"] = p3

		// Imprison p2 (Virus) -> Service wins (p3 wins)
		state.Vote.Submitted = map[string]string{
			"p1": "p2",
			"p2": "p3",
			"p3": "p2",
		}
		state.resolveVote()

		lb := buildLeaderboard(state)
		for _, entry := range lb {
			if entry.PlayerID == "p1" {
				if entry.Result != "WINNER" {
					t.Fatalf("expected Infatuation p1 targeting winner p3 to WIN, got %s", entry.Result)
				}
				if entry.ObjectiveName != p3.Name {
					t.Fatalf("expected ObjectiveName %s, got %s", p3.Name, entry.ObjectiveName)
				}
			}
			if entry.PlayerID == "p2" {
				if entry.Result != "LOSER" {
					t.Fatalf("expected Infatuation p2 targeting loser p2 to LOSE, got %s", entry.Result)
				}
			}
		}
	})

	// 5. Red Defector Win vs Loss
	t.Run("RedDefectorWinAndLoss", func(t *testing.T) {
		state := testLobby(t, 3)
		// p1 is Red Defector (initial Virus, turned Service)
		p1 := state.Players["p1"]
		p1.InitialFaction = FactionVirus
		p1.Faction = FactionService
		p1.ObjectiveKind = "RED_DEFECTOR"
		p1.Statuses = []string{"RED_DEFECTOR"}
		state.Players["p1"] = p1

		p2 := state.Players["p2"]
		p2.InitialFaction = FactionVirus
		p2.Faction = FactionVirus
		state.Players["p2"] = p2

		p3 := state.Players["p3"]
		p3.InitialFaction = FactionService
		p3.Faction = FactionService
		state.Players["p3"] = p3

		// Case A: Service wins, Virus p2 votes for p3 (NOT for p1) -> Red Defector p1 WINS
		state.Vote.Submitted = map[string]string{
			"p1": "p2",
			"p2": "p3",
			"p3": "p2",
		}
		state.resolveVote()

		lb := buildLeaderboard(state)
		for _, entry := range lb {
			if entry.PlayerID == "p1" {
				if entry.Result != "WINNER" {
					t.Fatalf("expected Red Defector p1 to WIN when no virus voted for them, got %s", entry.Result)
				}
				if entry.InitialFaction != FactionVirus || entry.Faction != FactionService {
					t.Fatalf("expected InitialFaction VIRUS and Faction SERVICE, got %s / %s", entry.InitialFaction, entry.Faction)
				}
			}
		}

		// Case B: Virus p2 votes for Red Defector p1 -> Red Defector p1 LOSES
		state.Vote.Totals = nil
		state.Vote.Submitted = map[string]string{
			"p1": "p2",
			"p2": "p1", // Virus voted for p1
			"p3": "p2",
		}
		state.resolveVote()

		lb = buildLeaderboard(state)
		for _, entry := range lb {
			if entry.PlayerID == "p1" {
				if entry.Result != "LOSER" {
					t.Fatalf("expected Red Defector p1 to LOSE when virus voted for them, got %s", entry.Result)
				}
			}
		}
	})
}

func TestRematchAllowedFromPhaseEnd(t *testing.T) {
	state := testLobby(t, 3)
	state.Phase = PhaseEnd

	next, err := Apply(state, state.HostID, Command{Kind: CommandRematch}, time.Now())
	if err != nil {
		t.Fatalf("host rematch from PhaseEnd failed: %v", err)
	}
	if next.Phase != PhaseLobby {
		t.Fatalf("expected PhaseLobby after rematch, got %s", next.Phase)
	}

	// Non-host cannot rematch
	_, err = Apply(state, "p2", Command{Kind: CommandRematch}, time.Now())
	if err == nil {
		t.Fatal("expected error when non-host requests rematch")
	}
}
