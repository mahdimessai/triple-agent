package domain

import "strings"

// beginOperation initializes the common operation state without resolving it.
// Input-bearing operations call this from Begin and wait for the corresponding
// operation command before producing their private result.
func beginOperation(state *GameState, definition OperationDefinition) error {
	if len(state.PlayerOrder) == 0 {
		return ErrNotEnoughPlayers
	}
	state.ActivePlayerID = activePlayerID(*state)
	state.Operation = newOperationState(state, definition)
	return nil
}

func beginPrivateOperation(state *GameState, definition OperationDefinition, resolve func(*GameState, Command) error) error {
	if err := beginOperation(state, definition); err != nil {
		return err
	}
	return resolve(state, Command{})
}

// randomOtherPlayer deals a target for operations whose target must not be
// visible as a choice.
func randomOtherPlayer(state *GameState, playerID string) (string, error) {
	candidates := otherPlayerIDs(*state, playerID)
	if len(candidates) == 0 {
		return "", ErrInvalidTarget
	}
	return candidates[nextRandom(state, len(candidates))], nil
}

func requiredTargets(state *GameState, command Command, count int) ([]string, error) {
	if _, ok := state.Players[state.ActivePlayerID]; !ok {
		return nil, ErrInvalidTarget
	}
	targets := operationTargetIDs(command)
	if len(targets) != count {
		return nil, ErrInvalidTarget
	}
	seen := make(map[string]bool, len(targets))
	for _, targetID := range targets {
		if targetID == state.ActivePlayerID || seen[targetID] {
			return nil, ErrInvalidTarget
		}
		if _, exists := state.Players[targetID]; !exists {
			return nil, ErrInvalidTarget
		}
		seen[targetID] = true
	}
	return targets, nil
}

func requiredTarget(state *GameState, command Command) (string, error) {
	targets, err := requiredTargets(state, command, 1)
	if err != nil {
		return "", err
	}
	return targets[0], nil
}

type recruitmentResolver struct{}

func (recruitmentResolver) Definition() OperationDefinition {
	return OperationDefinition{ID: "Injection", Name: "Recruitment", InputKind: OperationInputOneTarget, TargetCount: 1, MinPlayers: 5, PublicInstruction: "A private recruitment operation is in progress.", PrivateInstruction: "Choose one target for recruitment."}
}

func (r recruitmentResolver) Begin(state *GameState) error {
	return beginOperation(state, r.Definition())
}

func (recruitmentResolver) Resolve(state *GameState, command Command) error {
	active, ok := state.Players[state.ActivePlayerID]
	if !ok {
		return ErrInvalidTarget
	}
	targetID, err := requiredTarget(state, command)
	if err != nil {
		return err
	}
	setFaction(state, targetID, active.Faction)
	target := state.Players[targetID]
	result := OperationResult{Code: "AGENCY_ASSIGNED", TargetPlayerID: targetID, TargetFaction: target.Faction, Message: "The target now serves the active player's agency."}
	state.Operation.PrivateResults = map[string]OperationResult{targetID: result}
	return nil
}

type strainResolver struct{}

func (strainResolver) Definition() OperationDefinition {
	return OperationDefinition{
		ID:                 "Strain",
		Name:               "Operation: Scapegoat",
		InputKind:          OperationInputPrivateInfo,
		MinPlayers:         5,
		MinEventOrder:      1,
		Category:           1,
		Hidden:             true,
		Enabled:            true,
		PublicInstruction:  "The active player received a private objective.",
		PrivateInstruction: "You win only if you are imprisoned.",
	}
}

func (r strainResolver) Begin(state *GameState) error {
	return beginPrivateOperation(state, r.Definition(), r.Resolve)
}

