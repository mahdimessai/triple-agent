// Package room owns realtime room lifecycle and concurrency.
//
// Each Room runs exactly one goroutine. That goroutine is the sole owner of
// the room's mutable game state, reconnect credentials, and active sessions.
// Public Room methods send synchronous requests into that goroutine; callers
// never mutate a room runtime directly.
//
// Important invariants:
//   - a player has at most one active session;
//   - attaching a replacement session closes the previous one;
//   - session IDs protect a replacement connection from stale Detach calls;
//   - commands require both the current session and the current state version;
//   - failed or slow senders are disconnected instead of blocking the room;
//   - package room hosts game.State but does not implement game rules.
package room
