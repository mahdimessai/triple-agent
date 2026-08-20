# Triple Agent engineering instructions

Preserve the repository's deliberately small architecture. Do not introduce generic service/repository/use-case layers without a concrete second implementation or deployment requirement.

## Frontend

- `web/app/` owns Next.js routing, metadata, manifests, and route composition.
- `web/features/triple-agent/` owns the realtime game product.
- Keep the game-wide client boundary: the feature legitimately owns a persistent WebSocket, reconnect behavior, browser storage, and realtime interaction.
- Screens emit typed `ClientCommand` intentions. They must not instantiate WebSockets directly.
- Treat server JSON as `unknown` until `parseRoomServerMessage` validates it.
- Preserve exhaustive phase handling in `game.tsx`.
- Prefer extracting code for semantic ownership, not merely because a file is long.

## Backend

- `server/internal/httpapi` owns untrusted HTTP/WebSocket input and protocol translation.
- `server/internal/room` owns room lifecycle, sessions, timers, and concurrency.
- `server/internal/game` owns game rules and player projections.
- One room has one owner goroutine. Do not add locks around game state inside a room unless the ownership model itself changes.
- `game.State` must never be sent directly to clients. Only projections cross the network boundary.
- Public/private reveal timing is a server security invariant, not a frontend display concern.
- Never log reconnect tokens, private roles/factions, private operation results, or other secret game information.

## Change workflow

1. Read the closest README/doc.go and the nearest analogous code before editing.
2. Identify whether the change belongs to routing, frontend feature, network adapter, room runtime, or game rules.
3. Change the narrowest owning layer.
4. Add focused tests for behavioral or protocol invariants.
5. Run `make verify` before finishing when the local toolchain is available.
6. For concurrency changes, also run `make server-race`.
7. Report any check you could not run. Never claim an unrun check passed.

Do not replace Bun, Next.js, Gorilla WebSocket, or the current in-memory deployment model as incidental cleanup.
