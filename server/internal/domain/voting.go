package domain

func allVotesSubmitted(state GameState) bool {
	connected := 0
	for _, id := range state.PlayerOrder {
		player := state.Players[id]
		if player.Connected && player.CanVote {
			connected++
			if _, ok := state.Vote.Submitted[id]; !ok {
				return false
			}
		}
	}
	return connected > 0
}

func resolveVote(state *GameState) {
	for _, voterID := range state.PlayerOrder {
		targetID, ok := state.Vote.Submitted[voterID]
		if !ok {
			continue
		}
		target := state.Players[targetID]
		// A shield protects one submitted ballot. It is consumed before that
		// ballot's voting power is applied, so a power-2 ballot is fully blocked.
		if consumeStatus(&target, "VOTE_SHIELD") {
			state.Players[targetID] = target
			continue
		}
		power := state.Players[voterID].VotingPower
		if power == 0 {
			power = 1
		}
		state.Vote.Totals[targetID] += power
	}
	for _, targetID := range state.PlayerOrder {
		player := state.Players[targetID]
		for _, status := range player.Statuses {
			if status == "EXTRA_SUSPICION" {
				state.Vote.Totals[targetID]++
			}
		}
	}
	highest := 0
	winner := string("")
	tie := false
	for _, targetID := range state.PlayerOrder {
		total := state.Vote.Totals[targetID]
		switch {
		case total > highest:
			highest, winner, tie = total, targetID, false
		case total == highest && total > 0:
			tie = true
		}
	}
	if !tie && winner != "" {
		state.Vote.ImprisonedPlayerID = winner
		imprisoned := state.Players[winner]
		if imprisoned.ObjectiveKind == "IMPRISON_SELF" {
			state.Winner = FactionNone
			return
		}
		if imprisoned.Faction == FactionVirus {
			state.Winner = FactionService
		} else {
			state.Winner = FactionVirus
		}
		return
	}
	// A tie vote imprisons nobody. VIRUS wins the tie unless no VIRUS agents remain.
	state.Vote.ImprisonedPlayerID = ""
	virusCount := countCurrentVirus(*state)
	if virusCount > 0 {
		state.Winner = FactionVirus
	} else {
		state.Winner = FactionService
	}
}

func countCurrentVirus(state GameState) int {
	count := 0
	for _, player := range state.Players {
		if player.Faction == FactionVirus {
			count++
		}
	}
	return count
}

func hasStatus(player PlayerState, status string) bool {
	for _, candidate := range player.Statuses {
		if candidate == status {
			return true
		}
	}
	return false
}

func consumeStatus(player *PlayerState, status string) bool {
	for index, candidate := range player.Statuses {
		if candidate == status {
			player.Statuses = append(player.Statuses[:index], player.Statuses[index+1:]...)
			return true
		}
	}
	return false
}
