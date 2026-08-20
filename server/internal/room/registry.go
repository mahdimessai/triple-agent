package room

import (
	"errors"
	"strings"
	"sync"

	"tripleagent/server/internal/game"
)

var ErrRoomNotFound = errors.New("room not found")

type Identity struct {
	RoomID         string
	JoinCode       string
	PlayerID       string
	ReconnectToken string
}

type Registry struct {
	mu    sync.RWMutex
	rooms map[string]*Room
	codes map[string]string
}

func NewRegistry() *Registry {
	return &Registry{rooms: make(map[string]*Room), codes: make(map[string]string)}
}

func (r *Registry) Create(playerName string) (Identity, error) {
	for {
		roomID := "room_" + randomHex(6)
		playerID := "player_" + randomHex(6)
		token := randomHex(24)
		joinCode := newJoinCode()
		state := game.NewLobbyWithSeed(playerID, strings.TrimSpace(playerName), game.DefaultSettings(), randomUint64())
		active := newRoom(roomID, state, map[string]string{playerID: token}, func(closed *Room) {
			r.remove(roomID, joinCode, closed)
		})

		r.mu.Lock()
		_, roomTaken := r.rooms[roomID]
		_, codeTaken := r.codes[codeKey(joinCode)]
		if !roomTaken && !codeTaken {
			r.rooms[roomID] = active
			r.codes[codeKey(joinCode)] = roomID
			r.mu.Unlock()
			return Identity{RoomID: roomID, JoinCode: joinCode, PlayerID: playerID, ReconnectToken: token}, nil
		}
		r.mu.Unlock()
		active.Close()
	}
}

func (r *Registry) Join(joinCode, playerName string) (Identity, error) {
	key := codeKey(joinCode)
	r.mu.RLock()
	roomID, ok := r.codes[key]
	active := r.rooms[roomID]
	r.mu.RUnlock()
	if !ok || active == nil {
		return Identity{}, ErrRoomNotFound
	}
	playerID := "player_" + randomHex(6)
	token := randomHex(24)
	if err := active.Join(playerID, strings.TrimSpace(playerName), token); err != nil {
		if errors.Is(err, ErrClosed) {
			return Identity{}, ErrRoomNotFound
		}
		return Identity{}, err
	}
	return Identity{RoomID: roomID, JoinCode: strings.ToUpper(strings.TrimSpace(joinCode)), PlayerID: playerID, ReconnectToken: token}, nil
}

func (r *Registry) Get(roomID string) (*Room, bool) {
	r.mu.RLock()
	active, ok := r.rooms[roomID]
	r.mu.RUnlock()
	return active, ok
}

func (r *Registry) Leave(roomID, playerID, token string) error {
	active, ok := r.Get(roomID)
	if !ok {
		return ErrRoomNotFound
	}
	return active.Leave(playerID, token)
}

func (r *Registry) Close() {
	r.mu.Lock()
	rooms := make([]*Room, 0, len(r.rooms))
	for _, active := range r.rooms {
		rooms = append(rooms, active)
	}
	r.rooms = make(map[string]*Room)
	r.codes = make(map[string]string)
	r.mu.Unlock()
	for _, active := range rooms {
		active.Close()
	}
}

func (r *Registry) remove(roomID, joinCode string, expected *Room) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if current, ok := r.rooms[roomID]; !ok || current != expected {
		return
	}
	delete(r.rooms, roomID)
	key := codeKey(joinCode)
	if r.codes[key] == roomID {
		delete(r.codes, key)
	}
}
