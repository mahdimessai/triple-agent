package game

import (
	"fmt"
	"strings"
)

const (
	hiddenAgendaKind              = "HiddenAgenda"
	hiddenAgendaName              = "Hidden Agenda"
	hiddenAgendaPublicInstruction = "The active player has new orders from up top. They could switch sides, gain a new win condition, or learn another agent's agency."
)

type operationDefinition struct {
	ID                 string             `json:"id"`
	Name               string             `json:"name"`
	InputKind          OperationInputKind `json:"input_kind"`
	TargetCount        int                `json:"target_count,omitempty"`
	MinPlayers         int                `json:"min_players"`
	Category           int                `json:"category,omitempty"`
	MinEventOrder      int                `json:"min_event_order,omitempty"`
	Hidden             bool               `json:"hidden,omitempty"`
	IsPack             bool               `json:"is_pack,omitempty"`
	PublicInstruction  string             `json:"public_instruction"`
	PrivateInstruction string             `json:"private_instruction"`
}

type operation struct {
	definition operationDefinition
	begin      func(*State) error
	resolve    func(*State, Command) error
}

var operations map[string]operation

func init() {
	operations = map[string]operation{
		"Grudge": {
			definition: operationDefinition{ID: "Grudge", Name: "Grudge", InputKind: OperationInputPrivateInfo, MinPlayers: 5, MinEventOrder: 1, Category: 1, Hidden: true, PublicInstruction: "The active player received a private objective.", PrivateInstruction: "You now win if and only if your assigned target is imprisoned."},
			begin:      beginGrudge,
		},
		"Infatuation": {
			definition: operationDefinition{ID: "Infatuation", Name: "Infatuation", InputKind: OperationInputPrivateInfo, MinPlayers: 5, MinEventOrder: 1, Category: 1, Hidden: true, PublicInstruction: "The active player received a private objective.", PrivateInstruction: "You now win if and only if your assigned target wins at the end of the round."},
			begin:      beginInfatuation,
		},
		"Share": {
			definition: operationDefinition{ID: "Share", Name: "Confession", InputKind: OperationInputOneTarget, TargetCount: 1, MinPlayers: 5, MinEventOrder: 1, PublicInstruction: "The active player is sharing one private agency fact.", PrivateInstruction: "Choose one other agent who may view your agency information."},
			begin:      beginSimple("Share"), resolve: resolveShare,
		},
		"Detector": {
			definition: operationDefinition{ID: "Detector", Name: "Secret Intel", InputKind: OperationInputTwoTargets, TargetCount: 2, MinPlayers: 4, MinEventOrder: 1, PublicInstruction: "The active player is reviewing a two-agent intelligence check.", PrivateInstruction: "Choose two other agents. You learn whether either one is VIRUS."},
			begin:      beginSimple("Detector"), resolve: resolveDetector,
		},
		"Strain": {
			definition: operationDefinition{ID: "Strain", Name: "Operation: Scapegoat", InputKind: OperationInputPrivateInfo, MinPlayers: 5, MinEventOrder: 1, Category: 1, Hidden: true, PublicInstruction: "The active player received a private objective.", PrivateInstruction: "You win only if you are imprisoned."},
			begin:      beginStrain,
		},
		"Flip": {
			definition: operationDefinition{ID: "Flip", Name: "Sleeper Agent", InputKind: OperationInputPrivateInfo, MinPlayers: 5, MinEventOrder: 1, Category: 1, Hidden: true, PublicInstruction: "The active player's current agency is being re-evaluated.", PrivateInstruction: "Your current agency may change; your initial agency stays fixed."},
			begin:      beginFlip,
		},
		"HiddenOneRandom": {
			definition: operationDefinition{ID: "HiddenOneRandom", Name: "Secret Tip", InputKind: OperationInputPrivateInfo, MinPlayers: 5, MinEventOrder: 1, Category: 1, Hidden: true, PublicInstruction: "The active player received a private tip.", PrivateInstruction: "Your source reveals one other agent's agency."},
			begin:      beginHiddenTip,
		},
		"TwoFriends": {
			definition: operationDefinition{ID: "TwoFriends", Name: "Old Photographs", InputKind: OperationInputPrivateInfo, MinPlayers: 5, MinEventOrder: 1, PublicInstruction: "The active player received evidence about two players who worked for the same agency at the start.", PrivateInstruction: "The server selected two names who worked for the same agency at the start."},
			begin:      beginTwoFriends,
		},
		"OneOfTwo": {
			definition: operationDefinition{ID: "OneOfTwo", Name: "Danish Intelligence", InputKind: OperationInputPrivateInfo, MinPlayers: 4, MinEventOrder: 1, PublicInstruction: "The active player received a two-name intelligence intercept.", PrivateInstruction: "The server selected two names and prepared a private result."},
			begin:      beginOneOfTwo,
		},
		"OneRandom": {
			definition: operationDefinition{ID: "OneRandom", Name: "Anonymous Tip", InputKind: OperationInputPrivateInfo, MinPlayers: 5, MinEventOrder: 1, PublicInstruction: "The active player has received a private source message.", PrivateInstruction: "Your source reveals the agency of one other agent."},
			begin:      beginAnonymousTip,
		},
		"Swap": {
			definition: operationDefinition{ID: "Swap", Name: "Spy Transfer", InputKind: OperationInputOneTarget, TargetCount: 1, MinPlayers: 5, MinEventOrder: 1, IsPack: true, PublicInstruction: "The active player is choosing an exchange target.", PrivateInstruction: "Choose one other agent. You and that agent secretly exchange agencies."},
			begin:      beginSimple("Swap"), resolve: resolveSwap,
		},
		"Undercover": {
			definition: operationDefinition{ID: "Undercover", Name: "Deep Undercover", InputKind: OperationInputOneTarget, TargetCount: 1, MinPlayers: 5, MinEventOrder: 1, IsPack: true, PublicInstruction: "The active player is following one agent undercover.", PrivateInstruction: "Choose one agent to investigate."},
			begin:      beginSimple("Undercover"), resolve: resolveUndercover,
		},
		"InfoForTwo": {
			definition: operationDefinition{ID: "InfoForTwo", Name: "Unfortunate Encounter", InputKind: OperationInputOneTarget, TargetCount: 1, MinPlayers: 5, MinEventOrder: 1, IsPack: true, PublicInstruction: "Two players are receiving a shared intelligence result.", PrivateInstruction: "Choose one agent to receive the result with you."},
			begin:      beginSimple("InfoForTwo"), resolve: resolveSharedIntel,
		},
		"ChooseVoteShield": {
			definition: operationDefinition{ID: "ChooseVoteShield", Name: "Incriminating Evidence", InputKind: OperationInputOneTarget, TargetCount: 1, MinPlayers: 5, MinEventOrder: 2, IsPack: true, PublicInstruction: "The active player must choose another player who will add one vote against them or shield them from one vote.", PrivateInstruction: "Choose one agent to review the incriminating evidence."},
			begin:      beginChooseVoteShield, resolve: resolveChooseVoteShield,
		},
		"Defect": {
			definition: operationDefinition{ID: "Defect", Name: "Defector", InputKind: OperationInputChoice, MinPlayers: 5, MinEventOrder: 1, IsPack: true, PublicInstruction: "The active player is deciding whether to defect.", PrivateInstruction: "Choose DEFECT or STAY."},
			begin:      beginSimple("Defect"), resolve: resolveDefector,
		},
	}
}

