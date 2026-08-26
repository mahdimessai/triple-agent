package room

import (
	"errors"
	"strings"
	"sync"

	"tripleagent/server/internal/game"
)

// PlayerCredentials represents the joining credentials returned to a client upon creating or joining a room.
type PlayerCredentials struct {
	JoinCode       string
	ReconnectToken string
}

// RoomManager maintains the catalog of active rooms, join code mapping, and lifecycle cleanup.
type RoomManager struct {
	mu    sync.RWMutex
	rooms map[string]*Room
	codes map[string]string
}

// Registry is a type alias for RoomManager for backward compatibility.
type Registry = RoomManager

// NewManager creates an empty RoomManager.
func NewManager() *RoomManager {
	return &RoomManager{
		rooms: make(map[string]*Room),
		codes: make(map[string]string),
	}
}

// NewRegistry creates a new Registry (alias for NewManager).
func NewRegistry() *Registry {
	return NewManager()
}

// Create generates a new room hosted by the specified player name.
func (m *RoomManager) Create(playerName string) (PlayerCredentials, error) {
	roomID := "room_" + randomHex(6)
	playerID := "player_" + randomHex(6)
	reconnectToken := randomHex(24)
	joinCode := newJoinCode()

	tokens := NewTokenDirectory()
	tokens.Register(playerID, reconnectToken)
	state := game.NewLobby(playerID, playerName)

	active := newRoom(roomID, state, tokens, func(closed *Room) {
		m.remove(roomID, joinCode, closed)
	})

	m.mu.Lock()
	m.rooms[roomID] = active
	m.codes[joinCode] = roomID
	m.mu.Unlock()

	return PlayerCredentials{
		JoinCode:       joinCode,
		ReconnectToken: reconnectToken,
	}, nil
}

// Join adds a player to an existing room by join code.
func (m *RoomManager) Join(joinCode, playerName string) (PlayerCredentials, error) {
	active, ok := m.GetByCode(joinCode)
	if !ok {
		return PlayerCredentials{}, ErrRoomNotFound
	}
	playerID := "player_" + randomHex(6)
	token := randomHex(24)
	if err := active.Join(playerID, strings.TrimSpace(playerName), token); err != nil {
		if errors.Is(err, ErrClosed) {
			return PlayerCredentials{}, ErrRoomNotFound
		}
		return PlayerCredentials{}, err
	}
	return PlayerCredentials{JoinCode: joinCode, ReconnectToken: token}, nil
}

// GetByCode returns the active room associated with a join code.
func (m *RoomManager) GetByCode(joinCode string) (*Room, bool) {
	m.mu.RLock()
	roomID, ok := m.codes[joinCode]
	active := m.rooms[roomID]
	m.mu.RUnlock()
	return active, ok && active != nil
}

// Resolve looks up a room and verifies a player's reconnect token.
func (m *RoomManager) Resolve(joinCode, token string) (*Room, string, error) {
	active, ok := m.GetByCode(joinCode)
	if !ok {
		return nil, "", ErrRoomNotFound
	}
	playerID, err := active.PlayerIDForToken(token)
	if err != nil {
		return nil, "", err
	}
	return active, playerID, nil
}

// Leave removes a player from a room using their credentials.
func (m *RoomManager) Leave(joinCode, token string) error {
	active, playerID, err := m.Resolve(joinCode, token)
	if err != nil {
		return err
	}
	return active.Leave(playerID, token)
}

// Close closes all active rooms managed by this RoomManager.
func (m *RoomManager) Close() {
	m.mu.Lock()
	rooms := make([]*Room, 0, len(m.rooms))
	for _, active := range m.rooms {
		rooms = append(rooms, active)
	}
	m.rooms = make(map[string]*Room)
	m.codes = make(map[string]string)
	m.mu.Unlock()

	for _, active := range rooms {
		active.Close()
	}
}

func (m *RoomManager) remove(roomID, joinCode string, expected *Room) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if current, ok := m.rooms[roomID]; !ok || current != expected {
		return
	}
	delete(m.rooms, roomID)
	if m.codes[joinCode] == roomID {
		delete(m.codes, joinCode)
	}
}
