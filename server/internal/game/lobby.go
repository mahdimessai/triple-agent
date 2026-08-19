package game

import (
	"strings"
	"time"
)

func NewLobby(hostID, hostName string, settings Settings) State {
	if settings.MinPlayers == 0 {
		settings = DefaultSettings()
	}
	if settings.EnabledOperations == nil {
		settings.EnabledOperations = defaultEnabledOperations()
	}
	if settings.EnabledRoles == nil {
		settings.EnabledRoles = defaultEnabledRoles()
	}
	hostName = strings.TrimSpace(hostName)
	return State{
		HostID:      hostID,
		Settings:    settings,
		Phase:       PhaseLobby,
		Players:     map[string]Player{hostID: {ID: hostID, Name: hostName, Connected: true, CanVote: true, VotingPower: 1}},
		PlayerOrder: []string{hostID},
		Vote:        VoteState{Submitted: map[string]string{}, Totals: map[string]int{}},
		RandomState: uint64(time.Now().UnixNano()),
	}
}

func AddPlayer(state State, playerID, name string) (State, error) {
	if state.Phase != PhaseLobby {
		return state, ErrNotAllowed
	}
	if len(state.PlayerOrder) >= state.Settings.MaxPlayers {
		return state, ErrRoomFull
	}
	if _, exists := state.Players[playerID]; exists {
		return state, ErrPlayerExists
	}
	next := cloneState(state)
	next.Players[playerID] = Player{ID: playerID, Name: strings.TrimSpace(name), CanVote: true, VotingPower: 1}
	next.PlayerOrder = append(next.PlayerOrder, playerID)
	return committed(next), nil
}

func Connect(state State, playerID string) (State, error) {
	player, ok := state.Players[playerID]
	if !ok {
		return state, ErrPlayerNotInRoom
	}
	if player.Connected {
		return state, nil
	}
	next := cloneState(state)
	player = next.Players[playerID]
	player.Connected = true
	next.Players[playerID] = player
	return committed(next), nil
}

func Leave(state State, playerID string) (State, error) {
	if state.Phase != PhaseLobby {
		return state, ErrNotAllowed
	}
	if _, ok := state.Players[playerID]; !ok {
		return state, ErrPlayerNotInRoom
	}
	next := cloneState(state)
	removePlayer(&next, playerID)
	return committed(next), nil
}

func Disconnect(state State, playerID string, now time.Time) (State, error) {
	player, ok := state.Players[playerID]
	if !ok {
		return state, ErrPlayerNotInRoom
	}
	if state.Phase == PhaseLobby {
		next := cloneState(state)
		player = next.Players[playerID]
		player.Connected = false
		next.Players[playerID] = player
		if next.HostID == playerID {
			transferHost(&next)
		}
		return committed(next), nil
	}
	if !player.Connected && state.HostID != playerID {
		return state, nil
	}
	next := cloneState(state)
	player = next.Players[playerID]
	player.Connected = false
	next.Players[playerID] = player
	if next.HostID == playerID {
		transferHost(&next)
	}
	if next.Operation != nil && next.ActivePlayerID == playerID && (next.Phase == PhaseOperationInput || next.Phase == PhaseOperationResult) {
		next.Operation.PrivateResults = nil
		next.Phase = PhaseDiscussion
		next.ActivePlayerID = ""
		next.DiscussionAcks = make(map[string]bool, len(next.PlayerOrder))
		next.DiscussionDeadline = nil
		if next.Settings.DiscussionTimerEnabled {
			deadline := now.Add(time.Duration(next.Settings.DiscussionSeconds) * time.Second)
			next.DiscussionDeadline = &deadline
		}
		return committed(next), nil
	}
	if next.Phase == PhaseRoleReveal && allRoleAcks(next) {
		if err := beginPlannedOperation(&next); err != nil {
			return state, err
		}
	}
	if next.Phase == PhaseVoteInput && allVotesSubmitted(next) {
		resolveVote(&next)
		next.Phase = PhaseResultsIntro
	}
	return committed(next), nil
}

func Empty(state State) bool { return len(state.PlayerOrder) == 0 }

func HasPlayer(state State, playerID string) bool {
	_, ok := state.Players[playerID]
	return ok
}

func removePlayer(state *State, playerID string) {
	delete(state.Players, playerID)
	remaining := state.PlayerOrder[:0]
	for _, id := range state.PlayerOrder {
		if id != playerID {
			remaining = append(remaining, id)
		}
	}
	state.PlayerOrder = remaining
	delete(state.RoleAcks, playerID)
	delete(state.DiscussionAcks, playerID)
	delete(state.Vote.Submitted, playerID)
	if state.HostID == playerID {
		state.HostID = replacementHost(*state)
	}
}

func transferHost(state *State) {
	for _, id := range state.PlayerOrder {
		if state.Players[id].Connected {
			state.HostID = id
			return
		}
	}
}

func replacementHost(state State) string {
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

func resetForRematch(state *State) {
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
	state.OperationQueueIndex = 0
	state.OperationDeck = nil
	state.OperationLastKind = ""
	state.OperationDealTarget = 0
	state.OperationDeals = 0
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

func allRoleAcks(state State) bool {
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

func allDiscussionAcks(state State) bool {
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

func nextRandom(state *State, max int) int {
	if max <= 0 {
		return 0
	}
	if state.RandomState == 0 {
		state.RandomState = 88172645463393265
	}
	state.RandomState ^= state.RandomState << 7
	state.RandomState ^= state.RandomState >> 9
	return int(state.RandomState % uint64(max))
}