func (strainResolver) Resolve(state *GameState, _ Command) error {
	active, ok := state.Players[state.ActivePlayerID]
	if !ok {
		return ErrInvalidTarget
	}
	active.ObjectiveKind = "IMPRISON_SELF"
	active.Statuses = append(active.Statuses, "STRAIN")
	state.Players[active.ID] = active
	state.Operation.PrivateResults = map[string]OperationResult{active.ID: {Code: "OBJECTIVE_ASSIGNED", Message: "You win only if you are imprisoned."}}
	return nil
}

type grudgeResolver struct{}

func (grudgeResolver) Definition() OperationDefinition {
	return OperationDefinition{
		ID:                 "Grudge",
		Name:               "Grudge",
		InputKind:          OperationInputPrivateInfo,
		MinPlayers:         5,
		MinEventOrder:      1,
		Category:           1,
		Hidden:             true,
		Enabled:            true,
		PublicInstruction:  "The active player received a private objective.",
		PrivateInstruction: "You now win if and only if your assigned target is imprisoned.",
	}
}

// The target is dealt by the server, not chosen. A visible choice would tell
// the room which hidden agenda was drawn, which is exactly what the mask exists
// to prevent.
func (r grudgeResolver) Begin(state *GameState) error {
	return beginPrivateOperation(state, r.Definition(), grudgeResolver{}.Resolve)
}

func (grudgeResolver) Resolve(state *GameState, _ Command) error {
	active, ok := state.Players[state.ActivePlayerID]
	if !ok {
		return ErrInvalidTarget
	}
	targetID, err := randomOtherPlayer(state, active.ID)
	if err != nil {
		return err
	}
	active.ObjectiveKind = "IMPRISON_TARGET"
	active.ObjectiveTarget = targetID
	active.Statuses = append(active.Statuses, "GRUDGE")
	state.Players[active.ID] = active
	state.Operation.TargetPlayerID = targetID
	state.Operation.TargetPlayerIDs = []string{targetID}
	state.Operation.PrivateResults = map[string]OperationResult{active.ID: {Code: "GRUDGE_TARGET_ASSIGNED", TargetPlayerID: targetID, Message: "You win only if your assigned target is imprisoned."}}
	return nil
}

type infatuationResolver struct{}

func (infatuationResolver) Definition() OperationDefinition {
	return OperationDefinition{
		ID:                 "Infatuation",
		Name:               "Infatuation",
		InputKind:          OperationInputPrivateInfo,
		MinPlayers:         5,
		MinEventOrder:      1,
		Category:           1,
		Hidden:             true,
		Enabled:            true,
		PublicInstruction:  "The active player received a private objective.",
		PrivateInstruction: "You now win if and only if your assigned target wins at the end of the round.",
	}
}

func (r infatuationResolver) Begin(state *GameState) error {
	return beginPrivateOperation(state, r.Definition(), infatuationResolver{}.Resolve)
}

func (infatuationResolver) Resolve(state *GameState, _ Command) error {
	active, ok := state.Players[state.ActivePlayerID]
	if !ok {
		return ErrInvalidTarget
	}
	targetID, err := randomOtherPlayer(state, active.ID)
	if err != nil {
		return err
	}
	active.ObjectiveKind = "TARGET_WINS"
	active.ObjectiveTarget = targetID
	active.Statuses = append(active.Statuses, "INFATUATION")
	state.Players[active.ID] = active
	state.Operation.TargetPlayerID = targetID
	state.Operation.TargetPlayerIDs = []string{targetID}
	state.Operation.PrivateResults = map[string]OperationResult{active.ID: {Code: "INFATUATION_TARGET_ASSIGNED", TargetPlayerID: targetID, Message: "You win only if your assigned target wins the round."}}
	return nil
}

type flipResolver struct{}

func (flipResolver) Definition() OperationDefinition {
	return OperationDefinition{
		ID:                 "Flip",
		Name:               "Sleeper Agent",
		InputKind:          OperationInputPrivateInfo,
		MinPlayers:         5,
		MinEventOrder:      1,
		Category:           1,
		Hidden:             true,
		Enabled:            true,
		PublicInstruction:  "The active player's current agency is being re-evaluated.",
		PrivateInstruction: "Your current agency may change; your initial agency stays fixed.",
	}
}

