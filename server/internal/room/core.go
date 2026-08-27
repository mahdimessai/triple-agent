package room

import (
	"time"

	"tripleagent/server/internal/game"
)

// RoomCore encapsulates the pure, synchronous state container of a room.
// It contains zero concurrency primitives and can be 100% unit-tested deterministically.
type RoomCore struct {
	RoomID   string
	State    game.State
	Tokens   *TokenDirectory
	Sessions map[string]ClientSession
}

func NewRoomCore(roomID string, state game.State, tokens *TokenDirectory) *RoomCore {
	if tokens == nil {
		tokens = NewTokenDirectory()
	}
	return &RoomCore{
		RoomID:   roomID,
		State:    state,
		Tokens:   tokens,
		Sessions: make(map[string]ClientSession),
	}
}

func (c *RoomCore) HandleJoin(playerID, name, token string) error {
	if token == "" {
		return ErrUnauthorized
	}
	next, err := game.AddPlayer(c.State, playerID, name)
	if err != nil {
		return err
	}
	c.State = next
	c.Tokens.Register(playerID, token)
	return nil
}

func (c *RoomCore) HandleLeave(playerID, token string) error {
	if !c.Tokens.Authorize(playerID, token) {
		return ErrUnauthorized
	}
	next, err := game.Leave(c.State, playerID)
	if err != nil {
		return err
	}
	c.State = next
	c.Tokens.Remove(playerID)
	c.CloseSession(playerID)
	return nil
}

func (c *RoomCore) HandleAttach(token string, session ClientSession) (string, game.Projection, error) {
	if token == "" || session == nil {
		return "", game.Projection{}, ErrUnauthorized
	}
	playerID, ok := c.Tokens.PlayerID(token)
	if !ok {
		return "", game.Projection{}, ErrUnauthorized
	}
	if !game.HasPlayer(c.State, playerID) {
		return "", game.Projection{}, game.ErrPlayerNotInRoom
	}

	if _, occupied := c.Sessions[playerID]; occupied {
		return "", game.Projection{}, ErrSessionActive
	}
	next, err := game.Connect(c.State, playerID)
	if err != nil {
		return "", game.Projection{}, err
	}

	c.State = next
	c.Sessions[playerID] = session
	projection := game.Project(c.RoomID, c.State, playerID)
	return playerID, projection, nil
}

func (c *RoomCore) HandleDetach(playerID, sessionID string, now time.Time) (bool, error) {
	current, exists := c.Sessions[playerID]
	if !exists || current.ID() != sessionID {
		return false, nil
	}
	delete(c.Sessions, playerID)
	next, err := game.Disconnect(c.State, playerID, now)
	if err != nil {
		return false, err
	}
	changed := next.Version != c.State.Version
	c.State = next
	if !game.HasPlayer(c.State, playerID) {
		c.Tokens.Remove(playerID)
	}
	return changed, nil
}

// HandleReleaseSession detaches a player's live session identified only by
// their reconnect token, e.g. when a page-hide beacon frees the seat ahead of a
// refresh or tab close. It is a no-op when no session is live. Unlike a normal
// detach — where the dying socket owns its own teardown — the released
// transport is still healthy, so it is closed here.
func (c *RoomCore) HandleReleaseSession(token string, now time.Time) (bool, error) {
	playerID, ok := c.Tokens.PlayerID(token)
	if !ok {
		return false, ErrUnauthorized
	}
	current, exists := c.Sessions[playerID]
	if !exists {
		return false, nil
	}
	changed, err := c.HandleDetach(playerID, current.ID(), now)
	if err != nil {
		return changed, err
	}
	current.Close()
	return changed, nil
}

func (c *RoomCore) HandleGameCommand(playerID, sessionID string, expectedVersion uint64, cmd game.Command, now time.Time) error {
	current, exists := c.Sessions[playerID]
	if !exists || current.ID() != sessionID {
		return ErrSessionGone
	}
	if expectedVersion != c.State.Version {
		return ErrVersionConflict
	}
	before := c.State
	next, err := game.Apply(before, playerID, cmd, now)
	if err != nil {
		return err
	}
	if next.Version == before.Version {
		return nil
	}
	c.State = next
	c.cleanupRemovedPlayers()
	return nil
}

func (c *RoomCore) Snapshot(playerID string) (game.Projection, error) {
	if !game.HasPlayer(c.State, playerID) {
		return game.Projection{}, game.ErrPlayerNotInRoom
	}
	return game.Project(c.RoomID, c.State, playerID), nil
}

func (c *RoomCore) CloseSession(playerID string) {
	current, ok := c.Sessions[playerID]
	if !ok {
		return
	}
	delete(c.Sessions, playerID)
	current.Close()
}

func (c *RoomCore) CloseAllSessions() {
	for id, session := range c.Sessions {
		delete(c.Sessions, id)
		session.Close()
	}
}

func (c *RoomCore) EvictFailedSessions(failedPlayerIDs []string, now time.Time) bool {
	changed := false
	for _, playerID := range failedPlayerIDs {
		c.CloseSession(playerID)
		before := c.State.Version
		next, err := game.Disconnect(c.State, playerID, now)
		if err != nil {
			continue
		}
		c.State = next
		if next.Version != before {
			changed = true
		}
		if !game.HasPlayer(c.State, playerID) {
			c.Tokens.Remove(playerID)
		}
	}
	return changed
}

func (c *RoomCore) IsEmpty() bool {
	return game.Empty(c.State)
}

func (c *RoomCore) cleanupRemovedPlayers() {
	for playerID := range c.Tokens.byPlayer {
		if !game.HasPlayer(c.State, playerID) {
			c.Tokens.Remove(playerID)
			c.CloseSession(playerID)
		}
	}
}
