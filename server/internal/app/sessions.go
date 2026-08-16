package app

import (
	"tripleagent/server/internal/room"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
)

// Authenticate verifies a reconnect credential before exposing the
// active room actor. It intentionally preserves the old validation order.
func (s *Sessions) Authenticate(roomID, playerID string, reconnectToken string) (*room.Room, error) {
	if err := s.admit.ValidateReconnectToken(roomID, playerID, reconnectToken); err != nil {
		return nil, fault.Wrap(wrapSessionError(err), fmsg.With("failed to authenticate room session"))
	}
	activeRoom, ok := s.rooms.Get(roomID)
	if !ok {
		return nil, fault.Wrap(wrapSessionError(ErrRoomInactive), fmsg.With("failed to authenticate room session"))
	}
	return activeRoom, nil
}

type Session struct {
	// Room is the live actor this connection is attached to.
	Room *room.Room
	// RoomID is retained for credential revocation during detach.
	RoomID string
	// PlayerID identifies the authenticated player for this connection.
	PlayerID string
	// ID distinguishes this connection from an older connection for the same player.
	ID string
}

// Attach attaches a connection to the live room. Host transfer happens
// inside the actor and is not mirrored anywhere, so there is nothing to persist
// and nothing to roll back.
func (s *Sessions) Attach(activeRoom *room.Room, roomID, playerID string, sessionID string, sender room.Sender, closer room.Closer) (*Session, error) {
	if activeRoom == nil {
		return nil, fault.Wrap(wrapSessionError(ErrRoomInactive), fmsg.With("failed to attach room session"))
	}
	if err := activeRoom.Attach(playerID, sessionID, sender, closer); err != nil {
		return nil, fault.Wrap(wrapSessionError(err), fmsg.With("failed to attach room session"))
	}
	return &Session{Room: activeRoom, RoomID: roomID, PlayerID: playerID, ID: sessionID}, nil
}

// Detach removes the actor session while preserving the player's reconnect
// credential. Explicit lobby leave and room cleanup revoke credentials; a
// socket close must remain recoverable.
func (s *Sessions) Detach(session *Session) {
	if session == nil || session.Room == nil {
		return
	}
	session.Room.Detach(session.PlayerID, session.ID)
}