// operationOrder is stable presentation/deck iteration order. The map is the
// executable source of truth; this slice contains only live operation IDs.
var operationOrder = []string{
	"Grudge", "Infatuation", "Share", "Detector", "Strain", "Flip", "HiddenOneRandom", "TwoFriends", "OneOfTwo", "OneRandom",
	"Swap", "Undercover", "InfoForTwo", "ChooseVoteShield", "Defect",
}

func operationDefinitionFor(kind string) (operationDefinition, bool) {
	op, ok := operations[strings.TrimSpace(kind)]
	if !ok {
		return operationDefinition{}, false
	}
	return op.definition, true
}

func defaultEnabledOperations() map[string]bool {
	return map[string]bool{
		"Grudge": true, "Infatuation": true, "Share": true, "Detector": true, "Strain": true,
		"Flip": true, "HiddenOneRandom": true, "TwoFriends": true, "OneOfTwo": true, "OneRandom": true,
		"Swap": false, "Undercover": false, "InfoForTwo": false, "ChooseVoteShield": false, "Defect": false,
	}
}

func beginSimple(kind string) func(*State) error {
	return func(state *State) error {
		return beginOperation(state, operations[kind].definition)
	}
}

func beginOperation(state *State, definition operationDefinition) error {
	if len(state.PlayerOrder) == 0 {
		return ErrNotEnoughPlayers
	}
	activeID := activePlayerID(*state)
	if activeID == "" {
		return ErrNotEnoughPlayers
	}
	state.ActivePlayerID = activeID
	state.Operation = &OperationState{Kind: definition.ID, InputOwnerID: activeID, Step: 1}
	return nil
}

