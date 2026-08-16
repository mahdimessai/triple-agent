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
	ErrInvalidToken  = errors.New("invalid reconnect token")
)

type roomEntry struct {
	joinCode string
	tokens   map[string]string // playerID -> reconnectToken
}

// Store maps join codes to active rooms and tracks player reconnect credentials.
type Store struct {
	mu    sync.Mutex
	rooms map[string]*roomEntry // roomID -> roomEntry
	codes map[string]string     // UPPERCASE(joinCode) -> roomID
}

func NewStore() *Store {
	return &Store{
		rooms: make(map[string]*roomEntry),
		codes: make(map[string]string),
	}
}

// Reserve claims a join code for a new room and registers the host's credential.
func (s *Store) Reserve(roomID, joinCode, hostID, token string) error {
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
	s.rooms[roomID] = &roomEntry{
		joinCode: joinCode,
		tokens:   map[string]string{hostID: token},
	}
	return nil
}

// ResolveCode maps a user-typed join code to its room ID and canonical display code.
func (s *Store) ResolveCode(code string) (string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	roomID, ok := s.codes[codeKey(code)]
	if !ok {
		return "", "", ErrRoomNotFound
	}
	entry, ok := s.rooms[roomID]
	if !ok {
		delete(s.codes, codeKey(code))
		return "", "", ErrRoomNotFound
	}
	return roomID, entry.joinCode, nil
}

// Grant registers a reconnect credential for an admitted player.
func (s *Store) Grant(roomID, playerID, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.rooms[roomID]
	if !ok {
		return ErrRoomNotFound
	}
	entry.tokens[playerID] = token
	return nil
}

// Revoke removes a player's reconnect credential.
func (s *Store) Revoke(roomID, playerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.rooms[roomID]
	if !ok {
		return
	}
	delete(entry.tokens, playerID)
}

// ValidateReconnectToken checks if a reconnect token matches the registered player credential.
func (s *Store) ValidateReconnectToken(roomID, playerID, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.rooms[roomID]
	if !ok {
		return ErrRoomNotFound
	}
	expected, ok := entry.tokens[playerID]
	if !ok || expected != token {
		return ErrInvalidToken
	}
	return nil
}

// Release removes a room and frees its join code.
func (s *Store) Release(roomID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.rooms[roomID]
	if !ok {
		return
	}
	delete(s.codes, codeKey(entry.joinCode))
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
