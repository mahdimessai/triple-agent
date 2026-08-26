package game

import (
	"fmt"
	"time"
)

func Apply(state State, actorID string, command Command, now time.Time) (State, error) {
	player, exists := state.Players[actorID]
	if !exists {
		return state, ErrPlayerNotInRoom
	}

	switch command.Kind {
	case CommandTransferHost:
		if state.Phase != PhaseLobby || actorID != state.HostID {
			return state, ErrNotAllowed
		}
		if command.TargetID == state.HostID {
			return state, nil
		}
		if _, exists := state.Players[command.TargetID]; !exists {
			return state, ErrInvalidTarget
		}
		next := state
		next.HostID = command.TargetID
		return next.committed(), nil

	case CommandKickPlayer:
		if state.Phase != PhaseLobby || actorID != state.HostID || command.TargetID == state.HostID {
			return state, ErrNotAllowed
		}
		if _, exists := state.Players[command.TargetID]; !exists {
			return state, ErrInvalidTarget
		}
		next := copyStateForPlayerRemoval(state)
		next.removePlayer(command.TargetID)
		return next.committed(), nil

	case CommandVoteKick:
		if !player.Connected {
			return state, ErrNotAllowed
		}
		targetPlayer, exists := state.Players[command.TargetID]
		if !exists {
			return state, ErrInvalidTarget
		}
		if command.TargetID == actorID || targetPlayer.Connected {
			return state, ErrNotAllowed
		}
		if state.VoteKicks != nil && state.VoteKicks[command.TargetID] != nil && state.VoteKicks[command.TargetID][actorID] {
			return state, nil
		}
		connectedCount := 0
		votes := 1 // The duplicate-vote check above guarantees this is a new vote.
		for id, p := range state.Players {
			if !p.Connected {
				continue
			}
			connectedCount++
			if state.VoteKicks != nil && state.VoteKicks[command.TargetID] != nil && state.VoteKicks[command.TargetID][id] {
				votes++
			}
		}

		var next State
		if votes >= connectedCount && connectedCount > 0 {
			// A unanimous vote can immediately advance an operation or resolve
			// the ballot, both of which mutate nested state beyond player removal.
			next = cloneState(state)
		} else {
			next = copyVoteKicks(state)
		}
		if next.VoteKicks == nil {
			next.VoteKicks = make(map[string]map[string]bool)
		}
		if next.VoteKicks[command.TargetID] == nil {
			next.VoteKicks[command.TargetID] = make(map[string]bool)
		}
		next.VoteKicks[command.TargetID][actorID] = true

		required := connectedCount
		if votes >= required && required > 0 {
			kickedID := command.TargetID
			wasActive := next.ActivePlayerID == kickedID
			next.removePlayer(kickedID)
			delete(next.VoteKicks, kickedID)

			if (next.Phase == PhaseOperationInput || next.Phase == PhaseOperationResult) && wasActive {
				if err := next.advanceInterlude(now); err != nil {
					next.Phase = PhaseDiscussion
					next.ActivePlayerID = ""
					next.DiscussionAcks = make(map[string]bool, len(next.PlayerOrder))
					if next.Settings.DiscussionTimerEnabled {
						deadline := now.Add(time.Duration(next.Settings.DiscussionSeconds) * time.Second)
						next.DiscussionDeadline = &deadline
					}
				}
			} else if next.Phase == PhaseRoleReveal {
				if allRoleAcks(next) {
					if next.ActivePlayerID == kickedID {
						if nextRecipient := nextOperationRecipient(&next); nextRecipient != "" {
							next.ActivePlayerID = nextRecipient
							if op, err := takeOperationFromDeck(&next, nextRecipient, "", 1); err == nil {
								next.PlannedOperation = op.definition.ID
							}
						}
					}
					if err := next.beginPlannedOperation(); err != nil {
						return state, err
					}
				}
			} else if next.Phase == PhaseDiscussion {
				if allDiscussionAcks(next) {
					next.beginVote()
				}
			} else if next.Phase == PhaseVoteInput {
				if allVotesSubmitted(next) {
					next.resolveVote()
					next.Phase = PhaseEnd
				}
			}
		}
		return next.committed(), nil

	case CommandRematch:
		if (!isResultsPhase(state.Phase) && state.Phase != PhaseEnd) || actorID != state.HostID {
			return state, ErrNotAllowed
		}
		next := copyPlayers(state)
		next.resetForRematch()
		return next.committed(), nil

	case CommandSetOperationEnabled:
		if state.Phase != PhaseLobby || actorID != state.HostID {
			return state, ErrNotAllowed
		}
		if command.OperationKind == hiddenAgendaKind {
			if hiddenAgendaEnabled(state.Settings.EnabledOperations) == command.OperationEnabled {
				return state, nil
			}
			if !command.OperationEnabled && countEnabledOperations(state.Settings.EnabledOperations)-countEnabledHiddenAgenda(state.Settings.EnabledOperations) < 1 {
				return state, ErrNotAllowed
			}
			next := copyEnabledOperations(state)
			for _, id := range hiddenAgendaMemberIDs() {
				next.Settings.EnabledOperations[id] = command.OperationEnabled
			}
			return next.committed(), nil
		}
		if _, ok := operations[command.OperationKind]; !ok {
			return state, ErrNotAllowed
		}
		if state.Settings.EnabledOperations[command.OperationKind] == command.OperationEnabled {
			return state, nil
		}
		if !command.OperationEnabled && countEnabledOperations(state.Settings.EnabledOperations) <= 1 {
			return state, ErrNotAllowed
		}
		next := copyEnabledOperations(state)
		next.Settings.EnabledOperations[command.OperationKind] = command.OperationEnabled
		return next.committed(), nil

	case CommandSetRoleEnabled:
		if state.Phase != PhaseLobby || actorID != state.HostID {
			return state, ErrNotAllowed
		}
		definition, ok := roleDefinitionFor(command.RoleID)
		if !ok || !definition.Special {
			return state, ErrNotAllowed
		}
		if state.Settings.EnabledRoles[definition.ID] == command.RoleEnabled {
			return state, nil
		}
		next := copyEnabledRoles(state)
		next.Settings.EnabledRoles[definition.ID] = command.RoleEnabled
		return next.committed(), nil

	case CommandSetDiscussionTimer:
		if state.Phase != PhaseLobby || actorID != state.HostID {
			return state, ErrNotAllowed
		}
		seconds := command.DiscussionSeconds
		if seconds <= 0 {
			seconds = state.Settings.DiscussionSeconds
		}
		if seconds < 60 {
			seconds = 60
		}
		if seconds > 900 {
			seconds = 900
		}
		if state.Settings.DiscussionTimerEnabled == command.DiscussionTimerEnabled && state.Settings.DiscussionSeconds == seconds {
			return state, nil
		}
		next := state
		next.Settings.DiscussionTimerEnabled = command.DiscussionTimerEnabled
		next.Settings.DiscussionSeconds = seconds
		return next.committed(), nil

	case CommandSetVirusCount:
		if state.Phase != PhaseLobby || actorID != state.HostID || command.VirusCount < 0 || command.VirusCount > 4 {
			return state, ErrNotAllowed
		}
		if state.Settings.VirusCount == command.VirusCount {
			return state, nil
		}
		next := state
		next.Settings.VirusCount = command.VirusCount
		return next.committed(), nil

	case CommandSetReady:
		if state.Phase != PhaseLobby {
			return state, ErrNotAllowed
		}
		next := copyPlayers(state)
		player = next.Players[actorID]
		player.Ready = !player.Ready
		next.Players[actorID] = player
		return next.committed(), nil

	case CommandStartMatch:
		if state.Phase != PhaseLobby || actorID != state.HostID {
			return state, ErrNotAllowed
		}
		if len(state.PlayerOrder) < state.Settings.MinPlayers || len(state.PlayerOrder) > state.Settings.MaxPlayers {
			return state, ErrNotEnoughPlayers
		}
		for _, id := range state.PlayerOrder {
			candidate := state.Players[id]
			if !candidate.Connected || !candidate.Ready {
				return state, ErrNotReady
			}
		}
		next := cloneState(state)
		assignRoles(&next)
		op, err := operationForStart(&next, command.OperationKind)
		if err != nil {
			return state, err
		}
		next.Phase = PhaseRoleReveal
		next.PlannedOperation = op.definition.ID
		next.RoleAcks = make(map[string]bool, len(next.PlayerOrder))
		return next.committed(), nil

	case CommandAcknowledgeRole:
		if state.Phase != PhaseRoleReveal {
			return state, ErrNotAllowed
		}
		if state.RoleAcks[actorID] {
			return state, nil
		}
		next := copyRoleAcks(state)
		if !allRoleAcksAfter(next, actorID) {
			if next.RoleAcks == nil {
				next.RoleAcks = make(map[string]bool, len(next.PlayerOrder))
			}
			next.RoleAcks[actorID] = true
			return next.committed(), nil
		}
		next = cloneState(state)
		next.RoleAcks[actorID] = true
		if allRoleAcks(next) {
			if err := next.beginPlannedOperation(); err != nil {
				return state, err
			}
		}
		return next.committed(), nil

	case CommandResolveOperation, CommandSelectOperationTarget:
		if state.Phase != PhaseOperationInput || state.Operation == nil {
			return state, ErrNotAllowed
		}
		inputOwner := state.Operation.InputOwnerID
		if inputOwner == "" {
			inputOwner = state.ActivePlayerID
		}
		if actorID != inputOwner {
			return state, ErrNotAllowed
		}
		next := cloneState(state)
		if err := resolveCurrentOperation(&next, command); err != nil {
			return state, err
		}
		if next.Operation.Step == 2 && len(next.Operation.PrivateResults) == 0 {
			return next.committed(), nil
		}
		next.Phase = PhaseOperationResult
		return next.committed(), nil

	case CommandOperationExplainDone:
		if state.Phase != PhaseOperationResult || state.Operation == nil {
			return state, ErrNotAllowed
		}
		if !isOperationParticipant(state, actorID) {
			return state, ErrNotAllowed
		}
		if state.Operation.Acks != nil && state.Operation.Acks[actorID] {
			return state, nil
		}
		next := copyOperationAcks(state)
		if !allOperationAcksAfter(next, actorID) {
			if next.Operation.Acks == nil {
				next.Operation.Acks = make(map[string]bool)
			}
			next.Operation.Acks[actorID] = true
			return next.committed(), nil
		}
		next = cloneState(state)
		if next.Operation.Acks == nil {
			next.Operation.Acks = make(map[string]bool)
		}
		next.Operation.Acks[actorID] = true
		if allOperationAcks(next) {
			next.Phase = PhaseOperationInterlude
			deadline := now.Add(time.Duration(interludeSeconds(next.Settings)) * time.Second)
			next.DiscussionDeadline = &deadline
		}
		return next.committed(), nil

	case CommandAdvanceInterlude:
		if state.Phase != PhaseOperationInterlude || actorID != state.HostID {
			return state, ErrNotAllowed
		}
		next := cloneState(state)
		if err := next.advanceInterlude(now); err != nil {
			return state, err
		}
		return next.committed(), nil

	case CommandAdvanceDiscussion:
		if state.Phase != PhaseDiscussion {
			return state, ErrNotAllowed
		}
		next := copyDiscussionAcks(state)
		already := next.DiscussionAcks[actorID]
		timerExpired := next.DiscussionDeadline != nil && !now.Before(*next.DiscussionDeadline)
		allAfter := allDiscussionAcksAfter(next, actorID)
		if !timerExpired && already && !allAfter {
			return state, nil
		}
		if !timerExpired && !allAfter {
			if next.DiscussionAcks == nil {
				next.DiscussionAcks = make(map[string]bool, len(next.PlayerOrder))
			}
			next.DiscussionAcks[actorID] = true
			return next.committed(), nil
		}
		next = cloneState(state)
		if next.DiscussionAcks == nil {
			next.DiscussionAcks = make(map[string]bool, len(next.PlayerOrder))
		}
		already = next.DiscussionAcks[actorID]
		next.DiscussionAcks[actorID] = true
		timerExpired = next.DiscussionDeadline != nil && !now.Before(*next.DiscussionDeadline)
		if allDiscussionAcks(next) || timerExpired {
			next.beginVote()
			return next.committed(), nil
		}
		if already {
			return state, nil
		}
		return next.committed(), nil

	case CommandSubmitVote:
		if state.Phase != PhaseVoteInput || !player.CanVote {
			return state, ErrNotAllowed
		}
		if _, submitted := state.Vote.Submitted[actorID]; submitted {
			return state, ErrAlreadySubmitted
		}
		target, ok := state.Players[command.TargetID]
		if !ok || target.ID == actorID {
			return state, ErrInvalidTarget
		}
		next := copyVoteSubmitted(state)
		next.Vote.Submitted[actorID] = command.TargetID
		if !allVotesSubmitted(next) {
			return next.committed(), nil
		}
		next = cloneState(state)
		next.Vote.Submitted[actorID] = command.TargetID
		if allVotesSubmitted(next) {
			next.resolveVote()
			next.Phase = PhaseEnd
		}
		return next.committed(), nil

	case CommandContinueResults:
		if !isResultsPhase(state.Phase) || actorID != state.HostID {
			return state, ErrNotAllowed
		}
		next := state
		switch state.Phase {
		case PhaseResultsIntro:
			next.Phase = PhaseVoteResults
		case PhaseVoteResults:
			next.Phase = PhaseImprisonment
		case PhaseImprisonment:
			next.Phase = PhaseAgencyReveal
		case PhaseAgencyReveal:
			next.Phase = PhaseOutcomeReveal
		case PhaseOutcomeReveal:
			next.Phase = PhaseLeaderboard
		case PhaseLeaderboard:
			next.Phase = PhaseOutOfLoop
		case PhaseOutOfLoop:
			next.Phase = PhaseEnd
		default:
			return state, ErrNotAllowed
		}
		return next.committed(), nil
	}

	return state, fmt.Errorf("unknown command: %s", command.Kind)
}