func beginPrivate(state *State, kind string, resolve func(*State) error) error {
	if err := beginOperation(state, operations[kind].definition); err != nil {
		return err
	}
	return resolve(state)
}

func beginAnonymousTip(state *State) error {
	return beginPrivate(state, "OneRandom", func(state *State) error {
		activeID := state.ActivePlayerID
		targetID := chooseRandomOther(state, activeID)
		if targetID == "" {
			return ErrInvalidTarget
		}
		state.Operation.TargetPlayerID = targetID
		state.Operation.PrivateResults = map[string]OperationResult{activeID: {
			Code: "FACTION_REVEALED", TargetPlayerID: targetID, TargetFaction: checkFaction(state.Players[targetID]), Message: "Your source reveals the agency of one other agent.",
		}}
		return nil
	})
}

func beginStrain(state *State) error {
	return beginPrivate(state, "Strain", func(state *State) error {
		active := state.Players[state.ActivePlayerID]
		active.ObjectiveKind = "IMPRISON_SELF"
		active.Statuses = append(active.Statuses, "STRAIN")
		state.Players[active.ID] = active
		state.Operation.PrivateResults = map[string]OperationResult{active.ID: {Code: "OBJECTIVE_ASSIGNED", Message: "You win only if you are imprisoned."}}
		return nil
	})
}

func beginGrudge(state *State) error {
	return beginPrivate(state, "Grudge", func(state *State) error {
		active := state.Players[state.ActivePlayerID]
		targetID := chooseRandomOther(state, active.ID)
		if targetID == "" {
			return ErrInvalidTarget
		}
		active.ObjectiveKind = "IMPRISON_TARGET"
		active.ObjectiveTarget = targetID
		active.Statuses = append(active.Statuses, "GRUDGE")
		state.Players[active.ID] = active
		state.Operation.TargetPlayerID = targetID
		state.Operation.TargetPlayerIDs = []string{targetID}
		state.Operation.PrivateResults = map[string]OperationResult{active.ID: {Code: "GRUDGE_TARGET_ASSIGNED", TargetPlayerID: targetID, Message: "You win only if your assigned target is imprisoned."}}
		return nil
	})
}

func beginInfatuation(state *State) error {
	return beginPrivate(state, "Infatuation", func(state *State) error {
		active := state.Players[state.ActivePlayerID]
		targetID := chooseRandomOther(state, active.ID)
		if targetID == "" {
			return ErrInvalidTarget
		}
		active.ObjectiveKind = "TARGET_WINS"
		active.ObjectiveTarget = targetID
		active.Statuses = append(active.Statuses, "INFATUATION")
		state.Players[active.ID] = active
		state.Operation.TargetPlayerID = targetID
		state.Operation.TargetPlayerIDs = []string{targetID}
		state.Operation.PrivateResults = map[string]OperationResult{active.ID: {Code: "INFATUATION_TARGET_ASSIGNED", TargetPlayerID: targetID, Message: "You win only if your assigned target wins the round."}}
		return nil
	})
}

