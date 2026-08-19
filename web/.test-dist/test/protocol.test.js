"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const node_test_1 = __importDefault(require("node:test"));
const strict_1 = __importDefault(require("node:assert/strict"));
const protocol_1 = require("../app/_game/protocol");
function projection() {
    return {
        type: "room.projection",
        public: {
            room_id: "r1", host_id: "p1", phase: "LOBBY", version: 1,
            players: [{ id: "p1", name: "A", seat: 1, ready: false, connected: true, vote_submitted: false }],
            settings: { discussion_timer_enabled: false, discussion_seconds: 300, enabled_operations: [] },
        },
        private: { player_id: "p1", vote_submitted: false, can_submit: true },
    };
}
(0, node_test_1.default)("parses the valid server message variants", () => {
    strict_1.default.deepEqual((0, protocol_1.parseRoomServerMessage)({ type: "session.authenticated" }), { type: "session.authenticated" });
    strict_1.default.deepEqual((0, protocol_1.parseRoomServerMessage)({ type: "command.ack", request_id: "a", ok: true }), { type: "command.ack", request_id: "a", ok: true });
    strict_1.default.deepEqual((0, protocol_1.parseRoomServerMessage)({ type: "session.error", status: 410, error: "gone" }), { type: "session.error", status: 410, error: "gone" });
    strict_1.default.equal((0, protocol_1.parseRoomServerMessage)(projection())?.type, "room.projection");
});
(0, node_test_1.default)("rejects unknown or structurally invalid messages", () => {
    strict_1.default.equal((0, protocol_1.parseRoomServerMessage)({ type: "mystery" }), null);
    strict_1.default.equal((0, protocol_1.parseRoomServerMessage)({ type: "command.ack", request_id: 3, ok: true }), null);
    const invalid = projection();
    invalid.public.players = [{ id: "p1" }];
    strict_1.default.equal((0, protocol_1.parseRoomServerMessage)(invalid), null);
    const badOperation = projection();
    badOperation.public.operation = {
        kind: "Detector", name: "Detector", input_kind: "TWO_TARGETS",
        active_player_id: "p1", active_player_name: 42, public_instruction: "Choose two",
    };
    strict_1.default.equal((0, protocol_1.parseRoomServerMessage)(badOperation), null);
    const badLeaderboard = projection();
    badLeaderboard.public.leaderboard = [{
            player_id: "p1", name: "A", faction: "SERVICE", votes: Number.NaN, result: "WINNER",
        }];
    strict_1.default.equal((0, protocol_1.parseRoomServerMessage)(badLeaderboard), null);
});
(0, node_test_1.default)("validates persisted room identities before reuse", () => {
    strict_1.default.equal((0, protocol_1.isRoomIdentity)({ room_id: "r", join_code: "ABC123", player_id: "p", reconnect_token: "t" }), true);
    strict_1.default.equal((0, protocol_1.isRoomIdentity)({ room_id: "r", join_code: "ABC123", player_id: "p" }), false);
});
