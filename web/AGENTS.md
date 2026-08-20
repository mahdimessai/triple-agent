# Frontend architecture

Keep `app/` thin. Product behavior belongs under `features/`.

## Triple Agent dependency direction

- `protocol/` describes and validates the wire contract. It imports no React, session, transport, or screens.
- `transport/` owns HTTP and WebSocket mechanics. It may import protocol code but no React or screens.
- `session/` owns reducer state, persistence, reconnect policy, and React lifecycle orchestration. It may import protocol and transport but not screens.
- `screens/` render trusted projections and emit typed user intent. Screens must not create sockets or perform raw HTTP calls.
- `game.tsx` is the client composition root. Keep phase selection and cross-screen composition obvious here.
- Feature-local UI primitives may know the Triple Agent visual language, but they must not know transport details.
- `index.ts` is a deliberate public entry point, not a barrel dump.

## React

Effects are appropriate for WebSockets, browser lifecycle events, storage, audio, focus, and other external systems. Do not use effects to derive ordinary render values.

Do not add `memo`, `useMemo`, or `useCallback` automatically. Use them only for a concrete identity or computation reason.

Keep state near the interaction that owns it. Feature-level orchestration state belongs in the game/session boundary; local form and modal state belongs with that UI.

## TypeScript and boundaries

Use `unknown` at untrusted boundaries and narrow before the value enters trusted application code. Do not add `any`, `@ts-ignore`, or unchecked casts to silence errors.

Preserve exhaustive discriminated unions for phases, commands, connection events, and reducer actions.

## Checks

From `web/`, run `bun run verify`. From the repository root, prefer `make verify`.
