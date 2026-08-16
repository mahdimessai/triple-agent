package room

import (
	"time"

	"tripleagent/server/internal/domain"
)

type roomTimers struct {
	// expiryTimer controls the room's overall lifetime.
	expiryTimer *time.Timer
	// expiryC is the receive-only channel used by the actor select loop.
	expiryC <-chan time.Time
	// discussionTimer advances a discussion or interlude at its deadline.
	discussionTimer *time.Timer
	// discussionC is nil outside phases that have an active discussion deadline.
	discussionC <-chan time.Time
}

func newRoomTimers(room *Room) *roomTimers {
	timers := &roomTimers{
		expiryTimer: time.NewTimer(room.lifetime),
	}
	timers.expiryC = timers.expiryTimer.C
	return timers
}

func (t *roomTimers) stopDiscussion() {
	if t.discussionTimer == nil {
		return
	}
	if !t.discussionTimer.Stop() {
		select {
		case <-t.discussionTimer.C:
		default:
		}
	}
	t.discussionTimer = nil
	t.discussionC = nil
}

func (t *roomTimers) resetDiscussion(state domain.GameState) {
	t.stopDiscussion()
	// Both the between-operations beat and the final discussion run on the same
	// deadline field, so one timer drives both.
	if state.DiscussionDeadline == nil || (state.Phase != domain.PhaseDiscussion && state.Phase != domain.PhaseOperationInterlude) {
		return
	}
	duration := time.Until(*state.DiscussionDeadline)
	if duration <= 0 {
		duration = time.Nanosecond
	}
	t.discussionTimer = time.NewTimer(duration)
	t.discussionC = t.discussionTimer.C
}

func (t *roomTimers) stopExpiry() {
	if !t.expiryTimer.Stop() {
		select {
		case <-t.expiryTimer.C:
		default:
		}
	}
}

func (t *roomTimers) resetExpiry(duration time.Duration) {
	if !t.expiryTimer.Stop() {
		select {
		case <-t.expiryTimer.C:
		default:
		}
	}
	t.expiryTimer.Reset(duration)
	t.expiryC = t.expiryTimer.C
}