func (r flipResolver) Begin(state *GameState) error {
	return beginPrivateOperation(state, r.Definition(), r.Resolve)
}

func (flipResolver) Resolve(state *GameState, _ Command) error {
	active, ok := state.Players[state.ActivePlayerID]
	if !ok {
		return ErrInvalidTarget
	}
	// A Loyalist reads the same activation message as everyone else and simply
	// finds their agency unchanged.
	if isLoyalist(active) {
		state.Operation.PrivateResults = map[string]OperationResult{active.ID: {
			Code:        "AGENCY_HELD",
			YourFaction: active.Faction,
			Message:     "Your loyalty held. Your agency did not change.",
		}}
		return nil
	}
	result := OperationResult{Message: "Your current agency changed; your initial agency remains unchanged."}
	if active.Faction == FactionVirus {
		setFaction(state, active.ID, FactionService)
		active.Statuses = append(active.Statuses, "CURED")
		result.Code = "CURED"
	} else {
		setFaction(state, active.ID, FactionVirus)
		active.Statuses = append(active.Statuses, "INFECTED")
		result.Code = "INFECTED"
	}
	active.Faction = state.Players[active.ID].Faction
	result.YourFaction = active.Faction
	state.Players[active.ID] = active
	state.Operation.PrivateResults = map[string]OperationResult{active.ID: result}
	return nil
}

type hiddenTipResolver struct{}

func (hiddenTipResolver) Definition() OperationDefinition {
	return OperationDefinition{
		ID:                 "HiddenOneRandom",
		Name:               "Secret Tip",
		InputKind:          OperationInputPrivateInfo,
		MinPlayers:         5,
		MinEventOrder:      1,
		Category:           1,
		Hidden:             true,
		Enabled:            true,
		PublicInstruction:  "The active player received a private tip.",
		PrivateInstruction: "Your source reveals one other agent's agency.",
	}
}

func (r hiddenTipResolver) Begin(state *GameState) error {
	return beginPrivateOperation(state, r.Definition(), r.Resolve)
}

func (hiddenTipResolver) Resolve(state *GameState, _ Command) error {
	activeID := state.ActivePlayerID
	if _, ok := state.Players[activeID]; !ok {
		return ErrInvalidTarget
	}
	targetID := chooseRandomOther(state, activeID)
	if targetID == "" {
		return ErrInvalidTarget
	}
	target := state.Players[targetID]
	state.Operation.PrivateResults = map[string]OperationResult{activeID: {Code: "FACTION_REVEALED", TargetPlayerID: targetID, TargetFaction: checkFaction(target), Message: "Your source reveals one other agent's agency."}}
	return nil
}

type oneOfTwoResolver struct{}

func (oneOfTwoResolver) Definition() OperationDefinition {
	return OperationDefinition{
		ID:                 "OneOfTwo",
		Name:               "Danish Intelligence",
		InputKind:          OperationInputPrivateInfo,
		MinPlayers:         4,
		MinEventOrder:      1,
		Enabled:            true,
		PublicInstruction:  "The active player received a two-name intelligence intercept.",
		PrivateInstruction: "The server selected two names and prepared a private result.",
	}
}

func (r oneOfTwoResolver) Begin(state *GameState) error {
	return beginPrivateOperation(state, r.Definition(), r.Resolve)
}

