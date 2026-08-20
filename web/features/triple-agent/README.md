# Triple Agent frontend feature

This directory owns the realtime Triple Agent product. Next.js `app/` owns routing and route composition only.

## Mental model

- `game.tsx` is the client-side feature entry point and phase router.
- `use-room.ts` is the room/session orchestration hook.
- `session/room-state.ts` owns the pure reducer and UI-facing session state model.
- `session/reconnect-policy.ts` owns reconnect timing policy.
- `session/room-storage.ts` owns validated persisted room identity access.
- `room.ts` is the small public transport facade used by session orchestration.
- `transport/room-api.ts` owns HTTP create/join/leave mechanics.
- `transport/room-socket.ts` owns WebSocket authentication, messages, command serialization, and transport events.
- `transport/endpoints.ts` owns HTTP/WebSocket endpoint construction.
- `protocol.ts` owns the runtime-validated wire contract.
- `invite/` owns invite parsing, URL generation, clipboard/share behavior, and its UI feedback state.
- `operations.ts` and `roles.ts` own game presentation/catalog data.
- `screens/` owns phase-specific product UI.
- `ui.tsx` keeps Triple Agent-specific art composition while re-exporting domain-light shared primitives from `components/ui`.

The game is intentionally a client application because it owns a persistent WebSocket, reconnect behavior, browser storage, and realtime interaction. Do not move the whole route to the client merely because this feature is interactive; keep the client boundary at the feature entry point.

## Dependency rules

- Feature code must not import from `app/`.
- Routing code may compose this feature.
- Shared UI under `components/` must not import Triple Agent internals.
- Screens should emit typed `ClientCommand` intentions and must not create WebSockets directly.
- `use-room.ts` orchestrates transport and session policies rather than reimplementing their pure rules inline.
- Untrusted server data must pass through `parseRoomServerMessage` before the application trusts it.
- Persisted room identities must pass `isRoomIdentity` before reuse.
- Preserve exhaustive phase handling in `game.tsx`.

Prefer extracting a module when it creates a real ownership boundary, not merely to make a file shorter.
