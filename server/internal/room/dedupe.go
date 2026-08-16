package room

import "tripleagent/server/internal/domain"

const maxDedupeEntries = 1024

type dedupeEntry struct {
	// actorID binds a request ID to its original submitter.
	actorID string
	// projection is the successful result replayed to a retried request.
	projection domain.Projection
	// err is the original failure replayed to a retried request.
	err error
}

func rememberDedupe(entries map[string]dedupeEntry, order []string, requestID string, entry dedupeEntry) (map[string]dedupeEntry, []string) {
	if _, exists := entries[requestID]; !exists {
		order = append(order, requestID)
	}
	entries[requestID] = entry
	if len(order) > maxDedupeEntries {
		oldest := order[0]
		delete(entries, oldest)
		order = order[1:]
	}
	return entries, order
}
