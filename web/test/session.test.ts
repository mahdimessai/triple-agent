import test from "node:test";
import assert from "node:assert/strict";
import { reconnectDelay, RECONNECT_GRACE_PERIOD_MS } from "../features/triple-agent/session/reconnect-policy";
import { INITIAL_ROOM_STATE, roomReducer } from "../features/triple-agent/session/room-state";

test("reconnect policy is capped and jitter stays bounded", () => {
  assert.equal(RECONNECT_GRACE_PERIOD_MS, 300_000);
  assert.equal(reconnectDelay(0, 0), 500);
  assert.equal(reconnectDelay(0, 0.999), 749);
  assert.equal(reconnectDelay(4, 0), 8_000);
  assert.equal(reconnectDelay(20, 0.999), 8_249);
});

test("room reducer only acknowledges the currently pending command", () => {
  const pendingState = roomReducer(INITIAL_ROOM_STATE, {
    type: "command-sent",
    pending: { requestId: "current", kind: "vote.submit" },
  });

  const staleAck = roomReducer(pendingState, {
    type: "command-acked",
    requestId: "stale",
  });
  assert.equal(staleAck.pending?.requestId, "current");

  const currentAck = roomReducer(staleAck, {
    type: "command-acked",
    requestId: "current",
  });
  assert.equal(currentAck.pending, null);
});

test("ending a session clears transient state but preserves the notice", () => {
  const busyState = {
    ...INITIAL_ROOM_STATE,
    status: "reconnecting" as const,
    error: "old error",
    pending: { requestId: "pending", kind: "vote.submit" as const },
  };

  const ended = roomReducer(busyState, {
    type: "session-ended",
    notice: { kind: "session-expired", message: "expired" },
  });

  assert.equal(ended.status, "idle");
  assert.equal(ended.pending, null);
  assert.equal(ended.error, null);
  assert.deepEqual(ended.notice, { kind: "session-expired", message: "expired" });
});