func (oneOfTwoResolver) Resolve(state *GameState, _ Command) error {
	activeID := state.ActivePlayerID
	if _, ok := state.Players[activeID]; !ok {
		return ErrInvalidTarget
	}
	pair := chooseRandomOthers(state, activeID, 2)
	if len(pair) != 2 {
		return ErrInvalidTarget
	}
	result := OperationResult{Code: "OPERATION_RESOLVED", TargetPlayerIDs: pair, Message: "The server prepared a two-name intelligence result."}
	if checkFaction(state.Players[pair[0]]) != checkFaction(state.Players[pair[1]]) {
		result.Code = "ONE_VIRUS_ONE_SERVICE"
	} else {
		result.Code = "SAME_FACTION"
	}
	state.Operation.PrivateResults = map[string]OperationResult{activeID: result}
	return nil
}

type twoFriendsResolver struct{}

func (twoFriendsResolver) Definition() OperationDefinition {
	return OperationDefinition{
		ID:                 "TwoFriends",
		Name:               "Old Photographs",
		InputKind:          OperationInputPrivateInfo,
		MinPlayers:         5,
		MinEventOrder:      1,
		Enabled:            true,
		PublicInstruction:  "The active player received evidence about two starting agencies.",
		PrivateInstruction: "The server selected two names and compared their starting agencies.",
	}
}

func (r twoFriendsResolver) Begin(state *GameState) error {
	return beginPrivateOperation(state, r.Definition(), r.Resolve)
}

func (twoFriendsResolver) Resolve(state *GameState, _ Command) error {
	activeID := state.ActivePlayerID
	if _, ok := state.Players[activeID]; !ok {
		return ErrInvalidTarget
	}
	pair := chooseRandomOthers(state, activeID, 2)
	if len(pair) != 2 {
		return ErrInvalidTarget
	}
	code := "DIFFERENT_INITIAL_AGENCY"
	if checkInitialFaction(state.Players[pair[0]]) == checkInitialFaction(state.Players[pair[1]]) {
		code = "SAME_INITIAL_AGENCY"
	}
	state.Operation.PrivateResults = map[string]OperationResult{activeID: {Code: code, TargetPlayerIDs: append([]string(nil), pair...), Message: "The photographs compare the two agents' starting agencies."}}
	return nil
}

type undercoverResolver struct{}

func (undercoverResolver) Definition() OperationDefinition {
	return OperationDefinition{
		ID:                 "Undercover",
		Name:               "Deep Undercover",
		InputKind:          OperationInputOneTarget,
		TargetCount:        1,
		MinPlayers:         5,
		MinEventOrder:      1,
		IsPack:             true,
		Enabled:            true,
		PublicInstruction:  "The active player is following one agent undercover.",
		PrivateInstruction: "Choose one agent to investigate.",
	}
}

func (r undercoverResolver) Begin(state *GameState) error {
	return beginOperation(state, r.Definition())
}

func (undercoverResolver) Resolve(state *GameState, command Command) error {
	active, ok := state.Players[state.ActivePlayerID]
	if !ok {
		return ErrInvalidTarget
	}
	targetID, err := requiredTarget(state, command)
	if err != nil {
		return err
	}
	target := state.Players[targetID]
	result := OperationResult{TargetPlayerID: targetID, TargetFaction: checkFaction(target), Message: "The undercover investigation is complete."}
	if checkFaction(target) == FactionVirus {
		if setFaction(state, active.ID, FactionVirus) {
			active.Faction = FactionVirus
			active.Statuses = append(active.Statuses, "UNDERCOVER_JOINED_VIRUS")
			result.Code = "JOINED_VIRUS"
		} else {
			result.Code = "JOINED_VIRUS_HELD"
			result.Message = "Your target works for VIRUS, but your loyalty held and your agency did not change."
		}
	} else {
		result.Code = "TARGET_SERVICE"
	}
	result.YourFaction = active.Faction
	activeResult := result
	state.Players[active.ID] = active
	state.Operation.PrivateResults = map[string]OperationResult{active.ID: activeResult}
	return nil
}

type sharedIntelResolver struct{}

