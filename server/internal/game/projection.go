package game

import "time"

type PublicPlayer struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Seat          int    `json:"seat"`
	Ready         bool   `json:"ready"`
	Connected     bool   `json:"connected"`
	VoteSubmitted bool   `json:"vote_submitted"`
}

type PublicOperation struct {
	Kind              string             `json:"kind"`
	Name              string             `json:"name"`
	InputKind         OperationInputKind `json:"input_kind"`
	TargetCount       int                `json:"target_count,omitempty"`
	ActivePlayerID    string             `json:"active_player_id"`
	ActivePlayerName  string             `json:"active_player_name"`
	InputOwnerID      string             `json:"input_owner_id,omitempty"`
	Step              int                `json:"step,omitempty"`
	PublicInstruction string             `json:"public_instruction"`
}

type PublicRoomSettings struct {
	DiscussionTimerEnabled bool     `json:"discussion_timer_enabled"`
	DiscussionSeconds      int      `json:"discussion_seconds"`
	EnabledOperations      []string `json:"enabled_operations"`
	MinPlayers             int      `json:"min_players"`
	MaxPlayers             int      `json:"max_players"`
	InterludeSeconds       int      `json:"interlude_seconds"`
	VirusCount             int      `json:"virus_count"`
	EnabledRoles           []string `json:"enabled_roles"`
}

type LeaderboardEntry struct {
	PlayerID  string   `json:"player_id"`
	Name      string   `json:"name"`
	Faction   Faction  `json:"faction"`
	Role      RoleKind `json:"role,omitempty"`
	Defection string   `json:"defection,omitempty"`
	Votes     int      `json:"votes"`
	Result    string   `json:"result"`
}

type PublicProjection struct {
	RoomID               string             `json:"room_id"`
	HostID               string             `json:"host_id"`
	Phase                Phase              `json:"phase"`
	Version              uint64             `json:"version"`
	Players              []PublicPlayer     `json:"players"`
	Settings             PublicRoomSettings `json:"settings"`
	ActivePlayerID       string             `json:"active_player_id,omitempty"`
	Operation            *PublicOperation   `json:"operation,omitempty"`
	DiscussionDeadline   *time.Time         `json:"discussion_deadline,omitempty"`
	VoteTotals           map[string]int     `json:"vote_totals,omitempty"`
	ImprisonedPlayerID   string             `json:"imprisoned_player_id,omitempty"`
	RevealedFaction      Faction            `json:"revealed_faction,omitempty"`
	Winner               Faction            `json:"winner,omitempty"`
	Leaderboard          []LeaderboardEntry `json:"leaderboard,omitempty"`
	Activity             string             `json:"activity,omitempty"`
	PendingRoleAcks      int                `json:"pending_role_acks,omitempty"`
	DiscussionReadyCount int                `json:"discussion_ready_count,omitempty"`
}

type PrivateProjection struct {
	PlayerID             string           `json:"player_id"`
	Role                 RoleKind         `json:"role,omitempty"`
	InitialFaction       Faction          `json:"initial_faction,omitempty"`
	Faction              Faction          `json:"faction,omitempty"`
	ApparentFaction      *Faction         `json:"apparent_faction,omitempty"`
	OperationResult      *OperationResult `json:"operation_result,omitempty"`
	OperationInstruction string           `json:"operation_instruction,omitempty"`
	RoleName             string           `json:"role_name,omitempty"`
	RoleDescription      string           `json:"role_description,omitempty"`
	RoleEffect           string           `json:"role_effect,omitempty"`
	VirusRoster          []PublicPlayer   `json:"virus_roster,omitempty"`
	VirusTeamSize        int              `json:"virus_team_size,omitempty"`
	OperationKind        string           `json:"operation_kind,omitempty"`
	OperationName        string           `json:"operation_name,omitempty"`
	LegalTargetIDs       []string         `json:"legal_target_ids,omitempty"`
	Choices              []string         `json:"choices,omitempty"`
	VoteSubmitted        bool             `json:"vote_submitted"`
	CanSubmit            bool             `json:"can_submit"`
}

type Projection struct {
	Type    string            `json:"type"`
	Public  PublicProjection  `json:"public"`
	Private PrivateProjection `json:"private"`
}

func Project(roomID string, state State, playerID string) Projection {
	return ProjectWithPublic(state, playerID, PublicProjectionFor(roomID, state))
}

