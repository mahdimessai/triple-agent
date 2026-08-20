# Triple Agent engineering guidance

Triple Agent is a realtime multiplayer game. The Go server is authoritative for room and game state; the Next.js frontend presents server projections and owns browser-side room lifecycle only.

## Invariants

- Never move game authority into the browser.
- Never expose one player's private projection to another player.
- Treat HTTP responses, WebSocket messages, persisted JSON, URLs, and browser storage as untrusted runtime data.
- Preserve the explicit `ClientCommand` and projection protocol rather than replacing it with loosely typed payloads.
- The live game client boundary is intentional. Do not try to convert the realtime game into a Server Component architecture.
- Prefer the repository's existing Bun, Next.js, React, TypeScript, Tailwind, and Go toolchain over introducing competing frameworks.
- Do not add repositories/services/factories or state libraries unless a concrete ownership or behavior problem requires them.

## Ownership

- `server/internal/game`: authoritative game rules and projections.
- `server/internal/room`: room lifecycle and concurrency.
- `server/internal/httpapi`: HTTP/WebSocket delivery.
- `web/app`: Next.js routing, metadata, manifest, global boundaries, and route composition.
- `web/features/triple-agent`: the realtime product frontend.
- `web/features/pwa`: install and retired-service-worker browser integration.

## Verification

Run `make verify` from the repository root before considering a change complete. It executes server tests plus the frontend typecheck, lint, tests, and production build.