func (sharedIntelResolver) Definition() OperationDefinition {
	return OperationDefinition{
		ID:                 "InfoForTwo",
		Name:               "Unfortunate Encounter",
		InputKind:          OperationInputOneTarget,
		TargetCount:        1,
		MinPlayers:         5,
		MinEventOrder:      1,
		IsPack:             true,
		Enabled:            true,
		PublicInstruction:  "Two players are receiving a shared intelligence result.",
		PrivateInstruction: "Choose one agent to receive the result with you.",
	}
}

func (r sharedIntelResolver) Begin(state *GameState) error {
	return beginOperation(state, r.Definition())
}

func (sharedIntelResolver) Resolve(state *GameState, command Command) error {
	active, ok := state.Players[state.ActivePlayerID]
	if !ok {
		return ErrInvalidTarget
	}
	targetID, err := requiredTarget(state, command)
	if err != nil {
		return err
	}
	target := state.Players[targetID]
	code := "AT_LEAST_ONE_VIRUS"
	if checkFaction(active) != FactionVirus && checkFaction(target) != FactionVirus {
		code = "NO_VIRUS_FOUND"
	}
	state.Operation.PrivateResults = map[string]OperationResult{
		active.ID: {Code: code, TargetPlayerID: targetID, Message: "Both selected players receive the same intelligence result."},
		target.ID: {Code: code, TargetPlayerID: active.ID, Message: "The selected active player shares the intelligence result with you."},
	}
	return nil
}

type chooseVoteShieldResolver struct{}

func (chooseVoteShieldResolver) Definition() OperationDefinition {
	return OperationDefinition{
		ID:                 "ChooseVoteShield",
		Name:               "Incriminating Evidence",
		InputKind:          OperationInputOneTarget,
		TargetCount:        1,
		MinPlayers:         5,
		MinEventOrder:      2,
		IsPack:             true,
		Enabled:            true,
		PublicInstruction:  "The active player must choose another player who will add one vote against them or shield them from one vote.",
		PrivateInstruction: "Choose one agent to review the incriminating evidence.",
	}
}

func (r chooseVoteShieldResolver) Begin(state *GameState) error {
	if err := beginOperation(state, r.Definition()); err != nil {
		return err
	}
	state.Operation.InputOwnerID = state.ActivePlayerID
	state.Operation.Step = 1
	return nil
}

func (chooseVoteShieldResolver) Resolve(state *GameState, command Command) error {
	if state.Operation == nil {
		return ErrNotAllowed
	}
	if state.Operation.Step == 1 {
		targetID, err := requiredTarget(state, command)
		if err != nil {
			return err
		}
		state.Operation.TargetPlayerID = targetID
		state.Operation.TargetPlayerIDs = []string{targetID}
		state.Operation.InputOwnerID = targetID
		state.Operation.Step = 2
		return nil
	}

	activeID := state.ActivePlayerID
	active, ok := state.Players[activeID]
	if !ok {
		return ErrInvalidTarget
	}
	choice := strings.ToUpper(strings.TrimSpace(command.Choice))
	code := ""
	msg := ""
	if choice == "VOTE" || choice == "EXTRA_SUSPICION" {
		active.Statuses = append(active.Statuses, "EXTRA_SUSPICION")
		code = "EXTRA_SUSPICION_ASSIGNED"
		msg = "You placed an extra vote of suspicion against the active player."
	} else if choice == "SHIELD" || choice == "VOTE_SHIELD" {
		active.Statuses = append(active.Statuses, "VOTE_SHIELD")
		code = "VOTE_SHIELD_ASSIGNED"
		msg = "You granted a vote shield to the active player."
	} else {
		return ErrInvalidTarget
	}
	state.Players[active.ID] = active
	targetID := state.Operation.TargetPlayerID
	state.Operation.PrivateResults = map[string]OperationResult{
		targetID: {
			Code:           code,
			TargetPlayerID: active.ID,
			Message:        msg,
		},
	}
	return nil
}

type defectorResolver struct{}

