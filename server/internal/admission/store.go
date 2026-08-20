package admission

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
)

var (
	ErrRoomNotFound  = errors.New("room not found")
	ErrJoinCodeTaken = errors.New("join code is already in use")
)

// Store owns the cross-room admission index. Live player credentials belong to
// the room actor because their lifetime is exactly the lifetime of a seat.
type Store struct {
	mu    sync.RWMutex
	rooms map[string]string // roomID -> canonical join code
	codes map[string]string // UPPERCASE(joinCode) -> roomID
}

func NewStore() *Store {
	return &Store{
		rooms: make(map[string]string),
		codes: make(map[string]string),
	}
}

// Reserve claims a join code for a live room.
func (s *Store) Reserve(roomID, joinCode string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	code := codeKey(joinCode)
	if _, exists := s.codes[code]; exists {
		return ErrJoinCodeTaken
	}
	if _, exists := s.rooms[roomID]; exists {
		return ErrJoinCodeTaken
	}

	s.codes[code] = roomID
	s.rooms[roomID] = joinCode
	return nil
}

// ResolveCode maps a user-typed join code to its room ID and canonical display code.
func (s *Store) ResolveCode(code string) (string, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	roomID, ok := s.codes[codeKey(code)]
	if !ok {
		return "", "", ErrRoomNotFound
	}
	joinCode, ok := s.rooms[roomID]
	if !ok {
		return "", "", ErrRoomNotFound
	}
	return roomID, joinCode, nil
}

// Release removes a room and frees its join code.
func (s *Store) Release(roomID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	joinCode, ok := s.rooms[roomID]
	if !ok {
		return
	}
	delete(s.codes, codeKey(joinCode))
	delete(s.rooms, roomID)
}

func NewRoomID() string {
	return "room_" + randomHex(6)
}

func NewPlayerID() string {
	return "player_" + randomHex(6)
}

func NewReconnectToken() string {
	return randomHex(24)
}

func NewJoinCode() string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	for i := range bytes {
		bytes[i] = alphabet[int(bytes[i])%len(alphabet)]
	}
	return string(bytes)
}

func randomHex(size int) string {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

func codeKey(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}