func AdvanceDeadline(state State, now time.Time) (State, error) {
	if state.DiscussionDeadline == nil || now.Before(*state.DiscussionDeadline) {
		return state, nil
	}
	next := cloneState(state)
	switch state.Phase {
	case PhaseOperationInterlude:
		if err := next.advanceInterlude(now); err != nil {
			return state, err
		}
	case PhaseDiscussion:
		next.beginVote()
	default:
		return state, nil
	}
	return next.committed(), nil
}

func (state *State) advanceInterlude(now time.Time) error {
	state.DiscussionDeadline = nil
	if state.OperationDeals < state.OperationDealTarget {
		if nextRecipient := nextOperationRecipient(state); nextRecipient != "" {
			eventOrder := state.OperationDeals + 1
			op, err := takeOperationFromDeck(state, nextRecipient, "", eventOrder)
			if err != nil {
				return err
			}
			state.ActivePlayerID = nextRecipient
			state.PlannedOperation = op.definition.ID
			return state.beginPlannedOperation()
		}
	}
	state.Phase = PhaseDiscussion
	state.ActivePlayerID = ""
	state.DiscussionAcks = make(map[string]bool, len(state.PlayerOrder))
	if state.Settings.DiscussionTimerEnabled {
		deadline := now.Add(time.Duration(state.Settings.DiscussionSeconds) * time.Second)
		state.DiscussionDeadline = &deadline
	}
	return nil
}

