# Triple Agent frontend feature

This directory owns the realtime Triple Agent product. Next.js `app/` owns routing and route composition only.

## Mental model

- `game.tsx` is the client-side feature entry point and phase router.
- `use-room.ts` owns room/session orchestration.
- `room.ts` owns HTTP/WebSocket transport mechanics.
- `protocol.ts` owns the runtime-validated wire contract.
- `operations.ts` and `roles.ts` own game presentation/catalog data.
- `screens/` owns phase-specific product UI.

The game is intentionally a client application because it owns a persistent WebSocket, reconnect behavior, browser storage, and realtime interaction. Do not move the whole route to the client merely because this feature is interactive; keep the client boundary at the feature entry point.

## Dependency rules

- Feature code must not import from `app/`.
- Routing code may compose this feature.
- Shared UI, when introduced under `components/`, must not import Triple Agent internals.
- Screens should emit typed `ClientCommand` intentions and must not create WebSockets directly.
- Untrusted server data must pass through `parseRoomServerMessage` before the application trusts it.
- Preserve exhaustive phase handling in `game.tsx`.

Prefer extracting a module when it creates a real ownership boundary, not merely to make a file shorter.
