import test from "node:test";
import assert from "node:assert/strict";
import { INITIAL_ROOM_STATE, roomReducer } from "../features/triple-agent/session/room-state";

const identity = { room_id: "r", join_code: "ABC123", player_id: "p", reconnect_token: "token" };

test("room reducer models request and connection lifecycle explicitly", () => {
  const connecting = roomReducer(INITIAL_ROOM_STATE, { type: "request-started" });
  assert.equal(connecting.status, "connecting");
  const opening = roomReducer(connecting, { type: "connect-started", identity, reconnecting: false });
  assert.equal(opening.identity, identity);
  assert.equal(opening.status, "connecting");
  const online = roomReducer(opening, { type: "connected" });
  assert.equal(online.status, "online");
  const reconnecting = roomReducer(online, { type: "connection-lost" });
  assert.equal(reconnecting.status, "reconnecting");
});

test("only the matching command acknowledgement clears pending state", () => {
  const pending = roomReducer(INITIAL_ROOM_STATE, {
    type: "command-sent",
    pending: { requestId: "request-1", kind: "lobby.ready" },
  });
  assert.equal(roomReducer(pending, { type: "command-acked", requestId: "other" }), pending);
  assert.equal(roomReducer(pending, { type: "command-acked", requestId: "request-1" }).pending, null);
});

test("ending a session clears private state but preserves the user-facing notice", () => {
  const active = { ...INITIAL_ROOM_STATE, identity, status: "online" as const, error: "old error" };
  const ended = roomReducer(active, { type: "session-ended", notice: { kind: "session-expired", message: "expired" } });
  assert.equal(ended.identity, null);
  assert.equal(ended.status, "idle");
  assert.deepEqual(ended.notice, { kind: "session-expired", message: "expired" });
  assert.equal(ended.error, null);
});
