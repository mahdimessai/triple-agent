package domain

import (
	"fmt"
	"strings"
)

type OperationInputKind string

const (
	OperationInputNone        OperationInputKind = "NONE"
	OperationInputOneTarget   OperationInputKind = "ONE_TARGET"
	OperationInputTwoTargets  OperationInputKind = "TWO_TARGETS"
	OperationInputChoice      OperationInputKind = "CHOICE"
	OperationInputPrivateInfo OperationInputKind = "PRIVATE_INFO"
)

type OperationDefinition struct {
	ID                 string             `json:"id"`
	Name               string             `json:"name"`
	InputKind          OperationInputKind `json:"input_kind"`
	TargetCount        int                `json:"target_count,omitempty"`
	MinPlayers         int                `json:"min_players"`
	Category           int                `json:"category,omitempty"`
	MinEventOrder      int                `json:"min_event_order,omitempty"`
	Hidden             bool               `json:"hidden,omitempty"`
	IsPack             bool               `json:"is_pack,omitempty"`
	Enabled            bool               `json:"enabled"`
	RecoveredOnly      bool               `json:"recovered_only,omitempty"`
	PublicInstruction  string             `json:"public_instruction"`
	PrivateInstruction string             `json:"private_instruction"`
}

type OperationResolver interface {
	Definition() OperationDefinition
	Begin(state *GameState) error
	Resolve(state *GameState, command Command) error
}

type anonymousTipResolver struct{}

func (anonymousTipResolver) Definition() OperationDefinition {
	return OperationDefinition{
		ID:                 "OneRandom",
		Name:               "Anonymous Tip",
		InputKind:          OperationInputPrivateInfo,
		MinPlayers:         5,
		MinEventOrder:      1,
		Enabled:            true,
		PublicInstruction:  "The active player has received a private source message.",
		PrivateInstruction: "Your source reveals the agency of one other agent.",
	}
}

func (anonymousTipResolver) Begin(state *GameState) error {
	activeID := activePlayerID(*state)
	targets := otherPlayerIDs(*state, activeID)
	if len(targets) == 0 {
		return ErrInvalidTarget
	}
	targetID := targets[nextRandom(state, len(targets))]
	state.ActivePlayerID = activeID
	state.Operation = newOperationState(state, anonymousTipResolver{}.Definition())
	state.Operation.TargetPlayerID = targetID
	state.Operation.PrivateResults = map[string]OperationResult{activeID: {
		Code:           "FACTION_REVEALED",
		TargetPlayerID: targetID,
		TargetFaction:  checkFaction(state.Players[targetID]),
		Message:        "Your source reveals the agency of one other agent.",
	}}
	return nil
}

func (anonymousTipResolver) Resolve(_ *GameState, _ Command) error {
	return ErrNotAllowed
}

type spyTransferResolver struct{}

func (spyTransferResolver) Definition() OperationDefinition {
	return OperationDefinition{
		ID:                 "Swap",
		Name:               "Spy Transfer",
		InputKind:          OperationInputOneTarget,
		TargetCount:        1,
		MinPlayers:         5,
		MinEventOrder:      1,
		IsPack:             true,
		Enabled:            true,
		PublicInstruction:  "The active player is choosing an exchange target.",
		PrivateInstruction: "Choose one other agent. You and that agent secretly exchange agencies.",
	}
}

func (spyTransferResolver) Begin(state *GameState) error {
	state.ActivePlayerID = activePlayerID(*state)
	state.Operation = newOperationState(state, spyTransferResolver{}.Definition())
	return nil
}

