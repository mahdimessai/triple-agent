package domain

// Special roles are the Deep Cover pack. A role is dealt on top of an agency,
// never instead of one: the holder still wins with the agency they actually
// work for, so a role changes only what other agents can learn about them or do
// to them.
//
// Score is the balance weight from the original role table. A positive score
// helps The Service, a negative score helps VIRUS, and the dealer keeps the
// running total near zero so a table does not end up lopsided.
type RoleDefinition struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Faction     Faction `json:"faction"`
	Special     bool    `json:"special"`
	Score       float64 `json:"score"`
	Description string  `json:"description"`
	Effect      string  `json:"effect"`
}

var roleDefinitions = []RoleDefinition{
	{
		ID:          string(RoleNormalBlue),
		Name:        "Service Agent",
		Faction:     FactionService,
		Description: "Find and imprison the double agents. Be warned, there may be more than one.",
		Effect:      "No special ability.",
	},
	{
		ID:          string(RoleNormalRed),
		Name:        "VIRUS Agent",
		Faction:     FactionVirus,
		Description: "Keep your cover and get The Service to imprison one of their own.",
		Effect:      "No special ability.",
	},
	{
		ID:          string(RoleFakeBlue),
		Name:        "Rogue Agent",
		Faction:     FactionVirus,
		Special:     true,
		Score:       0.5,
		Description: "You work for VIRUS, but the other VIRUS agents were never told about you.",
		Effect:      "You are a Rogue Agent. The other VIRUS Agents do not know that you are a double agent",
	},
	{
		ID:          string(RoleFakeRed),
		Name:        "Triple Agent",
		Faction:     FactionService,
		Special:     true,
		Score:       0.5,
		Description: "The VIRUS agents think you are one of them. You are not.",
		Effect:      "You are a triple agent. The VIRUS double agents think you are on their side, but you are actually working for the Service.",
	},
	{
		ID:          string(RoleLyingRed),
		Name:        "Deep Cover Agent",
		Faction:     FactionVirus,
		Special:     true,
		Score:       -0.5,
		Description: "You are operating under deep cover for VIRUS.",
		Effect:      "You are operating under deep cover. Anytime someone tries to check your status they will see you as a Service agent.",
	},
	{
		ID:          string(RoleLyingBlue),
		Name:        "Suspicious Agent",
		Faction:     FactionService,
		Special:     true,
		Score:       -0.5,
		Description: "Your past includes some ties to suspicious figures.",
		Effect:      "Your past includes some ties to suspicious figures. Any time someone tries to check your status, they will see you as a VIRUS agent",
	},
	{
		ID:          string(RoleLoyalBlue),
		Name:        "Service Loyalist",
		Faction:     FactionService,
		Special:     true,
		Score:       0.5,
		Description: "You are a die-hard loyalist and will not be turned.",
		Effect:      "You are a die hard loyalist. Any operation that attempts to change your team from a Service agent will be cancelled.",
	},
	{
		ID:          string(RoleLoyalRed),
		Name:        "VIRUS Loyalist",
		Faction:     FactionVirus,
		Special:     true,
		Score:       -0.5,
		Description: "You are a die-hard loyalist and will not be turned.",
		Effect:      "You are a die hard loyalist. Any operation that attempts to change your team from a VIRUS agent will be cancelled.",
	},
}

func RoleDefinitions() []RoleDefinition {
	return append([]RoleDefinition(nil), roleDefinitions...)
}

func RoleDefinitionFor(id string) (RoleDefinition, bool) {
	for _, definition := range roleDefinitions {
		if definition.ID == id {
			return definition, true
		}
	}
	return RoleDefinition{}, false
}

// Special roles ship out of the pool. A room plays without them until the host
// puts at least one card in, which is also what "roles are on" means: there is
// no separate switch to disagree with the pool.
func defaultEnabledRoles() map[string]bool {
	return map[string]bool{}
}

func anySpecialRoleEnabled(pool map[string]bool) bool {
	for _, definition := range roleDefinitions {
		if definition.Special && pool[definition.ID] {
			return true
		}
	}
	return false
}

