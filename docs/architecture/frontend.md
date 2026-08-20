# Frontend architecture

The frontend is a Next.js App Router application wrapped around a deliberately client-side realtime game.

```text
app
  -> features/triple-agent
       -> screens + game composition
       -> session
            -> transport
                 -> protocol
       -> operation/role presentation
```

## App Router

`web/app` owns URLs, metadata, the manifest, global CSS entry point, and route composition. It must not become the implementation home for Triple Agent.

`app/page.tsx` renders the feature public entry point. The game itself is client-side because it requires a persistent WebSocket, browser persistence, share/clipboard APIs, audio, and immediate local interaction.

## Protocol

`features/triple-agent/protocol` is the trust boundary for server data. TypeScript types document the wire format; runtime decoders validate unknown JSON before it becomes a trusted projection or session identity.

## Transport

`transport/room-api.ts` owns create/join/leave HTTP calls. `transport/room-socket.ts` owns socket authentication, command serialization, resync, and normalized connection events. Transport code has no React or screen knowledge.

## Session

`session/room-state.ts` is the pure reducer/state model. `room-storage.ts` owns persisted identity validation. `reconnect-policy.ts` owns retry timing. `use-room-session.ts` coordinates those pieces with browser and WebSocket lifecycle events.

## Screens

Screens consume trusted projections and emit typed `ClientCommand` intents. They do not create sockets or make raw backend requests. Game phase selection remains exhaustive in `game.tsx`.

## Adding work

- New server message shape: update server projection/transport, protocol types/decoder, and focused tests.
- New game phase: update server phase behavior and the exhaustive frontend phase switch.
- New screen-only behavior: keep it under the owning screen or a feature-local helper.
- New connection behavior: session if it is lifecycle policy, transport if it is HTTP/WebSocket mechanics.
- New globally routed page: `app/` composes a feature rather than absorbing the feature implementation.
