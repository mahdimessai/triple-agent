# Triple Agent

A multiplayer PWA reimplementation of Triple Agent with a Go authoritative game server and a Next.js realtime client.

## Requirements

- Go
- Bun
- Docker and Docker Compose, optional

## Local development

```sh
make dev
```

The Go server and Next.js app run together. The frontend backend URL can be configured with `NEXT_PUBLIC_TRIPLE_AGENT_API_URL`; local development defaults to `http://localhost:8080`.

## Verification

```sh
make verify
```

This runs all Go tests and the frontend `typecheck`, `lint`, Bun tests, and production Next.js build.

## Architecture

- `server/internal/game`: authoritative game rules and player-specific projections
- `server/internal/room`: room/session concurrency and lifecycle
- `server/internal/httpapi`: HTTP and WebSocket delivery
- `web/app`: Next.js routing and platform integration
- `web/features/triple-agent`: realtime frontend feature
- `web/features/pwa`: install and service-worker migration behavior

Read `docs/architecture/frontend.md` and `docs/architecture/protocol.md` before making cross-cutting frontend or protocol changes. Coding-agent guidance lives in `AGENTS.md` and `web/AGENTS.md`.

## Docker

```sh
make docker-dev
```

For the production-style compose setup:

```sh
make docker
```