func cloneRolePool(pool map[string]bool) map[string]bool {
	clone := make(map[string]bool, len(pool))
	for id, enabled := range pool {
		clone[id] = enabled
	}
	return clone
}

// baselineRole is the role a player holds when no special role is dealt to them.
func baselineRole(faction Faction) RoleKind {
	if faction == FactionVirus {
		return RoleNormalRed
	}
	return RoleNormalBlue
}

// dealSpecialRoles hands out at most one special role per player, only ever to
// a player already on that role's faction, and only while the running balance
// score stays within half a point of even.
func dealSpecialRoles(state *GameState) {
	pool := make([]RoleDefinition, 0, len(roleDefinitions))
	for _, definition := range roleDefinitions {
		if definition.Special && state.Settings.EnabledRoles[definition.ID] {
			pool = append(pool, definition)
		}
	}
	if len(pool) == 0 {
		return
	}

	// Walk the seats in a random order so the same seat does not collect the
	// same role every match.
	seats := append([]string(nil), state.PlayerOrder...)
	for i := len(seats) - 1; i > 0; i-- {
		j := nextRandom(state, i+1)
		seats[i], seats[j] = seats[j], seats[i]
	}

	balance := 0.0
	for _, id := range seats {
		player := state.Players[id]
		candidates := make([]int, 0, len(pool))
		for index, definition := range pool {
			if definition.Faction != player.Faction {
				continue
			}
			if absScore(balance+definition.Score) > 0.5 {
				continue
			}
			candidates = append(candidates, index)
		}
		if len(candidates) == 0 {
			continue
		}
		chosen := candidates[nextRandom(state, len(candidates))]
		definition := pool[chosen]
		pool = append(pool[:chosen], pool[chosen+1:]...)
		balance += definition.Score
		player.Role = RoleKind(definition.ID)
		// A lying role is nothing more than a fixed answer to every check, so
		// the existing apparent-agency plumbing carries it.
		switch player.Role {
		case RoleLyingRed:
			service := FactionService
			player.ApparentFaction = &service
		case RoleLyingBlue:
			virus := FactionVirus
			player.ApparentFaction = &virus
		}
		state.Players[id] = player
	}
}

func absScore(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

// isLoyalist reports whether an operation is forbidden from moving this player
// off their current agency.
func isLoyalist(player PlayerState) bool {
	return player.Role == RoleLoyalBlue || player.Role == RoleLoyalRed
}

// setFaction is the single place an operation may change an agency. It returns
// false when a Loyalist cancels the change, so callers can report the no-op
// honestly instead of pretending the switch happened.
func setFaction(state *GameState, playerID string, faction Faction) bool {
	player, ok := state.Players[playerID]
	if !ok {
		return false
	}
	if isLoyalist(player) && player.Faction != faction {
		return false
	}
	player.Faction = faction
	state.Players[playerID] = player
	return true
}

// virusRoster is the list of names shown to the VIRUS side at role reveal. The
// Rogue Agent is missing from it and the Triple Agent is planted in it, which
// is the whole point of those two roles: the roster stops matching the count.
func virusRoster(state GameState) []string {
	roster := make([]string, 0, len(state.PlayerOrder))
	for _, id := range state.PlayerOrder {
		player := state.Players[id]
		if player.Role == RoleFakeBlue {
			continue
		}
		if player.Role == RoleFakeRed || player.InitialFaction == FactionVirus {
			roster = append(roster, id)
		}
	}
	return roster
}

// seesVirusRoster covers everyone who believes they are on the VIRUS side,
// including the Rogue Agent who is missing from the list and the Triple Agent
// who does not belong on it.
func seesVirusRoster(player PlayerState) bool {
	return player.InitialFaction == FactionVirus || player.Role == RoleFakeRed
}

// trueVirusCount is the number of agents who really started on VIRUS. It is
// shown next to the roster so a mismatch between the count and the names is
// visible to the side reading it.
func trueVirusCount(state GameState) int {
	count := 0
	for _, id := range state.PlayerOrder {
		if state.Players[id].InitialFaction == FactionVirus {
			count++
		}
	}
	return count
}
