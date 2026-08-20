package app

import (
	"strings"

	"tripleagent/server/internal/admission"
	"tripleagent/server/internal/domain"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
)

type CreateInput struct {
	PlayerName string
}

type CreateResult struct {
	RoomID         string
	JoinCode       string
	PlayerID       string
	ReconnectToken string
}

// Create claims the cross-room join code, then starts an actor that owns the
// host seat and reconnect credential together.
func (l *Lobbies) Create(input CreateInput) (CreateResult, error) {
	settings := domain.DefaultRoomSettings()
	hostName := strings.TrimSpace(input.PlayerName)
	roomID := admission.NewRoomID()
	hostID := admission.NewPlayerID()
	token := admission.NewReconnectToken()
	joinCode := admission.NewJoinCode()

	if err := l.admit.Reserve(roomID, joinCode); err != nil {
		return CreateResult{}, fault.Wrap(wrapLobbyError(err), fmsg.With("failed to create lobby"))
	}

	l.rooms.CreateWithCredentials(
		domain.NewLobby(roomID, hostID, hostName, settings),
		map[string]string{hostID: token},
		func() { l.admit.Release(roomID) },
	)

	return CreateResult{
		RoomID:         roomID,
		JoinCode:       joinCode,
		PlayerID:       hostID,
		ReconnectToken: token,
	}, nil
}

type JoinInput struct {
	JoinCode   string
	PlayerName string
}

type JoinResult struct {
	RoomID         string
	JoinCode       string
	PlayerID       string
	ReconnectToken string
}

type LeaveInput struct {
	RoomID         string
	PlayerID       string
	ReconnectToken string
}

// Join resolves the cross-room code, then lets the room actor atomically create
// both the seat and its reconnect credential.
func (l *Lobbies) Join(input JoinInput) (JoinResult, error) {
	roomID, joinCode, err := l.admit.ResolveCode(input.JoinCode)
	if err != nil {
		return JoinResult{}, fault.Wrap(wrapLobbyError(err), fmsg.With("failed to join lobby"))
	}
	activeRoom, ok := l.rooms.Get(roomID)
	if !ok {
		return JoinResult{}, fault.Wrap(wrapLobbyError(ErrRoomInactive), fmsg.With("failed to join lobby"))
	}

	playerID := admission.NewPlayerID()
	token := admission.NewReconnectToken()
	if err := activeRoom.AddPlayerWithCredential(playerID, strings.TrimSpace(input.PlayerName), token); err != nil {
		return JoinResult{}, fault.Wrap(wrapLobbyError(err), fmsg.With("failed to join lobby"))
	}

	return JoinResult{
		RoomID:         roomID,
		JoinCode:       joinCode,
		PlayerID:       playerID,
		ReconnectToken: token,
	}, nil
}

// Leave authenticates against the live room, then atomically releases the seat
// and credential. If it was the final lobby seat, the actor retires itself and
// its manager releases the join code.
func (l *Lobbies) Leave(input LeaveInput) error {
	activeRoom, ok := l.rooms.Get(input.RoomID)
	if !ok {
		return fault.Wrap(wrapLobbyError(ErrRoomInactive), fmsg.With("failed to leave lobby"))
	}
	if err := activeRoom.Authenticate(input.PlayerID, input.ReconnectToken); err != nil {
		return fault.Wrap(wrapLobbyError(err), fmsg.With("failed to leave lobby"))
	}
	if err := activeRoom.RemovePlayer(input.PlayerID); err != nil {
		return fault.Wrap(wrapLobbyError(err), fmsg.With("failed to leave lobby"))
	}
	return nil
}