func (state *State) beginVote() {
	state.DiscussionDeadline = nil
	state.Phase = PhaseVoteInput
	state.Vote = VoteState{Submitted: map[string]string{}, Totals: map[string]int{}}
}

func isOperationParticipant(state State, playerID string) bool {
	if state.Operation == nil {
		return false
	}
	if state.ActivePlayerID == playerID {
		return true
	}
	if state.Operation.InputOwnerID != "" && state.Operation.InputOwnerID == playerID {
		return true
	}
	if state.Operation.PrivateResults != nil {
		if _, ok := state.Operation.PrivateResults[playerID]; ok {
			return true
		}
	}
	return false
}

func allOperationAcks(state State) bool {
	if state.Operation == nil {
		return true
	}
	if state.ActivePlayerID != "" && (state.Operation.Acks == nil || !state.Operation.Acks[state.ActivePlayerID]) {
		return false
	}
	for id := range state.Operation.PrivateResults {
		if state.Operation.Acks == nil || !state.Operation.Acks[id] {
			return false
		}
	}
	return true
}

func allOperationAcksAfter(state State, acknowledgedID string) bool {
	if state.Operation == nil {
		return true
	}
	if state.ActivePlayerID != "" && state.ActivePlayerID != acknowledgedID {
		if state.Operation.Acks == nil || !state.Operation.Acks[state.ActivePlayerID] {
			return false
		}
	}
	for id := range state.Operation.PrivateResults {
		if id != acknowledgedID && (state.Operation.Acks == nil || !state.Operation.Acks[id]) {
			return false
		}
	}
	return true
}
