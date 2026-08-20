# Realtime protocol notes

The Go server is authoritative. The browser receives a player-specific `room.projection` and sends typed commands with an `expected_version`.

## Session lifecycle

1. Create or join returns a `RoomIdentity` containing room ID, player ID, join code, and reconnect token.
2. The identity is runtime validated before use and before reuse from local storage.
3. A WebSocket connects with room/player identifiers and authenticates with the reconnect token.
4. The UI is considered online only after `session.authenticated`.
5. Commands include a generated request ID and the projection version the client acted on.
6. Command acknowledgements clear only the matching pending request.

## Projection ordering and resync

The session tracks the latest projection version. Older projections are ignored. A version gap triggers a `room.resync` control frame instead of applying potentially incomplete state. The next projection completes the resync.

## Reconnection

Unexpected disconnects use capped exponential backoff with jitter and a five-minute offline grace period. Browser visibility/page lifecycle events can resume persisted sessions. A terminal session failure clears persisted credentials.

## Privacy

The server projection code is responsible for ensuring a player receives only their own private state. Frontend parsing protects type/runtime integrity but is not a security boundary against server data leakage.
