# Triple Agent server

The backend is intentionally small. Its architecture is three concepts, not a generic service-layer framework:

```text
cmd/tripleagent
      |
      v
   httpapi
      |
      v
     room
      |
      v
     game
```

## Five-minute mental model

1. **`httpapi` owns the network boundary.** HTTP and WebSocket bytes are untrusted until decoded and validated there.
2. **`room` owns realtime lifecycle and concurrency.** Every room has exactly one goroutine, and that goroutine is the sole owner of the room's mutable runtime.
3. **`game` owns game rules.** It has no HTTP/WebSocket knowledge. Commands produce a next `game.State`.
4. **`game.Projection` is the secrecy boundary.** Never serialize or send the internal game state to clients. Public/private reveal rules belong on the server.
5. **Room state is in memory.** A process restart destroys active rooms. The current deployment model assumes one authoritative server process.

## Critical request path

Creating a room:

```text
POST /api/lobbies
  -> httpapi.createLobby
  -> room.Registry.Create
  -> crypto/rand game seed
  -> game.NewLobbyWithSeed
  -> room actor starts
```

Opening the realtime session:

```text
GET /ws
  -> WebSocket upgrade
  -> room.auth
  -> Registry.Get
  -> Room.Attach
  -> game.Connect
  -> game.Project
```

Applying a player command:

```text
WebSocket command
  -> strict wire decode
  -> Room.Command
  -> session/version checks
  -> game.Apply
  -> new versioned State
  -> per-player game.Project
  -> room broadcast
```

## Where to start reading

| Order | File | Purpose |
| --- | --- | --- |
| 1 | `cmd/tripleagent/main.go` | process wiring, configuration, graceful shutdown |
| 2 | `internal/httpapi/server.go` | public HTTP/WS surface and error mapping |
| 3 | `internal/httpapi/websocket.go` | realtime protocol and room-session bridge |
| 4 | `internal/httpapi/connection.go` | single-writer WebSocket loop and backpressure |
| 5 | `internal/room/doc.go` + `room.go` | single-goroutine room actor |
| 6 | `internal/game/doc.go` + `state.go` | authoritative game model |
| 7 | `internal/game/apply.go` | command/state-machine surface |
| 8 | `internal/game/projection.go` | public/private information boundary |
| 9 | `internal/game/operations.go` | operation catalog and mechanics |
| 10 | `internal/game/roles.go`, `results.go` | roles, voting, outcomes |
| 11 | tests | executable contracts and examples |

## Core invariants

### Room/runtime

- Only the room goroutine mutates its runtime.
- A player has at most one active WebSocket session.
- Replacing a session closes the old one.
- A stale `Detach` must not disconnect a replacement session.
- Commands require the current session ID and expected game version.
- A slow or failed writer is disconnected instead of blocking the room.

### Game

- `game` must not import `net/http`, Gorilla WebSocket, `room`, or `httpapi`.
- Failed transitions must not partially mutate the authoritative state.
- State versions only advance for observable state changes.
- Randomness lives in game state so tests can use deterministic seeds.
- Production room creation seeds that deterministic PRNG from `crypto/rand`, not the clock.

### Information security

- Internal `game.State` never crosses the network.
- `PublicProjection` must not contain another player's secret state.
- Results are disclosed cumulatively only at their explicit reveal phases.
- Reconnect tokens and private role/operation data must not be logged.

## Wire ordering

Command acknowledgements and updated projections may arrive in either order. The room actor can enqueue a projection while the WebSocket handler independently enqueues the acknowledgement after `Room.Command` returns. Clients must use projection `version` as the authoritative ordering signal and ignore stale projections rather than assuming ACK/projection ordering.

## Configuration

- `TRIPLE_AGENT_ADDR`: listen address, default `:8080`.
- `TRIPLE_AGENT_ALLOWED_ORIGINS`: comma-separated browser origins. Local development defaults to `http://localhost:3000,http://127.0.0.1:3000`.

The same allowed-origin policy is used for HTTP CORS and WebSocket origin checks.

## Development

From the repository root:

```sh
make server
make server-test
make server-vet
make verify
```

CI additionally runs the race detector. `govulncheck` is included as a dedicated target because it may require the Go vulnerability database/network access.

## Deployment model

Triple Agent currently assumes **one authoritative backend process**. `room.Registry` stores live rooms in memory. Horizontal replicas would require an explicit room-ownership/routing strategy or shared coordination/persistence so reconnects reach the process that owns the room.

That constraint is intentional today. Do not add a repository/database abstraction until the deployment model actually changes.
