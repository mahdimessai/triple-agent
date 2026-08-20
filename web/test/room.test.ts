import test from "node:test";
import assert from "node:assert/strict";
import { connectRoom, type RoomSocketEvent } from "../features/triple-agent/room";
import type { RoomIdentity } from "../features/triple-agent/protocol";

type Listener = (event: { data?: unknown }) => void;

class FakeWebSocket {
  static OPEN = 1;
  static instances: FakeWebSocket[] = [];
  readyState = FakeWebSocket.OPEN;
  sent: string[] = [];
  listeners = new Map<string, Listener[]>();
  closed = false;
  constructor(readonly url: string) { FakeWebSocket.instances.push(this); }
  addEventListener(type: string, listener: Listener) { this.listeners.set(type, [...(this.listeners.get(type) ?? []), listener]); }
  send(value: string) { this.sent.push(value); }
  close() { this.closed = true; this.emit("close"); }
  emit(type: string, event: { data?: unknown } = {}) { for (const listener of this.listeners.get(type) ?? []) listener(event); }
}

const identity: RoomIdentity = { room_id: "room-1", join_code: "ABC123", player_id: "player-1", reconnect_token: "secret" };

function setup() {
  FakeWebSocket.instances = [];
  Object.assign(globalThis, { WebSocket: FakeWebSocket });
  const events: RoomSocketEvent[] = [];
  const room = connectRoom(identity, (event) => events.push(event));
  return { room, events, socket: FakeWebSocket.instances[0]! };
}

test("authenticates first and opens only after server confirmation", () => {
  const { events, socket } = setup();
  assert.match(socket.url, /^ws:\/\/localhost:8080\/ws\?/);
  socket.emit("open");
  assert.deepEqual(JSON.parse(socket.sent[0]!), { type: "room.auth", reconnect_token: "secret" });
  assert.equal(events.length, 0);
  socket.emit("message", { data: JSON.stringify({ type: "session.authenticated" }) });
  assert.deepEqual(events, [{ type: "open" }]);
});

test("serializes typed commands with request id and expected version", () => {
  const { room, socket } = setup();
  const requestId = room.send({ kind: "vote.submit", target_id: "p2" }, 17);
  const message = JSON.parse(socket.sent[0]!);
  assert.equal(message.kind, "vote.submit");
  assert.equal(message.target_id, "p2");
  assert.equal(message.expected_version, 17);
  assert.equal(message.request_id, requestId);
  assert.equal(typeof requestId, "string");
});

test("sends resync as a transport control frame", () => {
  const { room, socket } = setup();
  room.resync();
  assert.deepEqual(JSON.parse(socket.sent[0]!), { kind: "room.resync" });
});

test("reports malformed server data without crashing", () => {
  const { events, socket } = setup();
  socket.emit("message", { data: "not json" });
  socket.emit("message", { data: JSON.stringify({ type: "command.ack", ok: true }) });
  assert.deepEqual(events, [{ type: "invalid-message" }, { type: "invalid-message" }]);
});

test("emits one logical close when error and close both fire", () => {
  const { events, socket } = setup();
  socket.emit("error");
  socket.emit("close");
  assert.deepEqual(events, [{ type: "closed", terminal: false, message: "The room connection failed" }]);
  assert.equal(socket.closed, true);
});

test("marks authentication expiry as terminal and closes the socket once", () => {
  const { events, socket } = setup();
  socket.emit("message", { data: JSON.stringify({ type: "session.error", status: 410, error: "expired" }) });
  assert.equal(events[0]?.type, "message");
  assert.deepEqual(events[1], { type: "closed", terminal: true, status: 410, message: "expired" });
  assert.equal(events.filter((event) => event.type === "closed").length, 1);
  assert.equal(socket.closed, true);
});
