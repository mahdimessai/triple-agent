import test from "node:test";
import assert from "node:assert/strict";
import {
  getOperation, hiddenAgendaMemberIds, liveOperationIds, operationBrief, operationIdForServerKind,
  operationResultText, operations, packOperationIds, roomBriefing,
} from "../features/triple-agent/operations";
import type { RoomProjection } from "../features/triple-agent/protocol";

function projection(): RoomProjection {
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

test("operation ids and aliases resolve to the single merged catalog", () => {
  assert.equal(operationIdForServerKind("SpyTransfer"), "Swap");
  assert.equal(operationIdForServerKind("Detector"), "Detector");
  assert.equal(operationIdForServerKind("not-real"), "OneRandom");
  assert.equal(getOperation("OneRandom").name, "Anonymous Tip");
});

test("every configured operation set references a real catalog entry", () => {
  const ids = new Set<string>(operations.map((operation) => operation.id));
  for (const id of [...liveOperationIds, ...hiddenAgendaMemberIds, ...packOperationIds]) assert.equal(ids.has(id), true, `missing ${id}`);
  for (const operation of operations) {
    assert.equal(operationBrief(operation).length > 0, true, `${operation.id} needs a brief`);
    assert.equal(roomBriefing(operation, "ALPHA", true, operation.publicUpdate).length > 0, true, `${operation.id} needs recipient copy`);
  }
});

test("operation result copy resolves player ids through the projection", () => {
  const result = operationResultText({ code: "FACTION_REVEALED", target_player_id: "p2", target_faction: "VIRUS", message: "fallback" }, projection());
  assert.equal(result, "BRAVO is VIRUS.");
});
