import test from "node:test";
import assert from "node:assert/strict";
import { isRoomIdentity, parseRoomServerMessage, type RoomProjection } from "../app/_game/protocol";

function projection(): RoomProjection {
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

test("parses the valid server message variants", () => {
  assert.deepEqual(parseRoomServerMessage({ type: "session.authenticated" }), { type: "session.authenticated" });
  assert.deepEqual(parseRoomServerMessage({ type: "command.ack", request_id: "a", ok: true }), { type: "command.ack", request_id: "a", ok: true });
  assert.deepEqual(parseRoomServerMessage({ type: "session.error", status: 410, error: "gone" }), { type: "session.error", status: 410, error: "gone" });
  assert.equal(parseRoomServerMessage(projection())?.type, "room.projection");
});

test("rejects unknown or structurally invalid messages", () => {
  assert.equal(parseRoomServerMessage({ type: "mystery" }), null);
  assert.equal(parseRoomServerMessage({ type: "command.ack", request_id: 3, ok: true }), null);
  const invalid = projection() as unknown as { public: { players: unknown[] } };
  invalid.public.players = [{ id: "p1" }];
  assert.equal(parseRoomServerMessage(invalid), null);

  const badOperation = projection() as unknown as RoomProjection;
  badOperation.public.operation = {
    kind: "Detector", name: "Detector", input_kind: "TWO_TARGETS",
    active_player_id: "p1", active_player_name: 42 as unknown as string, public_instruction: "Choose two",
  };
  assert.equal(parseRoomServerMessage(badOperation), null);

  const badLeaderboard = projection() as unknown as RoomProjection;
  badLeaderboard.public.leaderboard = [{
    player_id: "p1", name: "A", faction: "SERVICE", votes: Number.NaN, result: "WINNER",
  }];
  assert.equal(parseRoomServerMessage(badLeaderboard), null);
});

test("validates persisted room identities before reuse", () => {
  assert.equal(isRoomIdentity({ room_id: "r", join_code: "ABC123", player_id: "p", reconnect_token: "t" }), true);
  assert.equal(isRoomIdentity({ room_id: "r", join_code: "ABC123", player_id: "p" }), false);
});