func (spyTransferResolver) Resolve(state *GameState, command Command) error {
	targetIDs := operationTargetIDs(command)
	if len(targetIDs) != 1 {
		return ErrInvalidTarget
	}
	targetID := targetIDs[0]
	activeID := state.ActivePlayerID
	if targetID == activeID {
		return ErrInvalidTarget
	}
	target, ok := state.Players[targetID]
	if !ok {
		return ErrInvalidTarget
	}
	active := state.Players[activeID]
	activeFaction, targetFaction := active.Faction, target.Faction
	// A Loyalist keeps their agency while the other side of the exchange still
	// takes theirs, so a cancelled half looks exactly like a completed one from
	// the outside.
	setFaction(state, active.ID, targetFaction)
	setFaction(state, target.ID, activeFaction)
	active, target = state.Players[active.ID], state.Players[target.ID]
	state.Operation.TargetPlayerID = target.ID
	state.Operation.TargetPlayerIDs = []string{target.ID}
	state.Operation.PrivateResults = map[string]OperationResult{
		active.ID: {
			Code:            "FACTIONS_EXCHANGED",
			TargetPlayerID:  target.ID,
			TargetPlayerIDs: []string{target.ID},
			TargetFaction:   target.Faction,
			OtherPlayerID:   target.ID,
			OtherFaction:    target.Faction,
			YourFaction:     active.Faction,
			Message:         "You exchanged agencies with the selected agent.",
		},
		target.ID: {
			Code:            "FACTIONS_EXCHANGED",
			TargetPlayerID:  active.ID,
			TargetPlayerIDs: []string{active.ID},
			TargetFaction:   active.Faction,
			OtherPlayerID:   active.ID,
			OtherFaction:    active.Faction,
			YourFaction:     target.Faction,
			Message:         "Your agency was exchanged with the active agent.",
		},
	}
	return nil
}

type detectorResolver struct{}

func (detectorResolver) Definition() OperationDefinition {
	return OperationDefinition{
		ID:                 "Detector",
		Name:               "Secret Intel",
		InputKind:          OperationInputTwoTargets,
		TargetCount:        2,
		MinPlayers:         4,
		MinEventOrder:      1,
		Enabled:            true,
		PublicInstruction:  "The active player is reviewing a two-agent intelligence check.",
		PrivateInstruction: "Choose two other agents. You learn whether either one is VIRUS.",
	}
}

func (detectorResolver) Begin(state *GameState) error {
	state.ActivePlayerID = activePlayerID(*state)
	state.Operation = newOperationState(state, detectorResolver{}.Definition())
	return nil
}

func (detectorResolver) Resolve(state *GameState, command Command) error {
	targetIDs := operationTargetIDs(command)
	if len(targetIDs) != 2 || targetIDs[0] == targetIDs[1] {
		return ErrInvalidTarget
	}
	activeID := state.ActivePlayerID
	for _, targetID := range targetIDs {
		if targetID == activeID {
			return ErrInvalidTarget
		}
		if _, ok := state.Players[targetID]; !ok {
			return ErrInvalidTarget
		}
	}
	hasVirus := false
	for _, targetID := range targetIDs {
		if checkFaction(state.Players[targetID]) == FactionVirus {
			hasVirus = true
			break
		}
	}
	code := "NO_VIRUS_FOUND"
	message := "Secret Intel found no VIRUS agent among the two selected players."
	if hasVirus {
		code = "AT_LEAST_ONE_VIRUS"
		message = "Secret Intel found at least one VIRUS agent among the two selected players."
	}
	state.Operation.TargetPlayerIDs = append([]string(nil), targetIDs...)
	state.Operation.PrivateResults = map[string]OperationResult{activeID: {
		Code:            code,
		TargetPlayerIDs: append([]string(nil), targetIDs...),
		Message:         message,
	}}
	return nil
}

type shareResolver struct{}

func (shareResolver) Definition() OperationDefinition {
	return OperationDefinition{
		ID:                 "Share",
		Name:               "Confession",
		InputKind:          OperationInputOneTarget,
		TargetCount:        1,
		MinPlayers:         5,
		MinEventOrder:      1,
		Enabled:            true,
		PublicInstruction:  "The active player is sharing one private agency fact.",
		PrivateInstruction: "Choose one other agent who may view your agency information.",
	}
}

func (shareResolver) Begin(state *GameState) error {
	state.ActivePlayerID = activePlayerID(*state)
	state.Operation = newOperationState(state, shareResolver{}.Definition())
	return nil
}

