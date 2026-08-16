package domain

import (
	"fmt"
	"time"
)

// Apply validates and applies one client command to a room state.
func Apply(state GameState, command Command, now time.Time) (Transition, error) {
	if command.ExpectedVersion != state.Version {
		return Transition{}, ErrStaleVersion
	}
	player, exists := state.Players[command.ActorID]
	if !exists {
		return Transition{}, fmt.Errorf("player %s is not in room", command.ActorID)
	}

	switch command.Kind {
	case CommandRematch:
		if (!isResultsPhase(state.Phase) && state.Phase != PhaseEnd) || command.ActorID != state.HostID {
			return Transition{}, ErrNotAllowed
		}
		resetForRematch(&state)
		return commit(state, "REMATCH_STARTED", "The room is ready for a new match", now), nil

	case CommandSetOperationEnabled:
		if state.Phase != PhaseLobby || command.ActorID != state.HostID {
			return Transition{}, ErrNotAllowed
		}
		// Hidden Agenda is one operation in the pool, so the host switches the
		// whole cover on or off with a single command; the members underneath it
		// are still individually toggleable for hosts who want a narrower deck.
		if CanonicalOperationKind(command.OperationKind) == HiddenAgendaKind {
			pool := state.Settings.EnabledOperations
			if hiddenAgendaEnabled(pool) == command.OperationEnabled {
				return Transition{State: state}, nil
			}
			if !command.OperationEnabled && countEnabledOperations(pool)-countEnabledHiddenAgenda(pool) < 1 {
				return Transition{}, ErrNotAllowed
			}
			pool = cloneOperationPool(pool)
			for _, id := range HiddenAgendaMemberIDs() {
				pool[id] = command.OperationEnabled
			}
			state.Settings.EnabledOperations = pool
			status := "enabled"
			if !command.OperationEnabled {
				status = "disabled"
			}
			return commit(state, "OPERATION_"+status, player.Name+" "+status+" "+HiddenAgendaName, now), nil
		}
		resolver, resolverErr := operationResolverFor(command.OperationKind)
		if resolverErr != nil || resolver.Definition().RecoveredOnly || !IsLiveOperation(resolver.Definition().ID) {
			return Transition{}, ErrNotAllowed
		}
		if state.Settings.EnabledOperations[resolver.Definition().ID] == command.OperationEnabled {
			return Transition{State: state}, nil
		}
		if !command.OperationEnabled && countEnabledOperations(state.Settings.EnabledOperations) <= 1 {
			return Transition{}, ErrNotAllowed
		}
		state.Settings.EnabledOperations = cloneOperationPool(state.Settings.EnabledOperations)
		state.Settings.EnabledOperations[resolver.Definition().ID] = command.OperationEnabled
		status := "enabled"
		if !command.OperationEnabled {
			status = "disabled"
		}
		return commit(state, "OPERATION_"+status, player.Name+" "+status+" "+resolver.Definition().Name, now), nil

	case CommandSetRoleEnabled:
		if state.Phase != PhaseLobby || command.ActorID != state.HostID {
			return Transition{}, ErrNotAllowed
		}
		definition, ok := RoleDefinitionFor(command.RoleID)
		if !ok || !definition.Special {
			return Transition{}, ErrNotAllowed
		}
		if state.Settings.EnabledRoles[definition.ID] == command.RoleEnabled {
			return Transition{State: state}, nil
		}
		state.Settings.EnabledRoles = cloneRolePool(state.Settings.EnabledRoles)
		state.Settings.EnabledRoles[definition.ID] = command.RoleEnabled
		status := "enabled"
		if !command.RoleEnabled {
			status = "disabled"
		}
		return commit(state, "ROLE_"+status, player.Name+" "+status+" "+definition.Name, now), nil

	case CommandSetDiscussionTimer:
		if state.Phase != PhaseLobby || command.ActorID != state.HostID {
			return Transition{}, ErrNotAllowed
		}
		newEnabled := command.DiscussionTimerEnabled
		newSeconds := command.DiscussionSeconds
		if newSeconds <= 0 {
			newSeconds = state.Settings.DiscussionSeconds
		}
		if newSeconds < 60 {
			newSeconds = 60
		}
		if newSeconds > 900 {
			newSeconds = 900
		}
		if state.Settings.DiscussionTimerEnabled == newEnabled && state.Settings.DiscussionSeconds == newSeconds {
			return Transition{State: state}, nil
		}
		state.Settings.DiscussionTimerEnabled = newEnabled
		state.Settings.DiscussionSeconds = newSeconds
		status := "disabled"
		if newEnabled {
			status = "enabled"
		}
		return commit(state, "DISCUSSION_TIMER_"+status, player.Name+" configured the discussion timer", now), nil

	case CommandSetVirusCount:
		if state.Phase != PhaseLobby || command.ActorID != state.HostID {
			return Transition{}, ErrNotAllowed
		}
		if command.VirusCount < 0 || command.VirusCount > 4 {
			return Transition{}, ErrNotAllowed
		}
		if state.Settings.VirusCount == command.VirusCount {
			return Transition{State: state}, nil
		}
		state.Settings.VirusCount = command.VirusCount
		msg := fmt.Sprintf("%s set VIRUS count to %d", player.Name, command.VirusCount)
		if command.VirusCount == 0 {
			msg = player.Name + " set VIRUS count to Auto Standard"
		}
		return commit(state, "VIRUS_COUNT_CHANGED", msg, now), nil

	case CommandSetReady:
		if state.Phase != PhaseLobby {
			return Transition{}, ErrNotAllowed
		}
		// Ready is a toggle: a player who changes their mind in the lobby can
		// stand back down instead of forcing the host to start without them.
		player.Ready = !player.Ready
		player.Connected = true
		state.Players[command.ActorID] = player
		if !player.Ready {
			return commit(state, "PLAYER_NOT_READY", player.Name+" is no longer ready", now), nil
		}
		return commit(state, "PLAYER_READY", player.Name+" is ready", now), nil

	case CommandStartMatch:
		if state.Phase != PhaseLobby || command.ActorID != state.HostID {
			return Transition{}, ErrNotAllowed
		}
		if len(state.PlayerOrder) < state.Settings.MinPlayers || len(state.PlayerOrder) > state.Settings.MaxPlayers {
			return Transition{}, ErrNotEnoughPlayers
		}
		for _, id := range state.PlayerOrder {
			player := state.Players[id]
			if !player.Connected || !player.Ready {
				return Transition{}, ErrNotReady
			}
		}
		assignRoles(&state)
		resolver, resolverErr := operationForStart(&state, command.OperationKind)
		if resolverErr != nil {
			return Transition{}, resolverErr
		}
		state.Phase = PhaseRoleReveal
		state.PlannedOperation = resolver.Definition().ID
		state.RoleAcks = make(map[string]bool, len(state.PlayerOrder))
		return commit(state, "MATCH_STARTED", "Role assignment is in progress", now), nil

	case CommandAcknowledgeRole:
		if state.Phase != PhaseRoleReveal {
			return Transition{}, ErrNotAllowed
		}
		if state.RoleAcks[command.ActorID] {
			return Transition{State: state}, nil
		}
		state.RoleAcks[command.ActorID] = true
		if allRoleAcks(state) {
			if err := beginPlannedOperation(&state); err != nil {
				return Transition{}, err
			}
		}
		return commit(state, "ROLE_ACKNOWLEDGED", player.Name+" acknowledged their role", now), nil

	case CommandResolveOperation, CommandSelectOperationTarget:
		if state.Phase != PhaseOperationInput || state.Operation == nil {
			return Transition{}, ErrNotAllowed
		}
		inputOwner := state.Operation.InputOwnerID
		if inputOwner == "" {
			inputOwner = state.ActivePlayerID
		}
		if command.ActorID != inputOwner {
			return Transition{}, ErrNotAllowed
		}
		resolver, resolverErr := operationResolverFor(state.Operation.Kind)
		if resolverErr != nil {
			return Transition{}, resolverErr
		}
		if err := resolver.Resolve(&state, command); err != nil {
			return Transition{}, err
		}
		if state.Operation.Step == 2 && len(state.Operation.PrivateResults) == 0 {
			return commit(state, "OPERATION_TARGET_SELECTED", player.Name+" selected a target for "+publicOperationName(state), now), nil
		}
		state.Phase = PhaseOperationResult
		return commit(state, "OPERATION_RESOLVED", player.Name+" completed "+publicOperationName(state), now), nil

	case CommandOperationExplainDone:
		if state.Phase != PhaseOperationResult {
			return Transition{}, ErrNotAllowed
		}
		allowedActor := state.ActivePlayerID
		if state.Operation != nil && state.Operation.InputOwnerID != "" {
			allowedActor = state.Operation.InputOwnerID
		}
		if command.ActorID != allowedActor && command.ActorID != state.ActivePlayerID {
			return Transition{}, ErrNotAllowed
		}
		// The device goes back on the table for a short, timed beat before the
		// next operation is dealt.
		state.Phase = PhaseOperationInterlude
		deadline := now.Add(time.Duration(interludeSeconds(state.Settings)) * time.Second)
		state.DiscussionDeadline = &deadline
		return commit(state, "OPERATION_EXPLAINED", player.Name+" finished their operation", now), nil

	case CommandAdvanceInterlude:
		if state.Phase != PhaseOperationInterlude || command.ActorID != state.HostID {
			return Transition{}, ErrNotAllowed
		}
		state.DiscussionDeadline = nil
		if next := nextOperationRecipient(state); next != "" {
			eventOrder := len(state.OperationsDealt) + 1
			resolver, resolverErr := randomEligibleOperation(&state, next, eventOrder)
			if resolverErr != nil {
				return Transition{}, resolverErr
			}
			state.ActivePlayerID = next
			state.PlannedOperation = resolver.Definition().ID
			if err := beginPlannedOperation(&state); err != nil {
				return Transition{}, err
			}
			return commit(state, "OPERATION_DEALT", state.Players[next].Name+" received an operation", now), nil
		}
		// Everyone has had a turn, so the room gets its full discussion before
		// the accusation.
		state.Phase = PhaseDiscussion
		state.ActivePlayerID = ""
		state.DiscussionAcks = make(map[string]bool, len(state.PlayerOrder))
		if state.Settings.DiscussionTimerEnabled {
			discussionDeadline := now.Add(time.Duration(state.Settings.DiscussionSeconds) * time.Second)
			state.DiscussionDeadline = &discussionDeadline
		}
		return commit(state, "DISCUSSION_STARTED", "The final discussion is open", now), nil

	case CommandAdvanceDiscussion:
		if state.Phase != PhaseDiscussion {
			return Transition{}, ErrNotAllowed
		}
		if state.DiscussionAcks == nil {
			state.DiscussionAcks = make(map[string]bool, len(state.PlayerOrder))
		}
		state.DiscussionAcks[command.ActorID] = true
		timerExpired := state.DiscussionDeadline != nil && !now.Before(*state.DiscussionDeadline)
		if allDiscussionAcks(state) || timerExpired {
			state.DiscussionDeadline = nil
			state.Phase = PhaseVoteInput
			state.Vote = VoteState{Submitted: map[string]string{}, Totals: map[string]int{}}
			return commit(state, "VOTE_STARTED", "Voting has started", now), nil
		}
		return commit(state, "DISCUSSION_ACKNOWLEDGED", player.Name+" is ready to vote", now), nil

	case CommandSubmitVote:
		if state.Phase != PhaseVoteInput || !player.CanVote {
			return Transition{}, ErrNotAllowed
		}
		if _, submitted := state.Vote.Submitted[command.ActorID]; submitted {
			return Transition{}, ErrAlreadySubmitted
		}
		target, ok := state.Players[command.TargetID]
		if !ok || target.ID == command.ActorID {
			return Transition{}, ErrInvalidTarget
		}
		state.Vote.Submitted[command.ActorID] = command.TargetID
		if allVotesSubmitted(state) {
			resolveVote(&state)
			state.Phase = PhaseResultsIntro
		}
		return commit(state, "VOTE_SUBMITTED", player.Name+" submitted an accusation", now), nil

	case CommandContinueResults:
		if !isResultsPhase(state.Phase) || command.ActorID != state.HostID {
			return Transition{}, ErrNotAllowed
		}
		switch state.Phase {
		case PhaseResultsIntro:
			state.Phase = PhaseVoteResults
			return commit(state, "VOTE_TOTALS_REVEALED", "The vote totals are revealed", now), nil
		case PhaseVoteResults:
			state.Phase = PhaseImprisonment
			return commit(state, "IMPRISONMENT_REVEALED", "The imprisoned player is revealed", now), nil
		case PhaseImprisonment:
			state.Phase = PhaseAgencyReveal
			return commit(state, "AGENCY_REVEALED", "The imprisoned player's agency is revealed", now), nil
		case PhaseAgencyReveal:
			state.Phase = PhaseOutcomeReveal
			return commit(state, "OUTCOME_REVEAL_STARTED", "The winning agency is ready to be revealed", now), nil
		case PhaseOutcomeReveal:
			state.Phase = PhaseLeaderboard
			return commit(state, "WINNER_REVEALED", winnerActivity(state), now), nil
		case PhaseLeaderboard:
			state.Phase = PhaseOutOfLoop
			return commit(state, "LEADERBOARD_REVEALED", "The winners and losers are revealed", now), nil
		case PhaseOutOfLoop:
			state.Phase = PhaseEnd
			return commit(state, "MATCH_ENDED", "The match has ended", now), nil
		}
	}

	return Transition{}, fmt.Errorf("unknown command: %s", command.Kind)
}

func commit(state GameState, kind, publicText string, now time.Time) Transition {
	state.Version++
	event := EventRecord{ID: fmt.Sprintf("evt_%d", state.Version), Kind: kind, PublicText: publicText, At: now}
	return Transition{State: state, Event: event, Changed: true}
}
