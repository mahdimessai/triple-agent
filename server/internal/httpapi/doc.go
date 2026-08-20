// Package httpapi owns Triple Agent's untrusted network boundary.
//
// It validates HTTP and WebSocket payloads, authenticates room sessions,
// translates wire commands into game commands, and maps domain errors to
// stable protocol responses. It must not implement game rules.
//
// Browser origin policy is shared by HTTP CORS and the WebSocket upgrader so
// there is one answer to which web clients may talk to the server.
package httpapi
