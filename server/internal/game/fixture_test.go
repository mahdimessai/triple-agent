package game

import (
	"fmt"
	"testing"
	"time"
)

type fixtureOptions struct {
	PlayerCount int
	VirusCount  int
	Seed        uint64

	EnabledOperations []string
	EnabledRoles      []RoleKind
}

type gameFixture struct {
	t     *testing.T
	state State
	now   time.Time
}

func newGameFixture(t *testing.T, opts fixtureOptions) *gameFixture {
	t.Helper()
	if opts.PlayerCount < 1 {
		t.Fatal("fixture requires at least one player")
	}

	settings := DefaultSettings()
	settings.MinPlayers = opts.PlayerCount
	settings.MaxPlayers = opts.PlayerCount
	settings.VirusCount = opts.VirusCount
	if opts.EnabledOperations != nil {
		settings.EnabledOperations = make(map[string]bool, len(opts.EnabledOperations))
		for _, id := range opts.EnabledOperations {
			settings.EnabledOperations[id] = true
		}
	}
	if opts.EnabledRoles != nil {
		settings.EnabledRoles = make(map[string]bool, len(opts.EnabledRoles))
		for _, role := range opts.EnabledRoles {
			settings.EnabledRoles[string(role)] = true
		}
	}

	state := NewLobby("p1", "P1", settings)
	if opts.Seed != 0 {
		state.RandomState = opts.Seed
	} else {
		state.RandomState = 1
	}
	for index := 2; index <= opts.PlayerCount; index++ {
		id := fmt.Sprintf("p%d", index)
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

	return &gameFixture{t: t, state: state, now: time.Unix(100, 0).UTC()}
}

func (f *gameFixture) State() State {
	return f.state
}

func (f *gameFixture) Player(id string) Player {
	player, ok := f.state.Players[id]
	if !ok {
		f.t.Fatalf("fixture player %q does not exist", id)
	}
	return player
}

func (f *gameFixture) StartMatch(operationKind string) {
	f.t.Helper()
	f.state = readyAll(f.t, f.state)
	state, err := Apply(f.state, f.state.HostID, Command{
		Kind:          CommandStartMatch,
		OperationKind: operationKind,
	}, f.now)
	if err != nil {
		f.t.Fatalf("start match: %v", err)
	}
	f.state = state
}

func (f *gameFixture) AcknowledgeRoles() {
	f.t.Helper()
	for _, id := range f.state.PlayerOrder {
		state, err := Apply(f.state, id, Command{Kind: CommandAcknowledgeRole}, f.now)
		if err != nil {
			f.t.Fatalf("acknowledge role for %s: %v", id, err)
		}
		f.state = state
	}
}

func (f *gameFixture) Start(operationKind string) {
	f.t.Helper()
	f.StartMatch(operationKind)
	f.AcknowledgeRoles()
}

func (f *gameFixture) Apply(actorID string, command Command) {
	f.t.Helper()
	state, err := Apply(f.state, actorID, command, f.now)
	if err != nil {
		f.t.Fatalf("apply %s as %s: %v", command.Kind, actorID, err)
	}
	f.state = state
}

func (f *gameFixture) Resolve(command Command) {
	f.t.Helper()
	actorID := f.state.ActivePlayerID
	if f.state.Operation != nil && f.state.Operation.InputOwnerID != "" {
		actorID = f.state.Operation.InputOwnerID
	}
	f.Apply(actorID, command)
}

func (f *gameFixture) FinishOperation() {
	f.t.Helper()
	if f.state.Phase == PhaseOperationResult {
		f.Apply(f.state.ActivePlayerID, Command{Kind: CommandOperationExplainDone})
	}
	if f.state.Phase == PhaseOperationInterlude {
		f.Apply(f.state.HostID, Command{Kind: CommandAdvanceInterlude})
	}
}

func (f *gameFixture) SetFaction(id string, faction Faction) {
	f.t.Helper()
	player := f.Player(id)
	player.InitialFaction = faction
	player.Faction = faction
	player.ApparentFaction = nil
	f.state.Players[id] = player
}

func (f *gameFixture) SetRole(id string, role RoleKind, faction Faction) {
	f.t.Helper()
	player := f.Player(id)
	player.Role = role
	player.InitialFaction = faction
	player.Faction = faction
	player.ApparentFaction = nil
	switch role {
	case RoleLyingRed:
		apparent := FactionService
		player.ApparentFaction = &apparent
	case RoleLyingBlue:
		apparent := FactionVirus
		player.ApparentFaction = &apparent
	}
	f.state.Players[id] = player
}

func (f *gameFixture) DealAllOperations() {
	f.t.Helper()
	op, err := operationForStart(&f.state, "")
	if err != nil {
		f.t.Fatalf("start operation dealing: %v", err)
	}
	f.state.PlannedOperation = op.definition.ID
	if err := beginPlannedOperation(&f.state); err != nil {
		f.t.Fatalf("begin first operation: %v", err)
	}
	for f.state.OperationDeals < f.state.OperationDealTarget {
		f.state.Phase = PhaseOperationInterlude
		if err := advanceInterlude(&f.state, f.now); err != nil {
			f.t.Fatalf("advance operation dealing: %v", err)
		}
	}
}
