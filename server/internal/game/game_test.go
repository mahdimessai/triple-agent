package game

import (
	"errors"
	"testing"
	"time"
)

func testLobby(t *testing.T, count int) State {
	t.Helper()
	settings := DefaultSettings()
	settings.MinPlayers = count
	state := NewLobby("p1", "P1", settings)
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
	state := NewLobby("p1", "Host", DefaultSettings())
	_, err := AddPlayer(state, "p1", "Again")
	if !errors.Is(err, ErrPlayerExists) {
		t.Fatalf("got %v, want ErrPlayerExists", err)
	}
}

func TestReadyDoesNotCreatePresence(t *testing.T) {
	state := NewLobby("p1", "Host", DefaultSettings())
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
	state := NewLobby("p1", "Host", DefaultSettings())
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
	resolveVote(&state)
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
