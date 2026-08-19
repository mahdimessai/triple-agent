"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const node_test_1 = __importDefault(require("node:test"));
const strict_1 = __importDefault(require("node:assert/strict"));
const operations_1 = require("../app/_game/operations");
function projection() {
    return {
        type: "room.projection",
        public: {
            room_id: "r", host_id: "p1", phase: "OPERATION_RESULT", version: 3,
            players: [
                { id: "p1", name: "ALPHA", seat: 1, ready: true, connected: true, vote_submitted: false },
                { id: "p2", name: "BRAVO", seat: 2, ready: true, connected: true, vote_submitted: false },
            ],
            settings: { discussion_timer_enabled: true, discussion_seconds: 300, enabled_operations: ["OneRandom"] },
        },
        private: { player_id: "p1", vote_submitted: false, can_submit: true },
    };
}
(0, node_test_1.default)("operation ids and aliases resolve to the single merged catalog", () => {
    strict_1.default.equal((0, operations_1.operationIdForServerKind)("SpyTransfer"), "Swap");
    strict_1.default.equal((0, operations_1.operationIdForServerKind)("Detector"), "Detector");
    strict_1.default.equal((0, operations_1.operationIdForServerKind)("not-real"), "OneRandom");
    strict_1.default.equal((0, operations_1.getOperation)("OneRandom").name, "Anonymous Tip");
});
(0, node_test_1.default)("every configured operation set references a real catalog entry", () => {
    const ids = new Set(operations_1.operations.map((operation) => operation.id));
    for (const id of [...operations_1.liveOperationIds, ...operations_1.hiddenAgendaMemberIds, ...operations_1.packOperationIds])
        strict_1.default.equal(ids.has(id), true, `missing ${id}`);
    for (const operation of operations_1.operations) {
        strict_1.default.equal((0, operations_1.operationBrief)(operation).length > 0, true, `${operation.id} needs a brief`);
        strict_1.default.equal((0, operations_1.roomBriefing)(operation, "ALPHA", true, operation.publicUpdate).length > 0, true, `${operation.id} needs recipient copy`);
    }
});
(0, node_test_1.default)("operation result copy resolves player ids through the projection", () => {
    const result = (0, operations_1.operationResultText)({ code: "FACTION_REVEALED", target_player_id: "p2", target_faction: "VIRUS", message: "fallback" }, projection());
    strict_1.default.equal(result, "BRAVO is VIRUS.");
});