func PublicProjectionFor(roomID string, state State) PublicProjection {
	players := make([]PublicPlayer, 0, len(state.PlayerOrder))
	for index, id := range state.PlayerOrder {
		player := state.Players[id]
		_, submitted := state.Vote.Submitted[id]
		players = append(players, PublicPlayer{ID: id, Name: player.Name, Seat: index + 1, Ready: player.Ready, Connected: player.Connected, VoteSubmitted: submitted})
	}
	voteTotals := map[string]int{}
	if revealsVoteTotals(state.Phase) {
		voteTotals = cloneIntMap(state.Vote.Totals)
	}
	var imprisonedID string
	if revealsImprisonment(state.Phase) {
		imprisonedID = state.Vote.ImprisonedPlayerID
	}
	var revealedFaction Faction
	if revealsAgency(state.Phase) && state.Vote.ImprisonedPlayerID != "" {
		revealedFaction = state.Players[state.Vote.ImprisonedPlayerID].Faction
	}
	var winner Faction
	var leaderboard []LeaderboardEntry
	if revealsWinner(state.Phase) {
		winner = state.Winner
	}
	if revealsLeaderboard(state.Phase) {
		leaderboard = buildLeaderboard(state)
	}
	public := PublicProjection{
		RoomID: roomID, HostID: state.HostID, Phase: state.Phase, Version: state.Version,
		Players: players, Settings: publicRoomSettings(state.Settings), ActivePlayerID: state.ActivePlayerID,
		DiscussionDeadline: state.DiscussionDeadline, VoteTotals: voteTotals,
		ImprisonedPlayerID: imprisonedID, RevealedFaction: revealedFaction, Winner: winner, Leaderboard: leaderboard,
	}
	if state.Phase == PhaseRoleReveal {
		for _, id := range state.PlayerOrder {
			if state.Players[id].Connected && !state.RoleAcks[id] {
				public.PendingRoleAcks++
			}
		}
	}
	if state.Phase == PhaseDiscussion {
		for _, id := range state.PlayerOrder {
			if state.DiscussionAcks[id] {
				public.DiscussionReadyCount++
			}
		}
	}
	if state.Operation != nil {
		if definition, ok := operationDefinitionFor(state.Operation.Kind); ok {
			inputOwner := state.Operation.InputOwnerID
			if inputOwner == "" {
				inputOwner = state.ActivePlayerID
			}
			public.Operation = &PublicOperation{
				Kind: definition.ID, Name: definition.Name, InputKind: definition.InputKind, TargetCount: definition.TargetCount,
				ActivePlayerID: state.ActivePlayerID, InputOwnerID: inputOwner, Step: state.Operation.Step, PublicInstruction: definition.PublicInstruction,
			}
			if active, exists := state.Players[state.ActivePlayerID]; exists {
				public.Operation.ActivePlayerName = active.Name
			}
			if definition.Hidden {
				public.Operation.Kind = hiddenAgendaKind
				public.Operation.Name = hiddenAgendaName
				public.Operation.InputKind = OperationInputPrivateInfo
				public.Operation.TargetCount = 0
				public.Operation.PublicInstruction = hiddenAgendaPublicInstruction
			}
		}
	}
	return public
}

