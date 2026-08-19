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
		next := cloneState(state)
		next.HostID = command.TargetID
		return committed(next), nil

	case CommandKickPlayer:
		if state.Phase != PhaseLobby || actorID != state.HostID || command.TargetID == state.HostID {
			return state, ErrNotAllowed
		}
		if _, exists := state.Players[command.TargetID]; !exists {
			return state, ErrInvalidTarget
		}
		next := cloneState(state)
		removePlayer(&next, command.TargetID)
		return committed(next), nil

	case CommandRematch:
		if (!isResultsPhase(state.Phase) && state.Phase != PhaseEnd) || actorID != state.HostID {
			return state, ErrNotAllowed
		}
		next := cloneState(state)
		resetForRematch(&next)
		return committed(next), nil

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
			next := cloneState(state)
			for _, id := range hiddenAgendaMemberIDs() {
				next.Settings.EnabledOperations[id] = command.OperationEnabled
			}
			return committed(next), nil
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
		next := cloneState(state)
		next.Settings.EnabledOperations[command.OperationKind] = command.OperationEnabled
		return committed(next), nil

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
		next := cloneState(state)
		next.Settings.EnabledRoles[definition.ID] = command.RoleEnabled
		return committed(next), nil

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
		next := cloneState(state)
		next.Settings.DiscussionTimerEnabled = command.DiscussionTimerEnabled
		next.Settings.DiscussionSeconds = seconds
		return committed(next), nil

	case CommandSetVirusCount:
		if state.Phase != PhaseLobby || actorID != state.HostID || command.VirusCount < 0 || command.VirusCount > 4 {
			return state, ErrNotAllowed
		}
		if state.Settings.VirusCount == command.VirusCount {
			return state, nil
		}
		next := cloneState(state)
		next.Settings.VirusCount = command.VirusCount
		return committed(next), nil

	case CommandSetReady:
		if state.Phase != PhaseLobby {
			return state, ErrNotAllowed
		}
		next := cloneState(state)
		player = next.Players[actorID]
		player.Ready = !player.Ready
		next.Players[actorID] = player
		return committed(next), nil

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
		return committed(next), nil

	case CommandAcknowledgeRole:
		if state.Phase != PhaseRoleReveal {
			return state, ErrNotAllowed
		}
		if state.RoleAcks[actorID] {
			return state, nil
		}
		next := cloneState(state)
		next.RoleAcks[actorID] = true
		if allRoleAcks(next) {
			if err := beginPlannedOperation(&next); err != nil {
				return state, err
			}
		}
		return committed(next), nil

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
			return committed(next), nil
		}
		next.Phase = PhaseOperationResult
		return committed(next), nil

	case CommandOperationExplainDone:
		if state.Phase != PhaseOperationResult {
			return state, ErrNotAllowed
		}
		allowedActor := state.ActivePlayerID
		if state.Operation != nil && state.Operation.InputOwnerID != "" {
			allowedActor = state.Operation.InputOwnerID
		}
		if actorID != allowedActor && actorID != state.ActivePlayerID {
			return state, ErrNotAllowed
		}
		next := cloneState(state)
		next.Phase = PhaseOperationInterlude
		deadline := now.Add(time.Duration(interludeSeconds(next.Settings)) * time.Second)
		next.DiscussionDeadline = &deadline
		return committed(next), nil

	case CommandAdvanceInterlude:
		if state.Phase != PhaseOperationInterlude || actorID != state.HostID {
			return state, ErrNotAllowed
		}
		next := cloneState(state)
		if err := advanceInterlude(&next, now); err != nil {
			return state, err
		}
		return committed(next), nil

	case CommandAdvanceDiscussion:
		if state.Phase != PhaseDiscussion {
			return state, ErrNotAllowed
		}
		next := cloneState(state)
		if next.DiscussionAcks == nil {
			next.DiscussionAcks = make(map[string]bool, len(next.PlayerOrder))
		}
		already := next.DiscussionAcks[actorID]
		next.DiscussionAcks[actorID] = true
		timerExpired := next.DiscussionDeadline != nil && !now.Before(*next.DiscussionDeadline)
		if allDiscussionAcks(next) || timerExpired {
			beginVote(&next)
			return committed(next), nil
		}
		if already {
			return state, nil
		}
		return committed(next), nil

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
		next := cloneState(state)
		next.Vote.Submitted[actorID] = command.TargetID
		if allVotesSubmitted(next) {
			resolveVote(&next)
			next.Phase = PhaseResultsIntro
		}
		return committed(next), nil

	case CommandContinueResults:
		if !isResultsPhase(state.Phase) || actorID != state.HostID {
			return state, ErrNotAllowed
		}
		next := cloneState(state)
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
		return committed(next), nil
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
		if err := advanceInterlude(&next, now); err != nil {
			return state, err
		}
	case PhaseDiscussion:
		beginVote(&next)
	default:
		return state, nil
	}
	return committed(next), nil
}

func advanceInterlude(state *State, now time.Time) error {
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
			return beginPlannedOperation(state)
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

func beginVote(state *State) {
	state.DiscussionDeadline = nil
	state.Phase = PhaseVoteInput
	state.Vote = VoteState{Submitted: map[string]string{}, Totals: map[string]int{}}
}
