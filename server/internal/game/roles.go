package game

type roleDefinition struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Faction     Faction `json:"faction"`
	Special     bool    `json:"special"`
	Score       float64 `json:"score"`
	Description string  `json:"description"`
	Effect      string  `json:"effect"`
}

var roleDefinitions = []roleDefinition{
	{ID: string(RoleNormalBlue), Name: "Service Agent", Faction: FactionService, Description: "Find and imprison the double agents. Be warned, there may be more than one.", Effect: "No special ability."},
	{ID: string(RoleNormalRed), Name: "VIRUS Agent", Faction: FactionVirus, Description: "Keep your cover and get The Service to imprison one of their own.", Effect: "No special ability."},
	{ID: string(RoleFakeBlue), Name: "Rogue Agent", Faction: FactionVirus, Special: true, Score: 0.5, Description: "You work for VIRUS, but the other VIRUS agents were never told about you.", Effect: "You are a Rogue Agent. The other VIRUS Agents do not know that you are a double agent"},
	{ID: string(RoleFakeRed), Name: "Triple Agent", Faction: FactionService, Special: true, Score: 0.5, Description: "The VIRUS agents think you are one of them. You are not.", Effect: "You are a triple agent. The VIRUS double agents think you are on their side, but you are actually working for the Service."},
	{ID: string(RoleLyingRed), Name: "Deep Cover Agent", Faction: FactionVirus, Special: true, Score: -0.5, Description: "You are operating under deep cover for VIRUS.", Effect: "You are operating under deep cover. Anytime someone tries to check your status they will see you as a Service agent."},
	{ID: string(RoleLyingBlue), Name: "Suspicious Agent", Faction: FactionService, Special: true, Score: -0.5, Description: "Your past includes some ties to suspicious figures.", Effect: "Your past includes some ties to suspicious figures. Any time someone tries to check your status, they will see you as a VIRUS agent"},
	{ID: string(RoleLoyalBlue), Name: "Service Loyalist", Faction: FactionService, Special: true, Score: 0.5, Description: "You are a die-hard loyalist and will not be turned.", Effect: "You are a die hard loyalist. Any operation that attempts to change your team from a Service agent will be cancelled."},
	{ID: string(RoleLoyalRed), Name: "VIRUS Loyalist", Faction: FactionVirus, Special: true, Score: -0.5, Description: "You are a die-hard loyalist and will not be turned.", Effect: "You are a die hard loyalist. Any operation that attempts to change your team from a VIRUS agent will be cancelled."},
}

func roleDefinitionFor(id string) (roleDefinition, bool) {
	for _, definition := range roleDefinitions {
		if definition.ID == id {
			return definition, true
		}
	}
	return roleDefinition{}, false
}

func defaultEnabledRoles() map[string]bool { return map[string]bool{} }

func anySpecialRoleEnabled(pool map[string]bool) bool {
	for _, definition := range roleDefinitions {
		if definition.Special && pool[definition.ID] {
			return true
		}
	}
	return false
}

func assignRoles(state *State) {
	remaining := append([]string(nil), state.PlayerOrder...)
	virusCount := state.Settings.VirusCount
	if virusCount <= 0 {
		virusCount = standardVirusCount(len(state.PlayerOrder))
	}
	if virusCount >= len(state.PlayerOrder) {
		virusCount = len(state.PlayerOrder) - 1
	}
	if virusCount < 1 {
		virusCount = 1
	}
	for i := 0; i < virusCount && len(remaining) > 0; i++ {
		index := nextRandom(state, len(remaining))
		id := remaining[index]
		remaining = append(remaining[:index], remaining[index+1:]...)
		player := state.Players[id]
		player.InitialFaction = FactionVirus
		player.Faction = FactionVirus
		player.Role = RoleNormalRed
		state.Players[id] = player
	}
	for _, id := range remaining {
		player := state.Players[id]
		player.InitialFaction = FactionService
		player.Faction = FactionService
		player.Role = RoleNormalBlue
		state.Players[id] = player
	}
	if anySpecialRoleEnabled(state.Settings.EnabledRoles) {
		dealSpecialRoles(state)
	}
}

func dealSpecialRoles(state *State) {
	pool := make([]roleDefinition, 0, len(roleDefinitions))
	for _, definition := range roleDefinitions {
		if definition.Special && state.Settings.EnabledRoles[definition.ID] {
			pool = append(pool, definition)
		}
	}
	if len(pool) == 0 {
		return
	}
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
			if definition.Faction == player.Faction && absScore(balance+definition.Score) <= 0.5 {
				candidates = append(candidates, index)
			}
		}
		if len(candidates) == 0 {
			continue
		}
		chosen := candidates[nextRandom(state, len(candidates))]
		definition := pool[chosen]
		pool = append(pool[:chosen], pool[chosen+1:]...)
		balance += definition.Score
		player.Role = RoleKind(definition.ID)
		switch player.Role {
		case RoleLyingRed:
			faction := FactionService
			player.ApparentFaction = &faction
		case RoleLyingBlue:
			faction := FactionVirus
			player.ApparentFaction = &faction
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

func isLoyalist(player Player) bool {
	return player.Role == RoleLoyalBlue || player.Role == RoleLoyalRed
}

func setFaction(state *State, playerID string, faction Faction) bool {
	player, ok := state.Players[playerID]
	if !ok || (isLoyalist(player) && player.Faction != faction) {
		return false
	}
	player.Faction = faction
	state.Players[playerID] = player
	return true
}

func virusRoster(state State) []string {
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

func seesVirusRoster(player Player) bool {
	return player.InitialFaction == FactionVirus || player.Role == RoleFakeRed
}

func trueVirusCount(state State) int {
	count := 0
	for _, id := range state.PlayerOrder {
		if state.Players[id].InitialFaction == FactionVirus {
			count++
		}
	}
	return count
}
