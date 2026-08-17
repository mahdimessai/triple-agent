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
		won := playerWins(state, id)
		result := "LOSER"
		if won {
			result = "WINNER"
		}
		targetName := ""
		if player.ObjectiveTarget != "" {
			if target, ok := state.Players[player.ObjectiveTarget]; ok {
				targetName = target.Name
			}
		}
		desc, reason := explainObjectiveAndResult(state, player, won, targetName)
		entries = append(entries, LeaderboardEntry{
			PlayerID:             id,
			Name:                 player.Name,
			Faction:              player.Faction,
			Role:                 player.Role,
			Defection:            defectionFor(player),
			Votes:                state.Vote.Totals[id],
			Result:               result,
			ObjectiveKind:        player.ObjectiveKind,
			ObjectiveTargetID:    player.ObjectiveTarget,
			ObjectiveTargetName:  targetName,
			ObjectiveDescription: desc,
			WinReason:            reason,
		})
	}
	return entries
}

func explainObjectiveAndResult(state GameState, player PlayerState, won bool, targetName string) (string, string) {
	switch player.ObjectiveKind {
	case "IMPRISON_SELF":
		desc := "Operation: Scapegoat — win only by being imprisoned"
		if won {
			return desc, "Imprisoned and achieved solo victory"
		}
		return desc, "Was not imprisoned"

	case "IMPRISON_TARGET":
		desc := "Grudge — win if " + targetName + " is imprisoned"
		if targetName == "" {
			desc = "Grudge — win if your assigned target is imprisoned"
		}
		if won {
			return desc, "Succeeded in getting " + targetName + " imprisoned"
		}
		return desc, "Failed: " + targetName + " was not imprisoned"

	case "TARGET_WINS":
		desc := "Infatuation — win if " + targetName + " wins"
		if targetName == "" {
			desc = "Infatuation — win if your assigned target wins"
		}
		if won {
			return desc, "Felt the love: " + targetName + " won the match"
		}
		return desc, "Heartbroken: " + targetName + " lost the match"

	case "RED_DEFECTOR":
		desc := "Red Defector — defected to Service; win if Service wins with no VIRUS votes against you"
		if won {
			return desc, "Defected to Service and avoided all VIRUS votes"
		}
		if state.Winner != FactionService {
			return desc, "Defected to Service, but The Service lost the round"
		}
		return desc, "Defected to Service, but was exposed by a VIRUS vote"

	default:
		defection := defectionFor(player)
		if defection == "BLUE_DEFECTOR" {
			desc := "Blue Defector — defected to VIRUS; win with VIRUS"
			if won {
				return desc, "Defected to VIRUS and won with VIRUS"
			}
			return desc, "Defected to VIRUS, but VIRUS lost the round"
		}

		if state.Vote.ImprisonedPlayerID != "" && state.Players[state.Vote.ImprisonedPlayerID].ObjectiveKind == "IMPRISON_SELF" {
			if player.Faction == FactionVirus {
				return "VIRUS agency victory", "Lost: Operation Scapegoat took the round"
			}
			return "The Service agency victory", "Lost: Operation Scapegoat took the round"
		}

		if player.Faction == FactionVirus {
			desc := "VIRUS agency victory"
			if won {
				return desc, "Won with VIRUS"
			}
			return desc, "Lost with VIRUS"
		} else if player.Faction == FactionService {
			desc := "The Service agency victory"
			if won {
				return desc, "Won with The Service"
			}
			return desc, "Lost with The Service"
		}

		return "Agency victory", "Match concluded"
	}
}

func defectionFor(player PlayerState) string {
	for _, status := range player.Statuses {
		switch status {
		case "BLUE_DEFECTOR", "RED_DEFECTOR":
			return status
		}
	}
	return ""
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
			if player.ObjectiveKind == "IMPRISON_TARGET" && player.ObjectiveTarget == state.Vote.ImprisonedPlayerID {
				return true
			}
			if player.ObjectiveKind == "TARGET_WINS" && player.ObjectiveTarget != "" {
				return playerWinsWithVisited(state, player.ObjectiveTarget, visited)
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
