package app

import "errors"

var (
	ErrRoomInactive = errors.New("room is no longer active")
	ErrLobbyStarted = errors.New("lobby has already started")
)
