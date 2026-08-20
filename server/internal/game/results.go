package game

func allVotesSubmitted(state State) bool {
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

func resolveVote(state *State) {
	if state.Vote.Totals == nil {
		state.Vote.Totals = map[string]int{}
	}
	for _, voterID := range state.PlayerOrder {
		targetID, ok := state.Vote.Submitted[voterID]
		if !ok {
			continue
		}
		target := state.Players[targetID]
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
		for _, status := range state.Players[targetID].Statuses {
			if status == "EXTRA_SUSPICION" {
				state.Vote.Totals[targetID]++
			}
		}
	}
	highest := 0
	winner := ""
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
	state.Vote.ImprisonedPlayerID = ""
	if countCurrentVirus(*state) > 0 {
		state.Winner = FactionVirus
	} else {
		state.Winner = FactionService
	}
}

func countCurrentVirus(state State) int {
	count := 0
	for _, player := range state.Players {
		if player.Faction == FactionVirus {
			count++
		}
	}
	return count
}

func hasStatus(player Player, status string) bool {
	for _, candidate := range player.Statuses {
		if candidate == status {
			return true
		}
	}
	return false
}

func consumeStatus(player *Player, status string) bool {
	for index, candidate := range player.Statuses {
		if candidate == status {
			player.Statuses = append(player.Statuses[:index], player.Statuses[index+1:]...)
			return true
		}
	}
	return false
}

func isResultsPhase(phase Phase) bool {
	switch phase {
	case PhaseResultsIntro, PhaseVoteResults, PhaseImprisonment, PhaseAgencyReveal, PhaseOutcomeReveal, PhaseLeaderboard, PhaseOutOfLoop:
		return true
	default:
		return false
	}
}

// Reveal helpers are intentionally cumulative. A projection is the server-side
// secrecy boundary, so data must not be sent before the phase where the game
// actually reveals it. The client UI is not trusted to hide premature data.
func revealsVoteTotals(phase Phase) bool {
	switch phase {
	case PhaseVoteResults, PhaseImprisonment, PhaseAgencyReveal, PhaseOutcomeReveal, PhaseLeaderboard, PhaseOutOfLoop, PhaseEnd:
		return true
	default:
		return false
	}
}

func revealsImprisonment(phase Phase) bool {
	switch phase {
	case PhaseImprisonment, PhaseAgencyReveal, PhaseOutcomeReveal, PhaseLeaderboard, PhaseOutOfLoop, PhaseEnd:
		return true
	default:
		return false
	}
}

func revealsAgency(phase Phase) bool {
	switch phase {
	case PhaseAgencyReveal, PhaseOutcomeReveal, PhaseLeaderboard, PhaseOutOfLoop, PhaseEnd:
		return true
	default:
		return false
	}
}

func revealsWinner(phase Phase) bool {
	switch phase {
	case PhaseOutcomeReveal, PhaseLeaderboard, PhaseOutOfLoop, PhaseEnd:
		return true
	default:
		return false
	}
}

func revealsLeaderboard(phase Phase) bool {
	switch phase {
	case PhaseLeaderboard, PhaseOutOfLoop, PhaseEnd:
		return true
	default:
		return false
	}
}

func winnerActivity(state State) string {
	if state.Vote.ImprisonedPlayerID != "" && state.Players[state.Vote.ImprisonedPlayerID].ObjectiveKind == "IMPRISON_SELF" {
		return "Operation: Scapegoat succeeded; both agencies lose"
	}
	if state.Vote.ImprisonedPlayerID == "" {
		if state.Winner == FactionVirus {
			return "The vote is tied; VIRUS wins the round"
		}
		return "The vote is tied; The Service wins the round"
	}
	return string(state.Winner) + " wins the round"
}

func buildLeaderboard(state State) []LeaderboardEntry {
	entries := make([]LeaderboardEntry, 0, len(state.PlayerOrder))
	for _, id := range state.PlayerOrder {
		player := state.Players[id]
		result := "LOSER"
		if playerWins(state, id) {
			result = "WINNER"
		}
		entries = append(entries, LeaderboardEntry{PlayerID: id, Name: player.Name, Faction: player.Faction, Role: player.Role, Defection: defectionFor(player), Votes: state.Vote.Totals[id], Result: result})
	}
	return entries
}

func defectionFor(player Player) string {
	for _, status := range player.Statuses {
		if status == "BLUE_DEFECTOR" || status == "RED_DEFECTOR" {
			return status
		}
	}
	return ""
}

func playerWins(state State, playerID string) bool {
	return playerWinsWithVisited(state, playerID, map[string]bool{})
}

func playerWinsWithVisited(state State, playerID string, visited map[string]bool) bool {
	if visited[playerID] {
		return false
	}
	visited[playerID] = true
	player, ok := state.Players[playerID]
	if !ok {
		return false
	}
	if state.Vote.ImprisonedPlayerID != "" {
		imprisoned := state.Players[state.Vote.ImprisonedPlayerID]
		if imprisoned.ObjectiveKind == "IMPRISON_SELF" {
			if playerID == state.Vote.ImprisonedPlayerID {
				return true
			}
			return player.ObjectiveKind == "TARGET_WINS" && player.ObjectiveTarget == state.Vote.ImprisonedPlayerID
		}
	}
	switch player.ObjectiveKind {
	case "IMPRISON_SELF":
		return state.Vote.ImprisonedPlayerID == playerID
	case "IMPRISON_TARGET":
		return player.ObjectiveTarget != "" && state.Vote.ImprisonedPlayerID == player.ObjectiveTarget
	case "TARGET_WINS":
		return player.ObjectiveTarget != "" && playerWinsWithVisited(state, player.ObjectiveTarget, visited)
	case "RED_DEFECTOR":
		if state.Winner != FactionService || player.Faction != FactionService {
			return false
		}
		for voterID, targetID := range state.Vote.Submitted {
			if voterID == playerID || targetID != playerID {
				continue
			}
			if voter, ok := state.Players[voterID]; ok && (voter.Faction == FactionVirus || voter.InitialFaction == FactionVirus) {
				return false
			}
		}
		return true
	default:
		return state.Winner != FactionNone && player.Faction == state.Winner
	}
}
