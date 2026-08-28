package room

import "tripleagent/server/internal/game"

// RoomMessage is the interface implemented by all commands and queries dispatched to the Room actor.
type RoomMessage interface {
	isRoomMessage()
}

type JoinCmd struct {
	PlayerID string
	Name     string
	Token    string
	Reply    chan error
}

func (JoinCmd) isRoomMessage() {}

type LeaveCmd struct {
	PlayerID string
	Token    string
	Reply    chan error
}

func (LeaveCmd) isRoomMessage() {}

type AttachSessionCmd struct {
	PlayerID string // Optional if Token is provided
	Token    string
	Session  ClientSession
	Reply    chan AttachResult
}

func (AttachSessionCmd) isRoomMessage() {}

type AttachResult struct {
	PlayerID   string
	Projection game.Projection
	Err        error
}

type DetachSessionCmd struct {
	PlayerID  string
	SessionID string
	Reply     chan error
}

func (DetachSessionCmd) isRoomMessage() {}

type ReleaseSessionCmd struct {
	Token string
	Reply chan error
}

func (ReleaseSessionCmd) isRoomMessage() {}

type ExecGameCmd struct {
	PlayerID        string
	SessionID       string
	ExpectedVersion uint64
	Command         game.Command
	Reply           chan error
}

func (ExecGameCmd) isRoomMessage() {}

type SnapshotQuery struct {
	PlayerID string
	Reply    chan SnapshotResult
}

func (SnapshotQuery) isRoomMessage() {}

type SnapshotResult struct {
	Projection game.Projection
	Err        error
}

type PlayerIDForTokenQuery struct {
	Token string
	Reply chan PlayerIDResult
}

func (PlayerIDForTokenQuery) isRoomMessage() {}

type PlayerIDResult struct {
	PlayerID string
	Err      error
}