func (defectorResolver) Definition() OperationDefinition {
	return OperationDefinition{
		ID:                 "Defect",
		Name:               "Defector",
		InputKind:          OperationInputChoice,
		MinPlayers:         5,
		MinEventOrder:      1,
		IsPack:             true,
		Enabled:            true,
		PublicInstruction:  "The active player is deciding whether to defect.",
		PrivateInstruction: "Choose DEFECT or STAY.",
	}
}

func (r defectorResolver) Begin(state *GameState) error { return beginOperation(state, r.Definition()) }

func (defectorResolver) Resolve(state *GameState, command Command) error {
	active, ok := state.Players[state.ActivePlayerID]
	if !ok {
		return ErrInvalidTarget
	}
	choice := strings.ToUpper(command.Choice)
	if choice != "DEFECT" && choice != "STAY" {
		return ErrInvalidTarget
	}
	result := OperationResult{Message: "Your defector decision has been recorded."}
	if choice == "DEFECT" && isLoyalist(active) {
		// Defecting is still an operation trying to move a Loyalist off their
		// agency, so it is cancelled like any other.
		state.Operation.PrivateResults = map[string]OperationResult{active.ID: {
			Code:        "AGENCY_HELD",
			YourFaction: active.Faction,
			Message:     "You could not go through with it. Your loyalty held and your agency did not change.",
		}}
		return nil
	}
	if choice == "DEFECT" {
		if active.Faction == FactionVirus {
			active.Faction = FactionService
			active.ObjectiveKind = "RED_DEFECTOR"
			active.Statuses = append(active.Statuses, "RED_DEFECTOR")
		} else {
			active.Faction = FactionVirus
			active.CanVote = false
			active.Statuses = append(active.Statuses, "BLUE_DEFECTOR")
		}
		result.Code = "DEFECTED"
		result.YourFaction = active.Faction
	} else {
		result.Code = "STAYED"
		result.YourFaction = active.Faction
	}
	state.Players[active.ID] = active
	state.Operation.PrivateResults = map[string]OperationResult{active.ID: result}
	return nil
}

type votePowerResolver struct{}

func (votePowerResolver) Definition() OperationDefinition {
	return OperationDefinition{ID: "Power", Name: "Vote of Confidence", InputKind: OperationInputOneTarget, TargetCount: 1, MinPlayers: 5, PublicInstruction: "The active player is assigning voting power.", PrivateInstruction: "Choose one target whose vote will count twice."}
}

func (r votePowerResolver) Begin(state *GameState) error {
	return beginOperation(state, r.Definition())
}

func (votePowerResolver) Resolve(state *GameState, command Command) error {
	targetID, err := requiredTarget(state, command)
	if err != nil {
		return err
	}
	target := state.Players[targetID]
	if target.VotingPower == 0 {
		target.VotingPower = 1
	}
	target.VotingPower++
	target.Statuses = append(target.Statuses, "DOUBLE_VOTE")
	state.Players[targetID] = target
	state.Operation.PrivateResults = map[string]OperationResult{state.ActivePlayerID: {Code: "DOUBLE_VOTE_ASSIGNED", TargetPlayerID: targetID, Message: "The selected player's later vote counts twice."}}
	return nil
}

type startRumorsResolver struct{}

func (startRumorsResolver) Definition() OperationDefinition {
	return OperationDefinition{ID: "Vote", Name: "Start Rumors", InputKind: OperationInputOneTarget, TargetCount: 1, MinPlayers: 5, PublicInstruction: "The active player is assigning an extra accusation vote.", PrivateInstruction: "Choose one target who receives extra suspicion."}
}

func (r startRumorsResolver) Begin(state *GameState) error {
	return beginOperation(state, r.Definition())
}

