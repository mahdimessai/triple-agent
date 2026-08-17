package domain

import (
	"errors"
	"fmt"
	"time"
)

type Phase string

const (
	PhaseLobby           Phase = "LOBBY"
	PhaseRoleReveal      Phase = "ROLE_REVEAL"
	PhaseOperationInput  Phase = "OPERATION_INPUT"
	PhaseOperationResult Phase = "OPERATION_RESULT"
	// A short timed beat between operations: the device goes back on the table
	// and the room talks about what just happened before the next operation.
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
	// Special roles. Each one belongs to a faction and is dealt only to a
	// player already on that faction, so a role never decides who wins with
	// whom; it only changes what the table can learn or do about that player.
	RoleFakeBlue  RoleKind = "FAKE_BLUE"  // Rogue Agent: VIRUS, kept off the VIRUS roster
	RoleFakeRed   RoleKind = "FAKE_RED"   // Triple Agent: Service, listed on the VIRUS roster
	RoleLyingRed  RoleKind = "LYING_RED"  // Deep Cover Agent: VIRUS, checks as Service
	RoleLyingBlue RoleKind = "LYING_BLUE" // Suspicious Agent: Service, checks as VIRUS
	RoleLoyalBlue RoleKind = "LOYAL_BLUE" // Service Loyalist: cannot be moved off Service
	RoleLoyalRed  RoleKind = "LOYAL_RED"  // VIRUS Loyalist: cannot be moved off VIRUS
)

type RoomSettings struct {
	MinPlayers             int  `json:"min_players"`
	MaxPlayers             int  `json:"max_players"`
	DiscussionTimerEnabled bool `json:"discussion_timer_enabled"`
	DiscussionSeconds      int  `json:"discussion_seconds"`
	// How long the room gets to talk between two operations.
	InterludeSeconds  int             `json:"interlude_seconds"`
	VirusCount        int             `json:"virus_count"`
	EnabledOperations map[string]bool `json:"enabled_operations"`
	EnabledRoles      map[string]bool `json:"enabled_roles"`
}

// interludeSeconds keeps rooms created before the setting existed playable, and
// holds the beat inside a sane range so a host cannot stall the table.
func interludeSeconds(settings RoomSettings) int {
	seconds := settings.InterludeSeconds
	if seconds <= 0 {
		seconds = 7
	}
	if seconds > 60 {
		seconds = 60
	}
	return seconds
}

func DefaultRoomSettings() RoomSettings {
	settings := RoomSettings{
		MinPlayers:             5,
		MaxPlayers:             9,
		DiscussionTimerEnabled: true,
		DiscussionSeconds:      300,
		InterludeSeconds:       7,
		VirusCount:             0,
	}
	settings.EnabledOperations = defaultEnabledOperations()
	settings.EnabledRoles = defaultEnabledRoles()
	return settings
}

type PlayerState struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Seat            int      `json:"seat"`
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

