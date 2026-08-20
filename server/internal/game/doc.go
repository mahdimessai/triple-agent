// Package game is the authoritative Triple Agent rules engine.
//
// It deliberately knows nothing about HTTP, WebSockets, reconnect tokens, or
// goroutines. State transitions accept an existing State plus an explicit
// actor/command/time and return the next State. Tests can therefore control
// time and random state without network fixtures.
//
// Projection is also a security boundary. Internal State must never be sent to
// a player directly. Only Projection values may cross the network boundary,
// and public/private reveal rules must be enforced here rather than delegated
// to the frontend.
package game