func (startRumorsResolver) Resolve(state *GameState, command Command) error {
	targetID, err := requiredTarget(state, command)
	if err != nil {
		return err
	}
	target := state.Players[targetID]
	target.Statuses = append(target.Statuses, "EXTRA_SUSPICION")
	state.Players[targetID] = target
	state.Operation.PrivateResults = map[string]OperationResult{state.ActivePlayerID: {Code: "EXTRA_SUSPICION_ASSIGNED", TargetPlayerID: targetID, Message: "The selected player carries one additional vote against them."}}
	return nil
}

type confirmResolver struct{}

func (confirmResolver) Definition() OperationDefinition {
	return OperationDefinition{ID: "Confirm", Name: "Paycheck", InputKind: OperationInputPrivateInfo, MinPlayers: 5, PublicInstruction: "The active player received a private agency confirmation.", PrivateInstruction: "The server confirms your current agency."}
}

func (r confirmResolver) Begin(state *GameState) error {
	return beginPrivateOperation(state, r.Definition(), r.Resolve)
}

func (confirmResolver) Resolve(state *GameState, _ Command) error {
	active, ok := state.Players[state.ActivePlayerID]
	if !ok {
		return ErrInvalidTarget
	}
	code := "AGENCY_CHANGED"
	if active.InitialFaction == active.Faction {
		code = "AGENCY_UNCHANGED"
	}
	state.Operation.PrivateResults = map[string]OperationResult{active.ID: {Code: code, YourFaction: active.Faction, Message: "The server confirms your current agency."}}
	return nil
}

type negativeVoteResolver struct{}

func (negativeVoteResolver) Definition() OperationDefinition {
	return OperationDefinition{ID: "NegativeVote", Name: "Burn Evidence", InputKind: OperationInputOneTarget, TargetCount: 1, MinPlayers: 5, PublicInstruction: "The active player is assigning a vote shield.", PrivateInstruction: "Choose one target to protect from one vote."}
}

func (r negativeVoteResolver) Begin(state *GameState) error {
	return beginOperation(state, r.Definition())
}

func (negativeVoteResolver) Resolve(state *GameState, command Command) error {
	targetID, err := requiredTarget(state, command)
	if err != nil {
		return err
	}
	target := state.Players[targetID]
	target.Statuses = append(target.Statuses, "VOTE_SHIELD")
	state.Players[targetID] = target
	state.Operation.PrivateResults = map[string]OperationResult{state.ActivePlayerID: {Code: "VOTE_SHIELD_ASSIGNED", TargetPlayerID: targetID, Message: "The selected player receives one vote shield."}}
	return nil
}

type brigResolver struct{}

func (brigResolver) Definition() OperationDefinition {
	return OperationDefinition{ID: "Brig", Name: "Brig", InputKind: OperationInputOneTarget, TargetCount: 1, MinPlayers: 5, RecoveredOnly: true, PublicInstruction: "A recovered Brig variant is being displayed.", PrivateInstruction: "Choose one player to silence."}
}

func (r brigResolver) Begin(state *GameState) error { return beginOperation(state, r.Definition()) }

func (brigResolver) Resolve(state *GameState, command Command) error {
	targetID, err := requiredTarget(state, command)
	if err != nil {
		return err
	}
	target := state.Players[targetID]
	target.CanVote = false
	target.Statuses = append(target.Statuses, "SILENCED")
	state.Players[targetID] = target
	state.Operation.PrivateResults = map[string]OperationResult{state.ActivePlayerID: {Code: "PLAYER_SILENCED", TargetPlayerID: targetID, Message: "The selected player cannot vote this round."}}
	return nil
}

type earlyVoteResolver struct{}

func (earlyVoteResolver) Definition() OperationDefinition {
	return OperationDefinition{ID: "EarlyVote", Name: "Early Vote", InputKind: OperationInputOneTarget, TargetCount: 1, MinPlayers: 5, RecoveredOnly: true, PublicInstruction: "A recovered Early Vote variant is being displayed.", PrivateInstruction: "Choose the recovered operation's target."}
}

func (r earlyVoteResolver) Begin(state *GameState) error {
	return beginOperation(state, r.Definition())
}

