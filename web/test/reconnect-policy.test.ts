import test from "node:test";
import assert from "node:assert/strict";
import { reconnectDelay, RECONNECT_GRACE_PERIOD_MS } from "../features/triple-agent/session/reconnect-policy";

test("reconnect delay grows exponentially and caps before jitter", () => {
  assert.equal(reconnectDelay(0, 0), 500);
  assert.equal(reconnectDelay(1, 0), 1000);
  assert.equal(reconnectDelay(4, 0), 8000);
  assert.equal(reconnectDelay(20, 0), 8000);
});

test("reconnect jitter stays below 250ms and grace period is five minutes", () => {
  assert.equal(reconnectDelay(0, 0.999), 749);
  assert.equal(RECONNECT_GRACE_PERIOD_MS, 300_000);
});
