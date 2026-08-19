package game

import (
	"errors"
	"time"
)

type Phase string

const (
	PhaseLobby              Phase = "LOBBY"
	PhaseRoleReveal         Phase = "ROLE_REVEAL"
	PhaseOperationInput     Phase = "OPERATION_INPUT"
	PhaseOperationResult    Phase = "OPERATION_RESULT"
	PhaseOperationInterlude Phase = "OPERATION_INTERLUDE"
	PhaseDiscussion         Phase = "DISCUSSION"
	PhaseVoteInput          Phase = "VOTE_INPUT"
	PhaseResultsIntro       Phase = "RESULTS_INTRO"
	PhaseVoteResults        Phase = "VOTE_RESULTS"
	PhaseImprisonment       Phase = "IMPRISONMENT_REVEAL"
	PhaseAgencyReveal       Phase = "AGENCY_REVEAL"
	PhaseOutcomeReveal      Phase = "OUTCOME_REVEAL"
	PhaseLeaderboard        Phase = "LEADERBOARD"
	PhaseOutOfLoop          Phase = "OUT_OF_LOOP"
	PhaseEnd                Phase = "END"
)

type CommandKind string

const (
	CommandSetReady              CommandKind = "lobby.ready"
	CommandStartMatch            CommandKind = "match.start"
	CommandAcknowledgeRole       CommandKind = "role.acknowledge"
	CommandResolveOperation      CommandKind = "operation.resolve"
	CommandSelectOperationTarget CommandKind = "operation.select_target"
	CommandOperationExplainDone  CommandKind = "operation.explain_done"
	CommandAdvanceInterlude      CommandKind = "interlude.advance"
	CommandAdvanceDiscussion     CommandKind = "discussion.advance"
	CommandSubmitVote            CommandKind = "vote.submit"
	CommandContinueResults       CommandKind = "results.continue"
	CommandRematch               CommandKind = "match.rematch"
	CommandSetOperationEnabled   CommandKind = "lobby.operation_enabled"
	CommandSetRoleEnabled        CommandKind = "lobby.role_enabled"
	CommandSetDiscussionTimer    CommandKind = "lobby.discussion_timer"
	CommandSetVirusCount         CommandKind = "lobby.virus_count"
	CommandTransferHost          CommandKind = "lobby.transfer_host"
	CommandKickPlayer            CommandKind = "lobby.kick_player"
)

type Faction string

const (
	FactionService Faction = "SERVICE"
	FactionVirus   Faction = "VIRUS"
	FactionNone    Faction = "NONE"
)

type RoleKind string

const (
	RoleNormalBlue RoleKind = "NORMAL_BLUE"
	RoleNormalRed  RoleKind = "NORMAL_RED"
	RoleFakeBlue   RoleKind = "FAKE_BLUE"
	RoleFakeRed    RoleKind = "FAKE_RED"
	RoleLyingRed   RoleKind = "LYING_RED"
	RoleLyingBlue  RoleKind = "LYING_BLUE"
	RoleLoyalBlue  RoleKind = "LOYAL_BLUE"
	RoleLoyalRed   RoleKind = "LOYAL_RED"
)

type Settings struct {
	MinPlayers             int             `json:"min_players"`
	MaxPlayers             int             `json:"max_players"`
	DiscussionTimerEnabled bool            `json:"discussion_timer_enabled"`
	DiscussionSeconds      int             `json:"discussion_seconds"`
	InterludeSeconds       int             `json:"interlude_seconds"`
	VirusCount             int             `json:"virus_count"`
	EnabledOperations      map[string]bool `json:"enabled_operations"`
	EnabledRoles           map[string]bool `json:"enabled_roles"`
}

func DefaultSettings() Settings {
	return Settings{
		MinPlayers:             5,
		MaxPlayers:             9,
		DiscussionTimerEnabled: true,
		DiscussionSeconds:      300,
		InterludeSeconds:       7,
		VirusCount:             0,
		EnabledOperations:      defaultEnabledOperations(),
		EnabledRoles:           defaultEnabledRoles(),
	}
}

