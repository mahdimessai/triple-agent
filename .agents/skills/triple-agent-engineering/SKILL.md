---
name: triple-agent-engineering
description: Implement, refactor, and review Triple Agent across its Next.js frontend and Go realtime backend while preserving explicit ownership, strict wire boundaries, room actor concurrency, projection secrecy, exhaustive game phases, and repository-native verification. Use for substantial changes in this repository.
---

# Triple Agent engineering

Optimize for time-to-understanding. A new engineer should be able to identify the owning layer before understanding every implementation detail.

## Read first

For backend work, read `server/README.md`, then the relevant package `doc.go`.
For frontend work, read `web/features/triple-agent/README.md` and `web/test/architecture.test.mjs`.
Repository-specific instructions in `AGENTS.md` override generic preferences.

## Architecture

```text
frontend
app -> features/triple-agent

backend
cmd -> httpapi -> room -> game
               \--------> game projections/types
```

### Frontend ownership

- `web/app`: routing and global Next.js concerns.
- `web/features/triple-agent`: realtime game application.
- Keep the game feature client-side where browser/realtime capabilities require it.
- Protocol/runtime validation belongs at the feature's wire boundary.
- Screens render state and emit typed intents. They do not own transport.

### Backend ownership

- `httpapi`: untrusted network bytes, browser origin policy, authentication handshake, wire/domain translation, stable protocol errors.
- `room`: registry, sessions, lifecycle, timers, optimistic version gate, single-goroutine runtime ownership, broadcasting.
- `game`: rules, state transitions, operations, roles, voting/results, projections.

Do not add repository/service/use-case layers to make the diagram look more sophisticated.

## Critical invariants

### Room actor

- One goroutine is the sole owner of a room runtime.
- A player has at most one current session.
- New Attach closes an old session.
- Session IDs make stale Detach harmless.
- Commands require current session and expected state version.
- Failed/slow senders are removed rather than blocking the game.

### Game

- Game rules do not import HTTP/WebSocket packages.
- A failed command must not partially mutate authoritative state.
- Versions advance only for observable state transitions.
- Keep command handling discoverable from `game.Apply`.

### Information boundary

- Never serialize `game.State` directly.
- Only `game.Projection` crosses the backend network boundary.
- A player's private projection contains only that player's secrets.
- Staged results must be withheld until their explicit reveal phase.
- Frontend hiding is not a security boundary.

## Wire protocol

Treat HTTP and WebSocket input as untrusted runtime data.

- Reject structurally invalid JSON.
- Prefer strict unknown-field handling for protocol objects.
- Keep protocol error codes stable and non-sensitive.
- Log unexpected server errors with safe identifiers, but never secrets.
- ACK and projection ordering is not a global guarantee. Projection version is authoritative.

## Frontend implementation rules

- Preserve the explicit `features/triple-agent` home.
- Do not move the implementation back under `app/`.
- Reuse local semantic components before inventing generic shared abstractions.
- Keep derived values derived instead of mirroring them into React state.
- Do not add `useMemo`, `useCallback`, or `memo` by ritual.
- Keep exhaustive phase switches and `assertNever` checks.
- Validate persisted identities before reuse and server messages before trusting them.

## Go implementation rules

- Prefer concrete types and package-local helpers over speculative interfaces.
- Keep package names `game`, `room`, and `httpapi` unless the responsibility actually changes.
- Split large files inside the same package when it improves navigation without exporting internals.
- Use `log/slog` for structured operational logging.
- Keep graceful shutdown aware that WebSockets are hijacked connections and must be closed through room/session ownership.
- Prefer domain-specific types/constants when raw strings become an error-prone vocabulary, but migrate them in focused changes rather than one giant rewrite.

## Testing by risk

For game-rule changes:
- focused pure state-transition tests;
- projection secrecy/reveal tests when information changes;
- deterministic seed/time when randomness or deadlines matter.

For room changes:
- stale session/version tests;
- replacement session behavior;
- failed sender convergence;
- expiry/close behavior;
- race detector.

For protocol changes:
- malformed/unknown JSON;
- real WebSocket integration when handshake/ordering changes;
- frontend parser contract where wire shape changes.

## Verification

Use repository commands:

```text
make verify
make server-race      # concurrency-sensitive backend changes
make vulncheck        # dependency/security changes or CI
```

`make verify` is the completion gate. Do not claim it passed unless it was run.

## Review checklist

Before finishing a substantial change, ask:

- Is the owning layer obvious?
- Did routing stay thin?
- Did game rules stay network-independent?
- Did room concurrency stay single-owner?
- Did untrusted data cross an explicit validation boundary?
- Could this change leak private game information earlier than intended?
- Are stable versions/session IDs still the synchronization contract?
- Did a new abstraction solve a real ownership or correctness problem?
- Are names domain-specific and searchable?
- Are the relevant tests smaller than the failure they protect?
- Were verification commands actually run and reported accurately?