func (shareResolver) Resolve(state *GameState, command Command) error {
	targetIDs := operationTargetIDs(command)
	if len(targetIDs) != 1 || targetIDs[0] == state.ActivePlayerID {
		return ErrInvalidTarget
	}
	targetID := targetIDs[0]
	if _, ok := state.Players[targetID]; !ok {
		return ErrInvalidTarget
	}
	active := state.Players[state.ActivePlayerID]
	activeFaction := checkFaction(active)
	state.Operation.TargetPlayerID = targetID
	state.Operation.TargetPlayerIDs = []string{targetID}
	state.Operation.PrivateResults = map[string]OperationResult{
		targetID: {
			Code:           "AGENCY_SHARED",
			TargetPlayerID: targetID,
			OtherPlayerID:  active.ID,
			OtherFaction:   activeFaction,
			Message:        fmt.Sprintf("%s is %s.", active.Name, activeFaction),
		},
	}
	return nil
}

var operationResolvers = append([]OperationResolver{
	anonymousTipResolver{},
	spyTransferResolver{},
	detectorResolver{},
	shareResolver{},
}, explicitOperationResolvers()...)

func defaultEnabledOperations() map[string]bool {
	return map[string]bool{
		"Grudge":           true,
		"Infatuation":      true,
		"Share":            true,
		"Detector":         true,
		"Strain":           true,
		"Flip":             true,
		"HiddenOneRandom":  true,
		"TwoFriends":       true,
		"OneOfTwo":         true,
		"OneRandom":        true,
		"Swap":             false,
		"Undercover":       false,
		"InfoForTwo":       false,
		"ChooseVoteShield": false,
		"Defect":           false,
	}
}

var liveOperationIDs = map[string]bool{
	"Grudge":           true,
	"Infatuation":      true,
	"Share":            true,
	"Detector":         true,
	"Strain":           true,
	"Flip":             true,
	"HiddenOneRandom":  true,
	"TwoFriends":       true,
	"OneOfTwo":         true,
	"OneRandom":        true,
	"Swap":             true,
	"Undercover":       true,
	"InfoForTwo":       true,
	"ChooseVoteShield": true,
	"Defect":           true,
}

func IsLiveOperation(kind string) bool {
	return liveOperationIDs[normalizeOperationKind(kind)]
}

func CanonicalOperationKind(kind string) string {
	return normalizeOperationKind(kind)
}

func initOperationQueue(state *GameState) {
	queue := append([]string(nil), state.PlayerOrder...)
	for i := len(queue) - 1; i > 0; i-- {
		j := nextRandom(state, i+1)
		queue[i], queue[j] = queue[j], queue[i]
	}
	state.OperationQueue = queue
	state.OperationQueueIndex = 0
}