func (earlyVoteResolver) Resolve(state *GameState, command Command) error {
	targetID, err := requiredTarget(state, command)
	if err != nil {
		return err
	}
	state.Operation.PrivateResults = map[string]OperationResult{state.ActivePlayerID: {Code: "EARLY-VOTE", TargetPlayerID: targetID, Message: "The recovered compatibility operation recorded its target."}}
	return nil
}

type hunterResolver struct{}

func (hunterResolver) Definition() OperationDefinition {
	return OperationDefinition{ID: "Hunter", Name: "Hunter", InputKind: OperationInputOneTarget, TargetCount: 1, MinPlayers: 5, RecoveredOnly: true, PublicInstruction: "A recovered Hunter variant is being displayed.", PrivateInstruction: "Choose the recovered operation's target."}
}

func (r hunterResolver) Begin(state *GameState) error { return beginOperation(state, r.Definition()) }

func (hunterResolver) Resolve(state *GameState, command Command) error {
	targetID, err := requiredTarget(state, command)
	if err != nil {
		return err
	}
	state.Operation.PrivateResults = map[string]OperationResult{state.ActivePlayerID: {Code: "HUNTER", TargetPlayerID: targetID, Message: "The recovered compatibility operation recorded its target."}}
	return nil
}

type ambassadorResolver struct{}

func (ambassadorResolver) Definition() OperationDefinition {
	return OperationDefinition{ID: "Ambassador", Name: "Ambassador", InputKind: OperationInputPrivateInfo, MinPlayers: 5, RecoveredOnly: true, PublicInstruction: "A recovered Ambassador variant is being displayed.", PrivateInstruction: "The server supplies the recovered Ambassador result."}
}

func (r ambassadorResolver) Begin(state *GameState) error {
	return beginPrivateOperation(state, r.Definition(), r.Resolve)
}

func (ambassadorResolver) Resolve(state *GameState, _ Command) error {
	if _, ok := state.Players[state.ActivePlayerID]; !ok {
		return ErrInvalidTarget
	}
	state.Operation.PrivateResults = map[string]OperationResult{state.ActivePlayerID: {Code: "AMBASSADOR", Message: "The recovered compatibility marker has been recorded."}}
	return nil
}

type lastEventResolver struct{}

func (lastEventResolver) Definition() OperationDefinition {
	return OperationDefinition{ID: "LastEvent", Name: "Last Event", InputKind: OperationInputPrivateInfo, MinPlayers: 5, RecoveredOnly: true, PublicInstruction: "A recovered Last Event marker is being displayed.", PrivateInstruction: "The server identifies the final operation marker."}
}

func (r lastEventResolver) Begin(state *GameState) error {
	return beginPrivateOperation(state, r.Definition(), r.Resolve)
}

func (lastEventResolver) Resolve(state *GameState, _ Command) error {
	if _, ok := state.Players[state.ActivePlayerID]; !ok {
		return ErrInvalidTarget
	}
	state.Operation.PrivateResults = map[string]OperationResult{state.ActivePlayerID: {Code: "LAST-EVENT", Message: "The recovered compatibility marker has been recorded."}}
	return nil
}

func explicitOperationResolvers() []OperationResolver {
	return []OperationResolver{
		recruitmentResolver{},
		strainResolver{},
		grudgeResolver{},
		infatuationResolver{},
		flipResolver{},
		hiddenTipResolver{},
		oneOfTwoResolver{},
		twoFriendsResolver{},
		undercoverResolver{},
		sharedIntelResolver{},
		chooseVoteShieldResolver{},
		defectorResolver{},
		votePowerResolver{},
		startRumorsResolver{},
		confirmResolver{},
		negativeVoteResolver{},
		ambassadorResolver{},
		brigResolver{},
		earlyVoteResolver{},
		hunterResolver{},
		lastEventResolver{},
	}
}