func beginFlip(state *State) error {
	return beginPrivate(state, "Flip", func(state *State) error {
		active := state.Players[state.ActivePlayerID]
		if isLoyalist(active) {
			state.Operation.PrivateResults = map[string]OperationResult{active.ID: {Code: "AGENCY_HELD", YourFaction: active.Faction, Message: "Your loyalty held. Your agency did not change."}}
			return nil
		}
		result := OperationResult{Message: "Your current agency changed; your initial agency remains unchanged."}
		if active.Faction == FactionVirus {
			setFaction(state, active.ID, FactionService)
			active = state.Players[active.ID]
			active.Statuses = append(active.Statuses, "CURED")
			result.Code = "CURED"
		} else {
			setFaction(state, active.ID, FactionVirus)
			active = state.Players[active.ID]
			active.Statuses = append(active.Statuses, "INFECTED")
			result.Code = "INFECTED"
		}
		result.YourFaction = active.Faction
		state.Players[active.ID] = active
		state.Operation.PrivateResults = map[string]OperationResult{active.ID: result}
		return nil
	})
}

func beginHiddenTip(state *State) error {
	return beginPrivate(state, "HiddenOneRandom", func(state *State) error {
		activeID := state.ActivePlayerID
		targetID := chooseRandomOther(state, activeID)
		if targetID == "" {
			return ErrInvalidTarget
		}
		state.Operation.PrivateResults = map[string]OperationResult{activeID: {Code: "FACTION_REVEALED", TargetPlayerID: targetID, TargetFaction: checkFaction(state.Players[targetID]), Message: "Your source reveals one other agent's agency."}}
		return nil
	})
}

func beginOneOfTwo(state *State) error {
	return beginPrivate(state, "OneOfTwo", func(state *State) error {
		activeID := state.ActivePlayerID
		pair := chooseRandomVirusAndNonVirus(state, activeID)
		if len(pair) != 2 {
			return ErrInvalidTarget
		}
		state.Operation.PrivateResults = map[string]OperationResult{activeID: {
			Code: "ONE_VIRUS_ONE_SERVICE", TargetPlayerIDs: pair, Message: "The server prepared a two-name intelligence result.",
		}}
		return nil
	})
}

func beginTwoFriends(state *State) error {
	return beginPrivate(state, "TwoFriends", func(state *State) error {
		activeID := state.ActivePlayerID
		pair := chooseRandomSameInitialAgencyPair(state, activeID)
		if len(pair) != 2 {
			return ErrInvalidTarget
		}
		state.Operation.PrivateResults = map[string]OperationResult{activeID: {Code: "SAME_INITIAL_AGENCY", TargetPlayerIDs: pair, Message: "The photographs show two agents who worked for the same agency at the start."}}
		return nil
	})
}

func beginChooseVoteShield(state *State) error {
	if err := beginOperation(state, operations["ChooseVoteShield"].definition); err != nil {
		return err
	}
	state.Operation.InputOwnerID = state.ActivePlayerID
	state.Operation.Step = 1
	return nil
}

func resolveSwap(state *State, command Command) error {
	targetID, err := requiredTarget(state, command)
	if err != nil {
		return err
	}
	active := state.Players[state.ActivePlayerID]
	target := state.Players[targetID]
	activeFaction, targetFaction := active.Faction, target.Faction
	setFaction(state, active.ID, targetFaction)
	setFaction(state, target.ID, activeFaction)
	active, target = state.Players[active.ID], state.Players[target.ID]
	state.Operation.TargetPlayerID = target.ID
	state.Operation.TargetPlayerIDs = []string{target.ID}
	state.Operation.PrivateResults = map[string]OperationResult{
		active.ID: {Code: "FACTIONS_EXCHANGED", TargetPlayerID: target.ID, TargetPlayerIDs: []string{target.ID}, TargetFaction: target.Faction, OtherPlayerID: target.ID, OtherFaction: target.Faction, YourFaction: active.Faction, Message: "You exchanged agencies with the selected agent."},
		target.ID: {Code: "FACTIONS_EXCHANGED", TargetPlayerID: active.ID, TargetPlayerIDs: []string{active.ID}, TargetFaction: active.Faction, OtherPlayerID: active.ID, OtherFaction: active.Faction, YourFaction: target.Faction, Message: "Your agency was exchanged with the active agent."},
	}
	return nil
}