// nextOperationRecipient cycles through the shuffled recipient order. The
// operation deck may outlive one player pass, so recipients are deliberately
// independent from operation uniqueness.
func nextOperationRecipient(state *GameState) string {
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

func beginPlannedOperation(state *GameState) error {
	resolver, err := operationResolverFor(state.PlannedOperation)
	if err != nil {
		return err
	}
	if err := resolver.Begin(state); err != nil {
		return err
	}
	state.OperationsDealt = append(state.OperationsDealt, state.ActivePlayerID)
	state.OperationDeals++
	if resolver.Definition().InputKind == OperationInputNone || resolver.Definition().InputKind == OperationInputPrivateInfo {
		state.Phase = PhaseOperationResult
	} else {
		state.Phase = PhaseOperationInput
	}
	return nil
}

func operationForStart(state *GameState, requested string) (OperationResolver, error) {
	initOperationQueue(state)
	if err := initOperationDeck(state); err != nil {
		return nil, err
	}
	state.OperationDealTarget = len(state.OperationDeck)
	if len(state.PlayerOrder) > state.OperationDealTarget {
		state.OperationDealTarget = len(state.PlayerOrder)
	}
	state.OperationDeals = 0
	firstRecipient := nextOperationRecipient(state)
	if firstRecipient == "" {
		return nil, ErrNotEnoughPlayers
	}
	state.ActivePlayerID = firstRecipient
	return takeOperationFromDeck(state, firstRecipient, requested, 1)
}

// operationDeckSlots creates one card per named operation and one card for the
// entire Hidden Agenda envelope group. The enabled pool is fixed for a match;
// only the order changes when a cycle is shuffled.
func operationDeckSlots(state GameState) []string {
	slots := make([]string, 0, len(operationResolvers))
	hidden := false
	for _, resolver := range operationResolvers {
		definition := resolver.Definition()
		if !dealableOperation(&state, definition) {
			continue
		}
		if definition.Hidden {
			hidden = true
			continue
		}
		slots = append(slots, definition.ID)
	}
	if hidden {
		slots = append(slots, HiddenAgendaKind)
	}
	return slots
}

func shuffleOperationDeck(state *GameState, slots []string) {
	for i := len(slots) - 1; i > 0; i-- {
		j := nextRandom(state, i+1)
		slots[i], slots[j] = slots[j], slots[i]
	}
	// A fresh cycle is still random, but avoid an immediate boundary duplicate
	// when another card is available.
	if len(slots) > 1 && state.OperationLastKind != "" && slots[0] == state.OperationLastKind {
		slots[0], slots[1] = slots[1], slots[0]
	}
	state.OperationDeck = slots
}

func initOperationDeck(state *GameState) error {
	state.OperationDeck = nil
	return refillOperationDeck(state)
}

func refillOperationDeck(state *GameState) error {
	slots := operationDeckSlots(*state)
	if len(slots) == 0 {
		return ErrNoEligibleOperations
	}
	shuffleOperationDeck(state, slots)
	return nil
}

func removeOperationDeckSlot(state *GameState, index int) string {
	kind := state.OperationDeck[index]
	state.OperationDeck = append(state.OperationDeck[:index], state.OperationDeck[index+1:]...)
	state.OperationLastKind = kind
	return kind
}

func playerHasCategory(player PlayerState, category int) bool {
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

func hiddenAgendaResolversFor(state *GameState, recipientID string, eventOrder int) []OperationResolver {
	player := state.Players[recipientID]
	members := make([]OperationResolver, 0)
	for _, resolver := range operationResolvers {
		definition := resolver.Definition()
		if !definition.Hidden || definition.MinEventOrder > eventOrder || !dealableOperation(state, definition) || playerHasCategory(player, definition.Category) {
			continue
		}
		members = append(members, resolver)
	}
	return members
}

func resolverForOperationSlot(state *GameState, slot, recipientID string, eventOrder int) OperationResolver {
	if slot == HiddenAgendaKind {
		return drawOperation(state, hiddenAgendaResolversFor(state, recipientID, eventOrder))
	}
	resolver, err := operationResolverFor(slot)
	if err != nil {
		return nil
	}
	definition := resolver.Definition()
	if !dealableOperation(state, definition) || definition.MinEventOrder > eventOrder {
		return nil
	}
	return resolver
}

func takeOperationFromDeck(state *GameState, recipientID, requested string, eventOrder int) (OperationResolver, error) {
	requested = normalizeOperationKind(requested)
	if requested != "" {
		if requested == HiddenAgendaKind || IsHiddenAgendaMember(requested) {
			member := ""
			if requested != HiddenAgendaKind {
				member = requested
			}
			for index, slot := range state.OperationDeck {
				if slot != HiddenAgendaKind {
					continue
				}
				var resolver OperationResolver
				if member == "" {
					resolver = drawOperation(state, hiddenAgendaResolversFor(state, recipientID, eventOrder))
				} else if candidate, err := operationResolverFor(member); err == nil {
					definition := candidate.Definition()
					player := state.Players[recipientID]
					if definition.MinEventOrder <= eventOrder && dealableOperation(state, definition) && !playerHasCategory(player, definition.Category) {
						resolver = candidate
					}
				}
				if resolver == nil {
					return nil, ErrNotAllowed
				}
				removeOperationDeckSlot(state, index)
				recordDealtOperation(state, recipientID, resolver.Definition())
				return resolver, nil
			}
			return nil, ErrNotAllowed
		}
		resolver, err := operationResolverFor(requested)
		if err != nil {
			return nil, err
		}
		definition := resolver.Definition()
		if !dealableOperation(state, definition) || definition.MinEventOrder > eventOrder {
			return nil, ErrNotAllowed
		}
		for index, slot := range state.OperationDeck {
			if slot != definition.ID {
				continue
			}
			removeOperationDeckSlot(state, index)
			recordDealtOperation(state, recipientID, definition)
			return resolver, nil
		}
		return nil, ErrNotAllowed
	}

	if len(state.OperationDeck) == 0 {
		if err := refillOperationDeck(state); err != nil {
			return nil, err
		}
	}
	for index, slot := range state.OperationDeck {
		resolver := resolverForOperationSlot(state, slot, recipientID, eventOrder)
		if resolver == nil {
			continue
		}
		removeOperationDeckSlot(state, index)
		recordDealtOperation(state, recipientID, resolver.Definition())
		return resolver, nil
	}
	return nil, ErrNoEligibleOperations
}

// hiddenAgendaResolvers is the set of envelopes the cover can currently resolve
// to: every hidden member the host left enabled and the table can support.
func hiddenAgendaResolvers(state *GameState) []OperationResolver {
	var members []OperationResolver
	for _, resolver := range operationResolvers {
		if def := resolver.Definition(); def.Hidden && dealableOperation(state, def) {
			members = append(members, resolver)
		}
	}
	return members
}

// HiddenAgendaMemberIDs lists every operation the Hidden Agenda cover can
// resolve to, whether or not the host currently has it enabled.
func HiddenAgendaMemberIDs() []string {
	var ids []string
	for _, resolver := range operationResolvers {
		if def := resolver.Definition(); def.Hidden && !def.RecoveredOnly && IsLiveOperation(def.ID) {
			ids = append(ids, def.ID)
		}
	}
	return ids
}

// IsHiddenAgendaMember reports whether an operation is dealt behind the cover
// rather than under its own name.
func IsHiddenAgendaMember(kind string) bool {
	def, ok := OperationDefinitionFor(kind)
	return ok && def.Hidden
}

// hiddenAgendaEnabled treats the group as one switch: the cover is in the pool
// as long as at least one envelope can still turn up inside it.
func hiddenAgendaEnabled(pool map[string]bool) bool {
	for _, id := range HiddenAgendaMemberIDs() {
		if pool[id] {
			return true
		}
	}
	return false
}

func countEnabledHiddenAgenda(pool map[string]bool) int {
	count := 0
	for _, id := range HiddenAgendaMemberIDs() {
		if pool[id] {
			count++
		}
	}
	return count
}

func cloneOperationPool(pool map[string]bool) map[string]bool {
	clone := make(map[string]bool, len(pool))
	for operationID, enabled := range pool {
		clone[operationID] = enabled
	}
	return clone
}

func countEnabledOperations(pool map[string]bool) int {
	count := 0
	for _, enabled := range pool {
		if enabled {
			count++
		}
	}
	return count
}

// dealableOperation is the floor every draw sits on: the operation exists as a
// live one, the host left it in the pool and the table is big enough for it.
func dealableOperation(state *GameState, def OperationDefinition) bool {
	return !def.RecoveredOnly && IsLiveOperation(def.ID) && state.Settings.EnabledOperations[def.ID] && len(state.PlayerOrder) >= def.MinPlayers
}

// drawOperation picks one operation, counting the whole Hidden Agenda group as
// a single candidate. Hidden Agenda is one operation that can resolve to any of
// its envelopes, so it takes one slot in the draw and which envelope arrived is
// only decided after that slot wins. Giving every member its own slot would
// make the cover several times likelier than any named operation.
func drawOperation(state *GameState, eligible []OperationResolver) OperationResolver {
	named := make([]OperationResolver, 0, len(eligible))
	hidden := make([]OperationResolver, 0, len(eligible))
	for _, resolver := range eligible {
		if resolver.Definition().Hidden {
			hidden = append(hidden, resolver)
		} else {
			named = append(named, resolver)
		}
	}
	slots := len(named)
	if len(hidden) > 0 {
		slots++
	}
	if slots == 0 {
		return nil
	}
	if slot := nextRandom(state, slots); slot < len(named) {
		return named[slot]
	}
	return hidden[nextRandom(state, len(hidden))]
}

// recordDealtOperation remembers what a player was handed. Hidden Agenda members
// share a category, so recording the category is what stops one player from
// drawing the cover twice even though the members have different IDs.
func recordDealtOperation(state *GameState, recipientID string, def OperationDefinition) {
	player := state.Players[recipientID]
	player.DealtOperations = append(player.DealtOperations, def.ID)
	if def.Category > 0 {
		player.DealtCategories = append(player.DealtCategories, def.Category)
	}
	state.Players[recipientID] = player
}

func randomEligibleOperation(state *GameState, recipientID string, eventOrder int) (OperationResolver, error) {
	if len(state.OperationDeck) == 0 {
		if err := initOperationDeck(state); err != nil {
			return nil, err
		}
	}
	resolver, err := takeOperationFromDeck(state, recipientID, "", eventOrder)
	if err == ErrNoEligibleOperations && len(state.OperationDeck) > 0 {
		// This compatibility helper is used by the per-player envelope test. The
		// real match scheduler assigns the blocked card to another recipient;
		// this isolated helper may instead start a fresh draw for the same player.
		if refillErr := initOperationDeck(state); refillErr != nil {
			return nil, err
		}
		return takeOperationFromDeck(state, recipientID, "", eventOrder)
	}
	return resolver, err
}

func operationResolverFor(kind string) (OperationResolver, error) {
	normalized := normalizeOperationKind(kind)
	for _, resolver := range operationResolvers {
		if resolver.Definition().ID == normalized {
			return resolver, nil
		}
	}
	return nil, ErrUnknownOperation
}

func OperationDefinitionFor(kind string) (OperationDefinition, bool) {
	resolver, err := operationResolverFor(kind)
	if err != nil {
		return OperationDefinition{}, false
	}
	return resolver.Definition(), true
}

func OperationDefinitions() []OperationDefinition {
	definitions := make([]OperationDefinition, 0, len(operationResolvers))
	for _, resolver := range operationResolvers {
		definitions = append(definitions, resolver.Definition())
	}
	return definitions
}

func normalizeOperationKind(kind string) string {
	kind = strings.TrimSpace(kind)
	if kind == "SpyTransfer" {
		return "Swap"
	}
	return kind
}

func newOperationState(state *GameState, definition OperationDefinition) *OperationState {
	return &OperationState{
		ID:             fmt.Sprintf("op_%d", state.Version+1),
		Kind:           definition.ID,
		Name:           definition.Name,
		InputKind:      definition.InputKind,
		ActivePlayerID: state.ActivePlayerID,
		InputOwnerID:   state.ActivePlayerID,
		Step:           1,
	}
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

func otherPlayerIDs(state GameState, playerID string) []string {
	players := make([]string, 0, len(state.PlayerOrder)-1)
	for _, id := range state.PlayerOrder {
		if id != playerID {
			players = append(players, id)
		}
	}
	return players
}

func chooseRandomOther(state *GameState, playerID string) string {
	players := otherPlayerIDs(*state, playerID)
	if len(players) == 0 {
		return ""
	}
	return players[nextRandom(state, len(players))]
}

func chooseRandomOthers(state *GameState, playerID string, count int) []string {
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

func chooseRandomSameInitialAgencyPair(state *GameState, playerID string) []string {
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

func checkFaction(player PlayerState) Faction {
	if player.ApparentFaction != nil {
		return *player.ApparentFaction
	}
	return player.Faction
}

// checkInitialFaction is what an operation digging into a player's past finds.
// A lying role rewrites the record as well as the present, so Old Photographs
// cannot be used to see through a Deep Cover or Suspicious Agent.
func checkInitialFaction(player PlayerState) Faction {
	if player.ApparentFaction != nil {
		return *player.ApparentFaction
	}
	return player.InitialFaction
}

func activePlayerID(state GameState) string {
	// The caller nominates whose turn it is; resolvers only fall back to seat
	// order when nobody has been nominated yet.
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
	if len(state.PlayerOrder) == 0 {
		return ""
	}
	return state.PlayerOrder[0]
}
