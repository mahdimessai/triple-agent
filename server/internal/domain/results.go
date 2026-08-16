package domain

func isResultsPhase(phase Phase) bool {
	switch phase {
	case PhaseResultsIntro, PhaseVoteResults, PhaseImprisonment, PhaseAgencyReveal, PhaseOutcomeReveal, PhaseLeaderboard, PhaseOutOfLoop:
		return true
	default:
		return false
	}
}

func revealsVoteTotals(phase Phase) bool {
	return isResultsPhase(phase) || phase == PhaseEnd
}

func revealsImprisonment(phase Phase) bool {
	return isResultsPhase(phase) || phase == PhaseEnd
}

func revealsAgency(phase Phase) bool {
	return isResultsPhase(phase) || phase == PhaseEnd
}

func revealsWinner(phase Phase) bool {
	return isResultsPhase(phase) || phase == PhaseEnd
}

func revealsLeaderboard(phase Phase) bool {
	return isResultsPhase(phase) || phase == PhaseEnd
}

func winnerActivity(state GameState) string {
	if state.Vote.ImprisonedPlayerID != "" {
		imprisoned := state.Players[state.Vote.ImprisonedPlayerID]
		if imprisoned.ObjectiveKind == "IMPRISON_SELF" {
			return "Operation: Scapegoat succeeded; both agencies lose"
		}
	}
	if state.Vote.ImprisonedPlayerID == "" {
		if state.Winner == FactionVirus {
			return "The vote is tied; VIRUS wins the round"
		}
		return "The vote is tied; The Service wins the round"
	}
	return string(state.Winner) + " wins the round"
}

func buildLeaderboard(state GameState) []LeaderboardEntry {
	entries := make([]LeaderboardEntry, 0, len(state.PlayerOrder))
	for _, id := range state.PlayerOrder {
		player := state.Players[id]
		result := "LOSER"
		if playerWins(state, id) {
			result = "WINNER"
		}
		entries = append(entries, LeaderboardEntry{
			PlayerID: id,
			Name:     player.Name,
			Faction:  player.Faction,
			Votes:    state.Vote.Totals[id],
			Result:   result,
		})
	}
	return entries
}

func playerWins(state GameState, playerID string) bool {
	return playerWinsWithVisited(state, playerID, map[string]bool{})
}

func playerWinsWithVisited(state GameState, playerID string, visited map[string]bool) bool {
	if visited[playerID] {
		return false
	}
	visited[playerID] = true
	player, ok := state.Players[playerID]
	if !ok {
		return false
	}

	// Strain / Scapegoat check:
	if state.Vote.ImprisonedPlayerID != "" {
		imprisoned := state.Players[state.Vote.ImprisonedPlayerID]
		if imprisoned.ObjectiveKind == "IMPRISON_SELF" {
			if playerID == state.Vote.ImprisonedPlayerID {
				return true
			}
			if player.ObjectiveKind == "TARGET_WINS" && player.ObjectiveTarget == state.Vote.ImprisonedPlayerID {
				return true
			}
			return false
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