func resolveDetector(state *State, command Command) error {
	targets, err := requiredTargets(state, command, 2)
	if err != nil {
		return err
	}
	hasVirus := false
	for _, id := range targets {
		if checkFaction(state.Players[id]) == FactionVirus {
			hasVirus = true
			break
		}
	}
	code, message := "NO_VIRUS_FOUND", "Secret Intel found no VIRUS agent among the two selected players."
	if hasVirus {
		code, message = "AT_LEAST_ONE_VIRUS", "Secret Intel found at least one VIRUS agent among the two selected players."
	}
	state.Operation.TargetPlayerIDs = append([]string(nil), targets...)
	state.Operation.PrivateResults = map[string]OperationResult{state.ActivePlayerID: {Code: code, TargetPlayerIDs: append([]string(nil), targets...), Message: message}}
	return nil
}

func resolveShare(state *State, command Command) error {
	targetID, err := requiredTarget(state, command)
	if err != nil {
		return err
	}
	active := state.Players[state.ActivePlayerID]
	activeFaction := checkFaction(active)
	state.Operation.TargetPlayerID = targetID
	state.Operation.TargetPlayerIDs = []string{targetID}
	state.Operation.PrivateResults = map[string]OperationResult{targetID: {Code: "AGENCY_SHARED", TargetPlayerID: targetID, OtherPlayerID: active.ID, OtherFaction: activeFaction, Message: fmt.Sprintf("%s is %s.", active.Name, activeFaction)}}
	return nil
}

