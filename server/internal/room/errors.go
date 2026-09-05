package room

import "errors"

var (
	ErrClosed          = errors.New("room is closed")
	ErrUnauthorized    = errors.New("invalid reconnect token")
	ErrSessionGone     = errors.New("session is no longer active")
	ErrSessionActive   = errors.New("session already active for this player")
	ErrVersionConflict = errors.New("stale room version")
	ErrRoomNotFound    = errors.New("room not found")
)