func interludeSeconds(settings Settings) int {
	seconds := settings.InterludeSeconds
	if seconds <= 0 {
		seconds = 7
	}
	if seconds > 60 {
		seconds = 60
	}
	return seconds
}

type Player struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Ready           bool     `json:"ready"`
	Connected       bool     `json:"connected"`
	InitialFaction  Faction  `json:"initial_faction"`
	Faction         Faction  `json:"faction"`
	ApparentFaction *Faction `json:"apparent_faction,omitempty"`
	Role            RoleKind `json:"role"`
	CanVote         bool     `json:"can_vote"`
	VotingPower     int      `json:"voting_power"`
	Statuses        []string `json:"statuses,omitempty"`
	ObjectiveKind   string   `json:"objective_kind,omitempty"`
	ObjectiveTarget string   `json:"objective_target,omitempty"`
	DealtOperations []string `json:"dealt_operations,omitempty"`
	DealtCategories []int    `json:"dealt_categories,omitempty"`
}

type OperationInputKind string

const (
	OperationInputNone        OperationInputKind = "NONE"
	OperationInputOneTarget   OperationInputKind = "ONE_TARGET"
	OperationInputTwoTargets  OperationInputKind = "TWO_TARGETS"
	OperationInputChoice      OperationInputKind = "CHOICE"
	OperationInputPrivateInfo OperationInputKind = "PRIVATE_INFO"
)

type OperationState struct {
	Kind            string                     `json:"kind"`
	InputOwnerID    string                     `json:"input_owner_id,omitempty"`
	Step            int                        `json:"step,omitempty"`
	TargetPlayerID  string                     `json:"target_player_id,omitempty"`
	TargetPlayerIDs []string                   `json:"target_player_ids,omitempty"`
	PrivateResults  map[string]OperationResult `json:"private_results,omitempty"`
}

type OperationResult struct {
	Code            string   `json:"code,omitempty"`
	TargetPlayerID  string   `json:"target_player_id,omitempty"`
	TargetPlayerIDs []string `json:"target_player_ids,omitempty"`
	TargetFaction   Faction  `json:"target_faction,omitempty"`
	OtherPlayerID   string   `json:"other_player_id,omitempty"`
	OtherFaction    Faction  `json:"other_faction,omitempty"`
	YourFaction     Faction  `json:"your_faction,omitempty"`
	Message         string   `json:"message"`
}

type VoteState struct {
	Submitted          map[string]string `json:"submitted"`
	Totals             map[string]int    `json:"totals"`
	ImprisonedPlayerID string            `json:"imprisoned_player_id,omitempty"`
}

type State struct {
	HostID              string            `json:"host_id"`
	Settings            Settings          `json:"settings"`
	Phase               Phase             `json:"phase"`
	Version             uint64            `json:"version"`
	Players             map[string]Player `json:"players"`
	PlayerOrder         []string          `json:"player_order"`
	RoleAcks            map[string]bool   `json:"role_acks,omitempty"`
	DiscussionAcks      map[string]bool   `json:"discussion_acks,omitempty"`
	ActivePlayerID      string            `json:"active_player_id,omitempty"`
	PlannedOperation    string            `json:"planned_operation,omitempty"`
	OperationQueue      []string          `json:"operation_queue,omitempty"`
	OperationQueueIndex int               `json:"operation_queue_index,omitempty"`
	OperationDeck       []string          `json:"operation_deck,omitempty"`
	OperationLastKind   string            `json:"operation_last_kind,omitempty"`
	OperationDealTarget int               `json:"operation_deal_target,omitempty"`
	OperationDeals      int               `json:"operation_deals,omitempty"`
	OperationsDealt     []string          `json:"operations_dealt,omitempty"`
	Operation           *OperationState   `json:"operation,omitempty"`
	DiscussionDeadline  *time.Time        `json:"discussion_deadline,omitempty"`
	Vote                VoteState         `json:"vote"`
	Winner              Faction           `json:"winner"`
	RandomState         uint64            `json:"-"`
}

