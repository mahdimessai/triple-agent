package transport

type lobbyResponse struct {
	// RoomID identifies the live room created or joined by the caller.
	RoomID string `json:"room_id"`
	// JoinCode is the human-readable code used to find the room.
	JoinCode string `json:"join_code"`
	// PlayerID identifies the caller's seat in the room.
	PlayerID string `json:"player_id"`
	// ReconnectToken authenticates the caller's WebSocket connection.
	ReconnectToken string `json:"reconnect_token"`
}
