package domain

import (
	"fmt"
	"time"
)

// ApplyLeave removes a player from an unstarted lobby. The player loses their
// seat and reconnect credential, so they must join again to return.
func ApplyLeave(state GameState, playerID string, now time.Time) (Transition, error) {
	if state.Phase != PhaseLobby {
		return Transition{}, ErrNotAllowed
	}
	player, ok := state.Players[playerID]
	if !ok {
		return Transition{}, fmt.Errorf("player %s is not in room", playerID)
	}
	state = removeSeat(state, playerID)
	return commit(state, "PLAYER_LEFT", player.Name+" left the lobby", now), nil
}

// removeSeat drops a player from the roster and closes the gap in the seating
// order. Seats are never held open: whoever is gone is gone, and the remaining
// players renumber from one.
func removeSeat(state GameState, playerID string) GameState {
	players := make(map[string]PlayerState, len(state.Players))
	for id, current := range state.Players {
		players[id] = current
	}
	state.Players = players
	state.PlayerOrder = append([]string(nil), state.PlayerOrder...)
	delete(state.Players, playerID)
	remaining := make([]string, 0, len(state.PlayerOrder))
	for _, id := range state.PlayerOrder {
		if id != playerID {
			remaining = append(remaining, id)
		}
	}
	state.PlayerOrder = remaining
	for seat, id := range state.PlayerOrder {
		current := state.Players[id]
		current.Seat = seat + 1
		state.Players[id] = current
	}
	delete(state.RoleAcks, playerID)
	delete(state.Vote.Submitted, playerID)
	if state.HostID == playerID {
		state.HostID = replacementHost(state)
	}
	// The seat is gone for good, so the room is handed over rather than waiting
	// for an owner who is no longer at the table.
	if state.OwnerID == playerID {
		state.OwnerID = state.HostID
	}
	return state
}

// ApplyDisconnect applies the server's absence policy. A disconnected player
// keeps their seat and reconnect credential once a match has started, so a
// reload or short network loss can restore them to the current game. Lobby
// disconnects are different: the seat is released immediately, and returning
// players must join again as a new seat. In-game absences no longer block role
// acknowledgements or voting; if they own an unfinished private step, that
// step is skipped without exposing a private result.
func ApplyDisconnect(state GameState, playerID string, now time.Time) (Transition, error) {
	player, ok := state.Players[playerID]
	if !ok {
		return Transition{}, fmt.Errorf("player %s is not in room", playerID)
	}
	if state.Phase == PhaseLobby {
		state = removeSeat(state, playerID)
		return commit(state, "PLAYER_LEFT", player.Name+" left the lobby", now), nil
	}
	if !player.Connected && state.HostID != playerID {
		return Transition{State: state}, nil
	}
	player.Connected = false
	state.Players[playerID] = player
	hostTransferred := false
	if state.HostID == playerID {
		transferHost(&state)
		hostTransferred = state.HostID != playerID
	}
	if state.Operation != nil && state.Operation.ActivePlayerID == playerID && (state.Phase == PhaseOperationInput || state.Phase == PhaseOperationResult) {
		state.Operation.PrivateResults = nil
		state.Phase = PhaseDiscussion
		state.ActivePlayerID = ""
		state.DiscussionDeadline = nil
		if state.Settings.DiscussionTimerEnabled {
			deadline := now.Add(time.Duration(state.Settings.DiscussionSeconds) * time.Second)
			state.DiscussionDeadline = &deadline
		}
		return commit(state, "OPERATION_SKIPPED_DISCONNECTED", "The active player's unfinished operation was skipped", now), nil
	}
	policyChanged := hostTransferred
	if state.Phase == PhaseRoleReveal && allRoleAcks(state) {
		if err := beginPlannedOperation(&state); err != nil {
			return Transition{}, err
		}
		policyChanged = true
	}
	if state.Phase == PhaseVoteInput && allVotesSubmitted(state) {
		resolveVote(&state)
		state.Phase = PhaseResultsIntro
		policyChanged = true
	}
	if !policyChanged {
		// Presence is part of the public projection, so it needs a new version.
		// Otherwise clients that already received the prior version can discard
		// this update as stale and never learn that a player went offline.
		return commit(state, "PLAYER_DISCONNECTED", player.Name+" disconnected", now), nil
	}
	return commit(state, "PLAYER_DISCONNECTED", player.Name+" disconnected", now), nil
}

func transferHost(state *GameState) {
	for _, id := range state.PlayerOrder {
		if state.Players[id].Connected {
			state.HostID = id
			return
		}
	}
}

func replacementHost(state GameState) string {
	for _, id := range state.PlayerOrder {
		if state.Players[id].Connected {
			return id
		}
	}
	if len(state.PlayerOrder) > 0 {
		return state.PlayerOrder[0]
	}
	return ""
}