type OperationState struct {
	ID              string                     `json:"id"`
	Kind            string                     `json:"kind"`
	Name            string                     `json:"name"`
	InputKind       OperationInputKind         `json:"input_kind"`
	ActivePlayerID  string                     `json:"active_player_id"`
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

type EventRecord struct {
	ID         string    `json:"id"`
	Kind       string    `json:"kind"`
	PublicText string    `json:"public_text"`
	At         time.Time `json:"at"`
}

type GameState struct {
	RoomID           string                 `json:"room_id"`
	HostID           string                 `json:"host_id"`
	OwnerID          string                 `json:"owner_id"`
	Settings         RoomSettings           `json:"settings"`
	Phase            Phase                  `json:"phase"`
	Version          uint64                 `json:"version"`
	Players          map[string]PlayerState `json:"players"`
	PlayerOrder      []string               `json:"player_order"`
	RoleAcks         map[string]bool        `json:"role_acks,omitempty"`
	DiscussionAcks   map[string]bool        `json:"discussion_acks,omitempty"`
	ActivePlayerID   string                 `json:"active_player_id,omitempty"`
	PlannedOperation string                 `json:"planned_operation,omitempty"`
	// OperationQueue is the randomized recipient order. It is independent from
	// OperationDeck so a deck can contain more cards than the table has players.
	OperationQueue      []string `json:"operation_queue,omitempty"`
	OperationQueueIndex int      `json:"operation_queue_index,omitempty"`
	OperationDeck       []string `json:"operation_deck,omitempty"`
	OperationLastKind   string   `json:"operation_last_kind,omitempty"`
	OperationDealTarget int      `json:"operation_deal_target,omitempty"`
	OperationDeals      int      `json:"operation_deals,omitempty"`
	// OperationsDealt records recipients, not operation IDs. It remains useful
	// for reconnect/debug history while OperationDeck owns global uniqueness.
	OperationsDealt    []string        `json:"operations_dealt,omitempty"`
	Operation          *OperationState `json:"operation,omitempty"`
	DiscussionDeadline *time.Time      `json:"discussion_deadline,omitempty"`
	Vote               VoteState       `json:"vote"`
	Winner             Faction         `json:"winner"`
	RandomState        uint64          `json:"-"`
}

type Command struct {
	RequestID              string      `json:"request_id"`
	ActorID                string      `json:"actor_id"`
	ExpectedVersion        uint64      `json:"expected_version"`
	Kind                   CommandKind `json:"kind"`
	OperationKind          string      `json:"operation_kind,omitempty"`
	RoleID                 string      `json:"role_id,omitempty"`
	RoleEnabled            bool        `json:"role_enabled,omitempty"`
	OperationEnabled       bool        `json:"operation_enabled,omitempty"`
	DiscussionTimerEnabled bool        `json:"discussion_timer_enabled,omitempty"`
	DiscussionSeconds      int         `json:"discussion_seconds,omitempty"`
	VirusCount             int         `json:"virus_count,omitempty"`
	TargetID               string      `json:"target_id,omitempty"`
	TargetIDs              []string    `json:"target_ids,omitempty"`
	Choice                 string      `json:"choice,omitempty"`
}

type Transition struct {
	State   GameState
	Event   EventRecord
	Changed bool
}

var (
	ErrStaleVersion         = errors.New("stale room version")
	ErrNotAllowed           = errors.New("command is not allowed")
	ErrNotEnoughPlayers     = errors.New("not enough players")
	ErrNotReady             = errors.New("all players must be ready")
	ErrInvalidTarget        = errors.New("invalid target")
	ErrUnknownOperation     = errors.New("unknown operation")
	ErrNoEligibleOperations = errors.New("no eligible operations are enabled")
	ErrAlreadySubmitted     = errors.New("command already submitted")
	ErrRoomFull             = errors.New("room is full")
	ErrPlayerNotInRoom      = errors.New("player is not in room")
)

func NewLobby(roomID, hostID string, hostName string, settings RoomSettings) GameState {
	if settings.MinPlayers == 0 {
		settings = DefaultRoomSettings()
	}
	if settings.EnabledOperations == nil {
		settings.EnabledOperations = defaultEnabledOperations()
	}
	return GameState{
		RoomID:      roomID,
		HostID:      hostID,
		OwnerID:     hostID,
		Settings:    settings,
		Phase:       PhaseLobby,
		Players:     map[string]PlayerState{hostID: {ID: hostID, Name: hostName, Seat: 1, Connected: true, CanVote: true, VotingPower: 1}},
		PlayerOrder: []string{hostID},
		Vote:        VoteState{Submitted: map[string]string{}, Totals: map[string]int{}},
		RandomState: uint64(time.Now().UnixNano()),
	}
}

func (s GameState) Player(id string) (PlayerState, bool) {
	player, ok := s.Players[id]
	return player, ok
}

func (s GameState) PublicPlayerCount() int {
	return len(s.PlayerOrder)
}

func (s *GameState) AddPlayer(id string, name string) error {
	if s.Phase != PhaseLobby {
		return ErrNotAllowed
	}
	if len(s.PlayerOrder) >= s.Settings.MaxPlayers {
		return ErrRoomFull
	}
	if _, exists := s.Players[id]; exists {
		return fmt.Errorf("player already exists")
	}
	s.Players[id] = PlayerState{ID: id, Name: name, Seat: len(s.PlayerOrder) + 1, CanVote: true, VotingPower: 1}
	s.PlayerOrder = append(s.PlayerOrder, id)
	return nil
}
