package app

import (
	"strings"

	"tripleagent/server/internal/admission"
	"tripleagent/server/internal/domain"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
)

type CreateInput struct {
	// PlayerName is the host name to seat in the newly created room.
	PlayerName string
}

type CreateResult struct {
	// RoomID identifies the newly created live room.
	RoomID         string
	// JoinCode is the human-readable code other players use to join.
	JoinCode       string
	// PlayerID identifies the host's seat.
	PlayerID       string
	// ReconnectToken authenticates the host's future WebSocket session.
	ReconnectToken string
}

// Create claims a join code and then starts the room that owns the game state.
func (l *Lobbies) Create(input CreateInput) (CreateResult, error) {
	settings := domain.DefaultRoomSettings()
	hostName := strings.TrimSpace(input.PlayerName)
	roomID := admission.NewRoomID()
	hostID := admission.NewPlayerID()
	token := admission.NewReconnectToken()
	joinCode := admission.NewJoinCode()
	if err := l.admit.Reserve(roomID, joinCode, hostID, token); err != nil {
		return CreateResult{}, fault.Wrap(wrapLobbyError(err), fmsg.With("failed to create lobby"))
	}
	l.rooms.CreateWithCleanup(domain.NewLobby(roomID, hostID, hostName, settings), func() {
		l.admit.Release(roomID)
	})
	return CreateResult{RoomID: roomID, JoinCode: joinCode, PlayerID: hostID, ReconnectToken: token}, nil
}

type JoinInput struct {
	// JoinCode identifies the lobby the player wants to enter.
	JoinCode   string
	// PlayerName is the display name to place in the room.
	PlayerName string
}

type JoinResult struct {
	// RoomID identifies the room that accepted the player.
	RoomID         string
	// JoinCode is the canonical code stored for the room.
	JoinCode       string
	// PlayerID identifies the admitted seat.
	PlayerID       string
	// ReconnectToken authenticates the player's future WebSocket session.
	ReconnectToken string
}

type LeaveInput struct {
	// RoomID identifies the room whose lobby seat should be released.
	RoomID         string
	// PlayerID identifies the seat being released.
	PlayerID       string
	// ReconnectToken authenticates the request before the seat is removed.
	ReconnectToken string
}

// Join validates the join code, adds the player to the room, and registers their credential.
func (l *Lobbies) Join(input JoinInput) (JoinResult, error) {
	roomID, joinCode, err := l.admit.ResolveCode(input.JoinCode)
	if err != nil {
		return JoinResult{}, fault.Wrap(wrapLobbyError(err), fmsg.With("failed to join lobby"))
	}
	activeRoom, ok := l.rooms.Get(roomID)
	if !ok {
		return JoinResult{}, fault.Wrap(wrapLobbyError(ErrRoomInactive), fmsg.With("failed to join lobby"))
	}
	joinName := strings.TrimSpace(input.PlayerName)
	playerID := admission.NewPlayerID()
	token := admission.NewReconnectToken()

	if err := activeRoom.AddPlayer(playerID, joinName); err != nil {
		return JoinResult{}, fault.Wrap(wrapLobbyError(err), fmsg.With("failed to join lobby"))
	}
	if err := l.admit.Grant(roomID, playerID, token); err != nil {
		return JoinResult{}, fault.Wrap(wrapLobbyError(err), fmsg.With("failed to join lobby"))
	}

	return JoinResult{
		RoomID:         roomID,
		JoinCode:       joinCode,
		PlayerID:       playerID,
		ReconnectToken: token,
	}, nil
}

// Leave removes an authenticated player from an unstarted room and revokes their credential.
func (l *Lobbies) Leave(input LeaveInput) error {
	if err := l.admit.ValidateReconnectToken(input.RoomID, input.PlayerID, input.ReconnectToken); err != nil {
		return fault.Wrap(wrapLobbyError(err), fmsg.With("failed to leave lobby"))
	}
	activeRoom, ok := l.rooms.Get(input.RoomID)
	if !ok {
		return fault.Wrap(wrapLobbyError(ErrRoomInactive), fmsg.With("failed to leave lobby"))
	}
	current, err := activeRoom.Snapshot(input.PlayerID)
	if err != nil {
		return fault.Wrap(wrapLobbyError(err), fmsg.With("failed to leave lobby"))
	}
	if current.Public.Phase != domain.PhaseLobby {
		return fault.Wrap(wrapLobbyError(ErrLobbyStarted), fmsg.With("failed to leave lobby"))
	}
	state, err := activeRoom.RemovePlayerWithCommit(input.PlayerID, func(domain.GameState) error {
		l.admit.Revoke(input.RoomID, input.PlayerID)
		return nil
	})
	if err != nil {
		return fault.Wrap(wrapLobbyError(err), fmsg.With("failed to leave lobby"))
	}
	if len(state.PlayerOrder) == 0 {
		l.rooms.Remove(input.RoomID)
	}
	return nil
}
