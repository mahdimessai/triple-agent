package room

import (
	"time"

	"tripleagent/server/internal/domain"
)

func (r *Room) broadcast(state *domain.GameState, sessions map[string]roomSession) {
	r.broadcastExcept(state, sessions, "")
}

func (r *Room) broadcastExcept(state *domain.GameState, sessions map[string]roomSession, excluded string) {
	for {
		failed := r.deliver(*state, sessions, excluded)
		if len(failed) == 0 {
			return
		}
		changed := false
		for _, playerID := range failed {
			transition, err := domain.ApplyDisconnect(*state, playerID, time.Now().UTC())
			if err == nil && transition.Changed {
				*state = transition.State
				changed = true
			}
		}
		if !changed {
			return
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
