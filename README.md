# Triple Agent

A realtime social-deduction game with a Next.js/TypeScript client and an in-memory Go WebSocket server.

## Start here

Requirements:

- Bun
- Go 1.25+

Run both applications:

```sh
make dev
```

- Web: `http://localhost:3000`
- Server: `http://localhost:8080`
- Health: `http://localhost:8080/healthz`

Run the complete local verification gate:

```sh
make verify
```

## Architecture

```text
web/app/                         Next.js routing and global app concerns
web/features/triple-agent/       realtime frontend product

server/internal/httpapi/         HTTP/WebSocket boundary
server/internal/room/            room registry, sessions, concurrency
server/internal/game/            pure-ish game rules and projections
```

The backend has one especially important rule: every live room is owned by one goroutine. The `game` package contains the rules; the `room` package serializes realtime access to them; `httpapi` translates untrusted network input into room/game operations.

The frontend has the analogous ownership rule: `app/` composes routes, while the Triple Agent implementation lives under `features/triple-agent/`.

See [`server/README.md`](server/README.md) for the backend reading path and invariants, and [`web/features/triple-agent/README.md`](web/features/triple-agent/README.md) for the frontend feature boundary.

## Useful commands

```sh
make dev           # run server and web dev servers
make server        # Go server only
make web           # Next.js only
make test          # backend + frontend tests
make server-vet    # go vet
make server-race   # Go tests with race detector
make verify        # formatting check, vet, Go tests, frontend verify
make vulncheck     # govulncheck when installed/network is available
make docker-dev    # development Docker Compose
make docker        # production-style Docker Compose
```

## Deployment note

The server currently stores rooms and game state in memory and assumes one authoritative server process. Restarting the backend ends active rooms. Horizontal scaling requires explicit room ownership/routing or shared coordination before adding replicas.
