package room

import (
	"sync"
	"time"

	"tripleagent/server/internal/domain"
)

const (
	roomLifetime      = 4 * time.Hour
	endedRoomLifetime = 15 * time.Minute
)

type Manager struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

func NewManager() *Manager {
	return &Manager{rooms: make(map[string]*Room)}
}

func (m *Manager) Create(state domain.GameState) *Room {
	return m.CreateWithCleanup(state, nil)
}

func (m *Manager) CreateWithCleanup(state domain.GameState, cleanup func()) *Room {
	var room *Room
	room = newRoom(state, roomLifetime, endedRoomLifetime, func() {
		if m.removeIf(state.RoomID, room) && cleanup != nil {
			go cleanup()
		}
	})
	room.cleanup = cleanup
	m.mu.Lock()
	old := m.rooms[state.RoomID]
	m.rooms[state.RoomID] = room
	m.mu.Unlock()
	if old != nil {
		old.Close()
	}
	return room
}

func (m *Manager) Get(roomID string) (*Room, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	room, ok := m.rooms[roomID]
	return room, ok
}

// Remove retires a room explicitly. It runs the same cleanup as expiry does, so
// a room's lifetime has exactly one owner however it ends.
func (m *Manager) Remove(roomID string) {
	m.mu.Lock()
	room := m.rooms[roomID]
	delete(m.rooms, roomID)
	m.mu.Unlock()
	if room == nil {
		return
	}
	room.Close()
	if room.cleanup != nil {
		room.cleanup()
	}
}

func (m *Manager) removeIf(roomID string, expected *Room) bool {
	m.mu.Lock()
	room, ok := m.rooms[roomID]
	if ok && room == expected {
		delete(m.rooms, roomID)
	}
	m.mu.Unlock()
	if ok && room == expected {
		expected.Close()
		return true
	}
	return false
}