func ProjectWithPublic(state State, playerID string, public PublicProjection) Projection {
	private := PrivateProjection{PlayerID: playerID}
	if player, ok := state.Players[playerID]; ok {
		private.Role = player.Role
		private.InitialFaction = player.InitialFaction
		private.Faction = player.Faction
		private.ApparentFaction = player.ApparentFaction
		_, private.VoteSubmitted = state.Vote.Submitted[playerID]
		private.CanSubmit = canSubmit(state, playerID)
		if definition, ok := roleDefinitionFor(string(player.Role)); ok && definition.Special {
			private.RoleName = definition.Name
			private.RoleDescription = definition.Description
			private.RoleEffect = definition.Effect
		}
		if player.Faction != "" && seesVirusRoster(player) {
			for _, id := range virusRoster(state) {
				if id == playerID {
					continue
				}
				if teammate, exists := state.Players[id]; exists {
					private.VirusRoster = append(private.VirusRoster, PublicPlayer{ID: teammate.ID, Name: teammate.Name, Seat: seatFor(state, id), Connected: teammate.Connected})
				}
			}
			private.VirusTeamSize = trueVirusCount(state)
		}
	}
	if state.Operation != nil {
		inputOwner := state.Operation.InputOwnerID
		if inputOwner == "" {
			inputOwner = state.ActivePlayerID
		}
		if definition, ok := operationDefinitionFor(state.Operation.Kind); ok {
			if state.Operation.Kind == "ChooseVoteShield" && state.Operation.Step == 2 {
				if playerID == state.Operation.TargetPlayerID {
					private.OperationInstruction = "Choose whether the active player receives extra suspicion or a vote shield in the accusation phase."
					if state.Phase == PhaseOperationInput && private.CanSubmit {
						private.Choices = []string{"EXTRA_SUSPICION", "VOTE_SHIELD"}
					}
				} else if playerID == state.ActivePlayerID {
					private.OperationInstruction = "Waiting for the selected agent to review your evidence..."
				}
			} else if playerID == inputOwner {
				private.OperationInstruction = definition.PrivateInstruction
				if definition.Hidden {
					private.OperationKind = definition.ID
					private.OperationName = definition.Name
				}
				if state.Phase == PhaseOperationInput && private.CanSubmit {
					private.LegalTargetIDs = legalOperationTargets(state, playerID, definition.TargetCount)
					if definition.InputKind == OperationInputChoice {
						private.Choices = []string{"STAY", "DEFECT"}
					}
				}
			}
		}
	}
	if state.Operation != nil && state.Operation.PrivateResults != nil {
		if result, ok := state.Operation.PrivateResults[playerID]; ok {
			resultCopy := result
			resultCopy.TargetPlayerIDs = append([]string(nil), result.TargetPlayerIDs...)
			private.OperationResult = &resultCopy
		}
	}
	return Projection{Type: "room.projection", Public: public, Private: private}
}

func publicRoomSettings(settings Settings) PublicRoomSettings {
	enabled := make([]string, 0, len(settings.EnabledOperations))
	for _, id := range operationOrder {
		if settings.EnabledOperations[id] {
			enabled = append(enabled, id)
		}
	}
	enabledRoles := make([]string, 0, len(settings.EnabledRoles))
	for _, definition := range roleDefinitions {
		if definition.Special && settings.EnabledRoles[definition.ID] {
			enabledRoles = append(enabledRoles, definition.ID)
		}
	}
	return PublicRoomSettings{DiscussionTimerEnabled: settings.DiscussionTimerEnabled, DiscussionSeconds: settings.DiscussionSeconds, EnabledOperations: enabled, MinPlayers: settings.MinPlayers, MaxPlayers: settings.MaxPlayers, InterludeSeconds: interludeSeconds(settings), VirusCount: settings.VirusCount, EnabledRoles: enabledRoles}
}

func seatFor(state State, playerID string) int {
	for index, id := range state.PlayerOrder {
		if id == playerID {
			return index + 1
		}
	}
	return 0
}

func legalOperationTargets(state State, activeID string, targetCount int) []string {
	if targetCount == 0 {
		return nil
	}
	players := make([]string, 0, len(state.PlayerOrder)-1)
	for _, id := range state.PlayerOrder {
		if id != activeID && state.Players[id].Connected {
			players = append(players, id)
		}
	}
	return players
}

func canSubmit(state State, playerID string) bool {
	player, ok := state.Players[playerID]
	if !ok {
		return false
	}
	switch state.Phase {
	case PhaseLobby:
		return !player.Ready
	case PhaseRoleReveal:
		return !state.RoleAcks[playerID]
	case PhaseOperationInput:
		if state.Operation != nil {
			inputOwner := state.Operation.InputOwnerID
			if inputOwner == "" {
				inputOwner = state.ActivePlayerID
			}
			return inputOwner == playerID
		}
		return state.ActivePlayerID == playerID
	case PhaseOperationResult:
		if state.Operation != nil && state.Operation.InputOwnerID != "" {
			return state.Operation.InputOwnerID == playerID || state.ActivePlayerID == playerID
		}
		return state.ActivePlayerID == playerID
	case PhaseOperationInterlude:
		return state.HostID == playerID
	case PhaseDiscussion:
		return state.HostID == playerID || !state.DiscussionAcks[playerID]
	case PhaseVoteInput:
		_, submitted := state.Vote.Submitted[playerID]
		return player.CanVote && !submitted
	case PhaseResultsIntro, PhaseVoteResults, PhaseImprisonment, PhaseAgencyReveal, PhaseOutcomeReveal, PhaseLeaderboard, PhaseOutOfLoop, PhaseEnd:
		return state.HostID == playerID
	default:
		return false
	}
}