type Command struct {
	Kind                   CommandKind
	OperationKind          string
	RoleID                 string
	RoleEnabled            bool
	OperationEnabled       bool
	DiscussionTimerEnabled bool
	DiscussionSeconds      int
	VirusCount             int
	TargetID               string
	TargetIDs              []string
	Choice                 string
}

var (
	ErrNotAllowed           = errors.New("command is not allowed")
	ErrNotEnoughPlayers     = errors.New("not enough players")
	ErrNotReady             = errors.New("all players must be ready")
	ErrInvalidTarget        = errors.New("invalid target")
	ErrUnknownOperation     = errors.New("unknown operation")
	ErrNoEligibleOperations = errors.New("no eligible operations are enabled")
	ErrAlreadySubmitted     = errors.New("command already submitted")
	ErrRoomFull             = errors.New("room is full")
	ErrPlayerNotInRoom      = errors.New("player is not in room")
	ErrPlayerExists         = errors.New("player already exists")
)

func committed(state State) State {
	state.Version++
	return state
}

func cloneState(state State) State {
	state.Settings.EnabledOperations = cloneBoolMap(state.Settings.EnabledOperations)
	state.Settings.EnabledRoles = cloneBoolMap(state.Settings.EnabledRoles)

	players := make(map[string]Player, len(state.Players))
	for id, player := range state.Players {
		player.Statuses = append([]string(nil), player.Statuses...)
		player.DealtOperations = append([]string(nil), player.DealtOperations...)
		player.DealtCategories = append([]int(nil), player.DealtCategories...)
		if player.ApparentFaction != nil {
			faction := *player.ApparentFaction
			player.ApparentFaction = &faction
		}
		players[id] = player
	}
	state.Players = players
	state.PlayerOrder = append([]string(nil), state.PlayerOrder...)
	state.RoleAcks = cloneBoolMap(state.RoleAcks)
	state.DiscussionAcks = cloneBoolMap(state.DiscussionAcks)
	state.OperationQueue = append([]string(nil), state.OperationQueue...)
	state.OperationDeck = append([]string(nil), state.OperationDeck...)
	state.OperationsDealt = append([]string(nil), state.OperationsDealt...)
	if state.Operation != nil {
		operation := *state.Operation
		operation.TargetPlayerIDs = append([]string(nil), operation.TargetPlayerIDs...)
		if state.Operation.PrivateResults != nil {
			operation.PrivateResults = make(map[string]OperationResult, len(state.Operation.PrivateResults))
			for id, result := range state.Operation.PrivateResults {
				result.TargetPlayerIDs = append([]string(nil), result.TargetPlayerIDs...)
				operation.PrivateResults[id] = result
			}
		}
		state.Operation = &operation
	}
	if state.DiscussionDeadline != nil {
		deadline := *state.DiscussionDeadline
		state.DiscussionDeadline = &deadline
	}
	state.Vote.Submitted = cloneStringMap(state.Vote.Submitted)
	state.Vote.Totals = cloneIntMap(state.Vote.Totals)
	return state
}

func cloneBoolMap(source map[string]bool) map[string]bool {
	if source == nil {
		return nil
	}
	copyMap := make(map[string]bool, len(source))
	for key, value := range source {
		copyMap[key] = value
	}
	return copyMap
}

func cloneStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	copyMap := make(map[string]string, len(source))
	for key, value := range source {
		copyMap[key] = value
	}
	return copyMap
}

func cloneIntMap(source map[string]int) map[string]int {
	if source == nil {
		return nil
	}
	copyMap := make(map[string]int, len(source))
	for key, value := range source {
		copyMap[key] = value
	}
	return copyMap
}