func resolveUndercover(state *State, command Command) error {
	targetID, err := requiredTarget(state, command)
	if err != nil {
		return err
	}
	active := state.Players[state.ActivePlayerID]
	target := state.Players[targetID]
	result := OperationResult{TargetPlayerID: targetID, TargetFaction: checkFaction(target), Message: "The undercover investigation is complete."}
	if checkFaction(target) == FactionVirus {
		if setFaction(state, active.ID, FactionVirus) {
			active = state.Players[active.ID]
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
	state.Players[active.ID] = active
	state.Operation.PrivateResults = map[string]OperationResult{active.ID: result}
	return nil
}

func resolveSharedIntel(state *State, command Command) error {
	targetID, err := requiredTarget(state, command)
	if err != nil {
		return err
	}
	active := state.Players[state.ActivePlayerID]
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

func resolveChooseVoteShield(state *State, command Command) error {
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
	active := state.Players[state.ActivePlayerID]
	choice := strings.ToUpper(strings.TrimSpace(command.Choice))
	var code, message string
	switch choice {
	case "VOTE", "EXTRA_SUSPICION":
		active.Statuses = append(active.Statuses, "EXTRA_SUSPICION")
		code, message = "EXTRA_SUSPICION_ASSIGNED", "You placed an extra vote of suspicion against the active player."
	case "SHIELD", "VOTE_SHIELD":
		active.Statuses = append(active.Statuses, "VOTE_SHIELD")
		code, message = "VOTE_SHIELD_ASSIGNED", "You granted a vote shield to the active player."
	default:
		return ErrInvalidTarget
	}
	state.Players[active.ID] = active
	targetID := state.Operation.TargetPlayerID
	state.Operation.PrivateResults = map[string]OperationResult{targetID: {Code: code, TargetPlayerID: active.ID, Message: message}}
	return nil
}

func resolveDefector(state *State, command Command) error {
	active := state.Players[state.ActivePlayerID]
	choice := strings.ToUpper(strings.TrimSpace(command.Choice))
	if choice != "DEFECT" && choice != "STAY" {
		return ErrInvalidTarget
	}
	if choice == "DEFECT" && isLoyalist(active) {
		state.Operation.PrivateResults = map[string]OperationResult{active.ID: {Code: "AGENCY_HELD", YourFaction: active.Faction, Message: "You could not go through with it. Your loyalty held and your agency did not change."}}
		return nil
	}
	result := OperationResult{Message: "Your defector decision has been recorded."}
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
	} else {
		result.Code = "STAYED"
	}
	result.YourFaction = active.Faction
	state.Players[active.ID] = active
	state.Operation.PrivateResults = map[string]OperationResult{active.ID: result}
	return nil
}

func operationFor(kind string) (operation, error) {
	op, ok := operations[strings.TrimSpace(kind)]
	if !ok {
		return operation{}, ErrUnknownOperation
	}
	return op, nil
}

func beginPlannedOperation(state *State) error {
	op, err := operationFor(state.PlannedOperation)
	if err != nil {
		return err
	}
	if err := op.begin(state); err != nil {
		return err
	}
	state.OperationsDealt = append(state.OperationsDealt, state.ActivePlayerID)
	state.OperationDeals++
	if op.definition.InputKind == OperationInputNone || op.definition.InputKind == OperationInputPrivateInfo {
		state.Phase = PhaseOperationResult
	} else {
		state.Phase = PhaseOperationInput
	}
	return nil
}

func resolveCurrentOperation(state *State, command Command) error {
	if state.Operation == nil {
		return ErrNotAllowed
	}
	op, err := operationFor(state.Operation.Kind)
	if err != nil || op.resolve == nil {
		return ErrNotAllowed
	}
	return op.resolve(state, command)
}

func operationForStart(state *State, requested string) (operation, error) {
	initOperationQueue(state)
	if err := initOperationDeck(state); err != nil {
		return operation{}, err
	}
	state.OperationDealTarget = connectedPlayerCount(*state)
	state.OperationDeals = 0
	firstRecipient := nextOperationRecipient(state)
	if firstRecipient == "" {
		return operation{}, ErrNotEnoughPlayers
	}
	state.ActivePlayerID = firstRecipient
	return takeOperationFromDeck(state, firstRecipient, requested, 1)
}

func connectedPlayerCount(state State) int {
	count := 0
	for _, id := range state.PlayerOrder {
		if player, ok := state.Players[id]; ok && player.Connected {
			count++
		}
	}
	return count
}

func initOperationQueue(state *State) {
	queue := append([]string(nil), state.PlayerOrder...)
	for i := len(queue) - 1; i > 0; i-- {
		j := nextRandom(state, i+1)
		queue[i], queue[j] = queue[j], queue[i]
	}
	state.OperationQueue = queue
	state.OperationQueueIndex = 0
}

func nextOperationRecipient(state *State) string {
	if len(state.OperationQueue) == 0 {
		return ""
	}
	for attempts := 0; attempts < len(state.OperationQueue); attempts++ {
		index := state.OperationQueueIndex % len(state.OperationQueue)
		state.OperationQueueIndex = (index + 1) % len(state.OperationQueue)
		id := state.OperationQueue[index]
		if player, ok := state.Players[id]; ok && player.Connected {
			return id
		}
	}
	return ""
}

func operationDeckSlots(state State) []string {
	slots := make([]string, 0, len(operationOrder))
	hidden := false
	for _, id := range operationOrder {
		definition := operations[id].definition
		if !dealableOperation(state, definition) {
			continue
		}
		if definition.Hidden {
			hidden = true
			continue
		}
		slots = append(slots, definition.ID)
	}
	if hidden {
		slots = append(slots, hiddenAgendaKind)
	}
	return slots
}

func initOperationDeck(state *State) error {
	state.OperationDeck = nil
	return refillOperationDeck(state)
}

func refillOperationDeck(state *State) error {
	slots := operationDeckSlots(*state)
	if len(slots) == 0 {
		return ErrNoEligibleOperations
	}
	for i := len(slots) - 1; i > 0; i-- {
		j := nextRandom(state, i+1)
		slots[i], slots[j] = slots[j], slots[i]
	}
	if len(slots) > 1 && state.OperationLastKind != "" && slots[0] == state.OperationLastKind {
		slots[0], slots[1] = slots[1], slots[0]
	}
	state.OperationDeck = slots
	return nil
}

func removeOperationDeckSlot(state *State, index int) string {
	kind := state.OperationDeck[index]
	state.OperationDeck = append(state.OperationDeck[:index], state.OperationDeck[index+1:]...)
	state.OperationLastKind = kind
	return kind
}

func hiddenAgendaMembers(state State, recipientID string, eventOrder int) []operation {
	player := state.Players[recipientID]
	members := make([]operation, 0, 5)
	for _, id := range operationOrder {
		op := operations[id]
		definition := op.definition
		if !definition.Hidden || definition.MinEventOrder > eventOrder || !dealableOperation(state, definition) || playerHasCategory(player, definition.Category) {
			continue
		}
		members = append(members, op)
	}
	return members
}

func drawHidden(state *State, recipientID string, eventOrder int) (operation, bool) {
	members := hiddenAgendaMembers(*state, recipientID, eventOrder)
	if len(members) == 0 {
		return operation{}, false
	}
	return members[nextRandom(state, len(members))], true
}

func takeOperationFromDeck(state *State, recipientID, requested string, eventOrder int) (operation, error) {
	requested = strings.TrimSpace(requested)
	if requested != "" {
		if requested == hiddenAgendaKind || isHiddenAgendaMember(requested) {
			for index, slot := range state.OperationDeck {
				if slot != hiddenAgendaKind {
					continue
				}
				var chosen operation
				var ok bool
				if requested == hiddenAgendaKind {
					chosen, ok = drawHidden(state, recipientID, eventOrder)
				} else if candidate, exists := operations[requested]; exists && candidate.definition.Hidden && candidate.definition.MinEventOrder <= eventOrder && dealableOperation(*state, candidate.definition) && !playerHasCategory(state.Players[recipientID], candidate.definition.Category) {
					chosen, ok = candidate, true
				}
				if !ok {
					return operation{}, ErrNotAllowed
				}
				removeOperationDeckSlot(state, index)
				recordDealtOperation(state, recipientID, chosen.definition)
				return chosen, nil
			}
			return operation{}, ErrNotAllowed
		}
		chosen, ok := operations[requested]
		if !ok || chosen.definition.Hidden || chosen.definition.MinEventOrder > eventOrder || !dealableOperation(*state, chosen.definition) {
			return operation{}, ErrNotAllowed
		}
		for index, slot := range state.OperationDeck {
			if slot == chosen.definition.ID {
				removeOperationDeckSlot(state, index)
				recordDealtOperation(state, recipientID, chosen.definition)
				return chosen, nil
			}
		}
		return operation{}, ErrNotAllowed
	}

	if len(state.OperationDeck) == 0 {
		if err := refillOperationDeck(state); err != nil {
			return operation{}, err
		}
	}
	for index, slot := range state.OperationDeck {
		var chosen operation
		var ok bool
		if slot == hiddenAgendaKind {
			chosen, ok = drawHidden(state, recipientID, eventOrder)
		} else {
			chosen, ok = operations[slot]
			if ok && (chosen.definition.MinEventOrder > eventOrder || !dealableOperation(*state, chosen.definition)) {
				ok = false
			}
		}
		if !ok {
			continue
		}
		removeOperationDeckSlot(state, index)
		recordDealtOperation(state, recipientID, chosen.definition)
		return chosen, nil
	}
	return operation{}, ErrNoEligibleOperations
}

func hiddenAgendaMemberIDs() []string {
	ids := make([]string, 0, 5)
	for _, id := range operationOrder {
		if operations[id].definition.Hidden {
			ids = append(ids, id)
		}
	}
	return ids
}

func isHiddenAgendaMember(kind string) bool {
	op, ok := operations[kind]
	return ok && op.definition.Hidden
}

func hiddenAgendaEnabled(pool map[string]bool) bool {
	for _, id := range hiddenAgendaMemberIDs() {
		if pool[id] {
			return true
		}
	}
	return false
}

func countEnabledHiddenAgenda(pool map[string]bool) int {
	count := 0
	for _, id := range hiddenAgendaMemberIDs() {
		if pool[id] {
			count++
		}
	}
	return count
}

func countEnabledOperations(pool map[string]bool) int {
	count := 0
	for id, enabled := range pool {
		if _, live := operations[id]; live && enabled {
			count++
		}
	}
	return count
}

func dealableOperation(state State, definition operationDefinition) bool {
	return state.Settings.EnabledOperations[definition.ID] && len(state.PlayerOrder) >= definition.MinPlayers
}

func playerHasCategory(player Player, category int) bool {
	if category <= 0 {
		return false
	}
	for _, dealt := range player.DealtCategories {
		if dealt == category {
			return true
		}
	}
	return false
}

func recordDealtOperation(state *State, recipientID string, definition operationDefinition) {
	player := state.Players[recipientID]
	player.DealtOperations = append(player.DealtOperations, definition.ID)
	if definition.Category > 0 {
		player.DealtCategories = append(player.DealtCategories, definition.Category)
	}
	state.Players[recipientID] = player
}

func requiredTargets(state *State, command Command, count int) ([]string, error) {
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

func requiredTarget(state *State, command Command) (string, error) {
	targets, err := requiredTargets(state, command, 1)
	if err != nil {
		return "", err
	}
	return targets[0], nil
}

func operationTargetIDs(command Command) []string {
	if len(command.TargetIDs) > 0 {
		return append([]string(nil), command.TargetIDs...)
	}
	if command.TargetID != "" {
		return []string{command.TargetID}
	}
	return nil
}

func otherPlayerIDs(state State, playerID string) []string {
	players := make([]string, 0, len(state.PlayerOrder)-1)
	for _, id := range state.PlayerOrder {
		if id != playerID {
			players = append(players, id)
		}
	}
	return players
}

func chooseRandomOther(state *State, playerID string) string {
	players := otherPlayerIDs(*state, playerID)
	if len(players) == 0 {
		return ""
	}
	return players[nextRandom(state, len(players))]
}

func chooseRandomOthers(state *State, playerID string, count int) []string {
	players := otherPlayerIDs(*state, playerID)
	if len(players) < count {
		return nil
	}
	selected := make([]string, 0, count)
	for len(selected) < count {
		index := nextRandom(state, len(players))
		selected = append(selected, players[index])
		players = append(players[:index], players[index+1:]...)
	}
	return selected
}

func chooseRandomVirusAndNonVirus(state *State, playerID string) []string {
	virus := make([]string, 0)
	nonVirus := make([]string, 0)
	for _, id := range otherPlayerIDs(*state, playerID) {
		if checkFaction(state.Players[id]) == FactionVirus {
			virus = append(virus, id)
		} else {
			nonVirus = append(nonVirus, id)
		}
	}
	if len(virus) == 0 || len(nonVirus) == 0 {
		return nil
	}
	return []string{
		virus[nextRandom(state, len(virus))],
		nonVirus[nextRandom(state, len(nonVirus))],
	}
}

func chooseRandomSameInitialAgencyPair(state *State, playerID string) []string {
	players := otherPlayerIDs(*state, playerID)
	pairs := make([][2]string, 0)
	for index, leftID := range players {
		for _, rightID := range players[index+1:] {
			if checkInitialFaction(state.Players[leftID]) == checkInitialFaction(state.Players[rightID]) {
				pairs = append(pairs, [2]string{leftID, rightID})
			}
		}
	}
	if len(pairs) == 0 {
		return nil
	}
	pair := pairs[nextRandom(state, len(pairs))]
	return []string{pair[0], pair[1]}
}

func checkFaction(player Player) Faction {
	if player.ApparentFaction != nil {
		return *player.ApparentFaction
	}
	return player.Faction
}

func checkInitialFaction(player Player) Faction {
	if player.ApparentFaction != nil {
		return *player.ApparentFaction
	}
	return player.InitialFaction
}

func activePlayerID(state State) string {
	if state.ActivePlayerID != "" {
		if player, ok := state.Players[state.ActivePlayerID]; ok && player.Connected {
			return state.ActivePlayerID
		}
	}
	for _, id := range state.PlayerOrder {
		if state.Players[id].Connected {
			return id
		}
	}
	if len(state.PlayerOrder) > 0 {
		return state.PlayerOrder[0]
	}
	return ""
}
