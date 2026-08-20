package app

import (
	"tripleagent/server/internal/room"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
)

// Authenticate resolves the live room and lets its actor verify the seat-owned
// reconnect credential.
func (s *Sessions) Authenticate(roomID, playerID, reconnectToken string) (*room.Room, error) {
	activeRoom, ok := s.rooms.Get(roomID)
	if !ok {
		return nil, fault.Wrap(wrapSessionError(ErrRoomInactive), fmsg.With("failed to authenticate room session"))
	}
	if err := activeRoom.Authenticate(playerID, reconnectToken); err != nil {
		return nil, fault.Wrap(wrapSessionError(err), fmsg.With("failed to authenticate room session"))
	}
	return activeRoom, nil
}

type Session struct {
	Room     *room.Room
	RoomID   string
	PlayerID string
	ID       string
}

// Attach attaches a connection to the live room. Session replacement and
// presence changes remain serialized by the room actor.
func (s *Sessions) Attach(activeRoom *room.Room, roomID, playerID, sessionID string, sender room.Sender, closer room.Closer) (*Session, error) {
	if activeRoom == nil {
		return nil, fault.Wrap(wrapSessionError(ErrRoomInactive), fmsg.With("failed to attach room session"))
	}
	if err := activeRoom.Attach(playerID, sessionID, sender, closer); err != nil {
		return nil, fault.Wrap(wrapSessionError(err), fmsg.With("failed to attach room session"))
	}
	return &Session{Room: activeRoom, RoomID: roomID, PlayerID: playerID, ID: sessionID}, nil
}

// Detach tells the actor that this connection disappeared. Lobby-seat release,
// credential revocation, host transfer, and empty-room retirement all happen in
// the same actor turn.
func (s *Sessions) Detach(session *Session) {
	if session == nil || session.Room == nil {
		return
	}
	session.Room.Detach(session.PlayerID, session.ID)
}
