package room

import (
	"time"

	"tripleagent/server/internal/domain"
)

// broadcast returns true when delivery failures release the final lobby seat.
func (r *Room) broadcast(runtime *runtimeState) bool {
	return r.broadcastExcept(runtime, "")
}

func (r *Room) broadcastExcept(runtime *runtimeState, excluded string) bool {
	for {
		failed := r.deliver(runtime.game, runtime.sessions, excluded)
		if len(failed) == 0 {
			return len(runtime.game.PlayerOrder) == 0
		}

		changed := false
		for _, playerID := range failed {
			transition, err := domain.ApplyDisconnect(runtime.game, playerID, time.Now().UTC())
			if err == nil && transition.Changed {
				runtime.game = transition.State
				pruneCredentials(runtime.credentials, runtime.game)
				changed = true
			}
		}
		if len(runtime.game.PlayerOrder) == 0 {
			return true
		}
		if !changed {
			return false
		}

		// Presence or host-transfer changes caused by a failed delivery must be
		// observed by every remaining session before the actor processes more work.
		excluded = ""
	}
}

func (r *Room) deliver(state domain.GameState, sessions map[string]roomSession, excluded string) []string {
	failed := make([]string, 0)
	public := domain.PublicProjectionFor(state)
	for playerID, session := range sessions {
		if playerID == excluded {
			continue
		}
		if err := session.sender(domain.ProjectWithPublic(state, playerID, public)); err != nil {
			delete(sessions, playerID)
			if session.close != nil {
				session.close()
			}
			failed = append(failed, playerID)
		}
	}
	return failed
}
