"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const node_test_1 = __importDefault(require("node:test"));
const strict_1 = __importDefault(require("node:assert/strict"));
const room_1 = require("../app/_game/room");
class FakeWebSocket {
    url;
    static OPEN = 1;
    static instances = [];
    readyState = FakeWebSocket.OPEN;
    sent = [];
    listeners = new Map();
    closed = false;
    constructor(url) {
        this.url = url;
        FakeWebSocket.instances.push(this);
    }
    addEventListener(type, listener) { this.listeners.set(type, [...(this.listeners.get(type) ?? []), listener]); }
    send(value) { this.sent.push(value); }
    close() { this.closed = true; this.emit("close"); }
    emit(type, event = {}) { for (const listener of this.listeners.get(type) ?? [])
        listener(event); }
}
const identity = { room_id: "room-1", join_code: "ABC123", player_id: "player-1", reconnect_token: "secret" };
function setup() {
    FakeWebSocket.instances = [];
    Object.assign(globalThis, { WebSocket: FakeWebSocket });
    const events = [];
    const room = (0, room_1.connectRoom)(identity, (event) => events.push(event));
    return { room, events, socket: FakeWebSocket.instances[0] };
}
(0, node_test_1.default)("authenticates first and opens only after server confirmation", () => {
    const { events, socket } = setup();
    strict_1.default.match(socket.url, /^ws:\/\/localhost:8080\/ws\?/);
    socket.emit("open");
    strict_1.default.deepEqual(JSON.parse(socket.sent[0]), { type: "room.auth", reconnect_token: "secret" });
    strict_1.default.equal(events.length, 0);
    socket.emit("message", { data: JSON.stringify({ type: "session.authenticated" }) });
    strict_1.default.deepEqual(events, [{ type: "open" }]);
});
(0, node_test_1.default)("serializes typed commands with request id and expected version", () => {
    const { room, socket } = setup();
    const requestId = room.send({ kind: "vote.submit", target_id: "p2" }, 17);
    const message = JSON.parse(socket.sent[0]);
    strict_1.default.equal(message.kind, "vote.submit");
    strict_1.default.equal(message.target_id, "p2");
    strict_1.default.equal(message.expected_version, 17);
    strict_1.default.equal(message.request_id, requestId);
    strict_1.default.equal(typeof requestId, "string");
});
(0, node_test_1.default)("sends resync as a transport control frame", () => {
    const { room, socket } = setup();
    room.resync();
    strict_1.default.deepEqual(JSON.parse(socket.sent[0]), { kind: "room.resync" });
});
(0, node_test_1.default)("reports malformed server data without crashing", () => {
    const { events, socket } = setup();
    socket.emit("message", { data: "not json" });
    socket.emit("message", { data: JSON.stringify({ type: "command.ack", ok: true }) });
    strict_1.default.deepEqual(events, [{ type: "invalid-message" }, { type: "invalid-message" }]);
});
(0, node_test_1.default)("emits one logical close when error and close both fire", () => {
    const { events, socket } = setup();
    socket.emit("error");
    socket.emit("close");
    strict_1.default.deepEqual(events, [{ type: "closed", terminal: false, message: "The room connection failed" }]);
    strict_1.default.equal(socket.closed, true);
});
(0, node_test_1.default)("marks authentication expiry as terminal and closes the socket once", () => {
    const { events, socket } = setup();
    socket.emit("message", { data: JSON.stringify({ type: "session.error", status: 410, error: "expired" }) });
    strict_1.default.equal(events[0]?.type, "message");
    strict_1.default.deepEqual(events[1], { type: "closed", terminal: true, status: 410, message: "expired" });
    strict_1.default.equal(events.filter((event) => event.type === "closed").length, 1);
    strict_1.default.equal(socket.closed, true);
});
