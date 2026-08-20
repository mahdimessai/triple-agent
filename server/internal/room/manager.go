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
	return m.CreateWithCredentials(state, nil, nil)
}

func (m *Manager) CreateWithCleanup(state domain.GameState, cleanup func()) *Room {
	return m.CreateWithCredentials(state, nil, cleanup)
}

// CreateWithCredentials starts a room whose actor owns the supplied reconnect
// credentials. cleanup releases the cross-room resources, such as the join code,
// when this exact room instance retires.
func (m *Manager) CreateWithCredentials(state domain.GameState, credentials map[string]string, cleanup func()) *Room {
	var active *Room
	retire := func() {
		if m.removeIf(state.RoomID, active) && cleanup != nil {
			cleanup()
		}
	}
	active = newRoomWithCredentials(state, credentials, roomLifetime, endedRoomLifetime, retire)
	active.cleanup = cleanup

	m.mu.Lock()
	old := m.rooms[state.RoomID]
	m.rooms[state.RoomID] = active
	m.mu.Unlock()
	if old != nil {
		old.Close()
	}
	return active
}

func (m *Manager) Get(roomID string) (*Room, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	active, ok := m.rooms[roomID]
	return active, ok
}

// Remove retires a room explicitly. Actor-driven expiry and empty-room cleanup
// use the same cleanup callback through removeIf.
func (m *Manager) Remove(roomID string) {
	m.mu.Lock()
	active := m.rooms[roomID]
	delete(m.rooms, roomID)
	m.mu.Unlock()
	if active == nil {
		return
	}
	active.Close()
	if active.cleanup != nil {
		active.cleanup()
	}
}

// removeIf removes an actor that is retiring itself. It deliberately does not
// call Close because the actor closes its own done channel after the callback.
func (m *Manager) removeIf(roomID string, expected *Room) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	active, ok := m.rooms[roomID]
	if !ok || active != expected {
		return false
	}
	delete(m.rooms, roomID)
	return true
}
