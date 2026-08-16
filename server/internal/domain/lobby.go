package domain

import "time"

// ApplyJoin seats a new player in an unstarted lobby. Re-applying the same
// player is not an error: an idempotent join retry has to land on the seat it
// already won rather than being told the room is occupied by itself.
func ApplyJoin(state GameState, playerID string, name string, now time.Time) (Transition, error) {
	if _, exists := state.Players[playerID]; exists {
		return Transition{State: state}, nil
	}
	next := clonePlayers(state)
	if err := next.AddPlayer(playerID, name); err != nil {
		return Transition{}, err
	}
	return commit(next, "PLAYER_JOINED", next.Players[playerID].Name+" joined the lobby", now), nil
}

// clonePlayers copies the roster so a rejected transition cannot leave the
// caller's state half-mutated.
func clonePlayers(state GameState) GameState {
	players := make(map[string]PlayerState, len(state.Players)+1)
	for id, player := range state.Players {
		players[id] = player
	}
	state.Players = players
	state.PlayerOrder = append([]string(nil), state.PlayerOrder...)
	return state
}

func resetForRematch(state *GameState) {
	for _, id := range state.PlayerOrder {
		player := state.Players[id]
		player.Ready = false
		player.InitialFaction = ""
		player.Faction = ""
		player.ApparentFaction = nil
		player.Role = ""
		player.CanVote = true
		player.VotingPower = 1
		player.Statuses = nil
		player.ObjectiveKind = ""
		player.ObjectiveTarget = ""
		player.DealtOperations = nil
		player.DealtCategories = nil
		state.Players[id] = player
	}
	state.Phase = PhaseLobby
	state.RoleAcks = nil
	state.DiscussionAcks = nil
	state.ActivePlayerID = ""
	state.PlannedOperation = ""
	state.OperationQueue = nil
	state.OperationsDealt = nil
	state.Operation = nil
	state.DiscussionDeadline = nil
	state.Vote = VoteState{Submitted: map[string]string{}, Totals: map[string]int{}}
	state.Winner = FactionNone
}

func standardVirusCount(playerCount int) int {
	if playerCount <= 6 {
		return 2
	}
	return 3
}

func assignRoles(state *GameState) {
	remaining := append([]string(nil), state.PlayerOrder...)
	redCount := state.Settings.VirusCount
	if redCount <= 0 {
		redCount = standardVirusCount(len(state.PlayerOrder))
	}
	if redCount >= len(state.PlayerOrder) {
		redCount = len(state.PlayerOrder) - 1
	}
	if redCount < 1 {
		redCount = 1
	}
	for i := 0; i < redCount && len(remaining) > 0; i++ {
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
	// The pool is the switch: a room with no roles in it plays without them.
	if anySpecialRoleEnabled(state.Settings.EnabledRoles) {
		dealSpecialRoles(state)
	}
}

func allRoleAcks(state GameState) bool {
	connected := 0
	for _, id := range state.PlayerOrder {
		if !state.Players[id].Connected {
			continue
		}
		connected++
		if !state.RoleAcks[id] {
			return false
		}
	}
	return connected > 0
}

func allDiscussionAcks(state GameState) bool {
	connected := 0
	for _, id := range state.PlayerOrder {
		if !state.Players[id].Connected {
			continue
		}
		connected++
		if !state.DiscussionAcks[id] {
			return false
		}
	}
	return connected > 0
}

func nextRandom(state *GameState, max int) int {
	if state.RandomState == 0 {
		state.RandomState = 88172645463393265
	}
	state.RandomState ^= state.RandomState << 7
	state.RandomState ^= state.RandomState >> 9
	return int(state.RandomState % uint64(max))
}
