import { strictEqual, notStrictEqual } from "node:assert";
import { test } from "node:test";
import type { RoomProjection } from "../protocol";
import { INITIAL_ROOM_STATE, roomReducer } from "./room-state";

const projection: RoomProjection = {
  type: "room.projection",
  public: {
    room_id: "room", host_id: "p1", phase: "LOBBY", version: 1, players: [],
    settings: { discussion_timer_enabled: true, discussion_seconds: 300, enabled_operations: [] },
  },
  private: { player_id: "p1", can_submit: false, vote_submitted: false },
};

test("resyncing an unchanged snapshot preserves render identity", () => {
  const state = { ...INITIAL_ROOM_STATE, projection };
  strictEqual(roomReducer(state, { type: "projection", projection: structuredClone(projection) }), state);
});

test("a repeated snapshot still clears a previous error without replacing the projection", () => {
  const state = { ...INITIAL_ROOM_STATE, projection, error: "Temporary error" };
  const next = roomReducer(state, { type: "projection", projection: structuredClone(projection) });
  strictEqual(next.error, null);
  strictEqual(next.projection, projection);
});

test("new versions, rooms, and players must not reuse an old snapshot", () => {
  const state = { ...INITIAL_ROOM_STATE, projection };
  const updates: RoomProjection[] = [
    { ...projection, public: { ...projection.public, version: 8 } },
    { ...projection, public: { ...projection.public, room_id: "another-room" } },
    { ...projection, private: { ...projection.private, player_id: "p2" } },
  ];
  for (const update of updates) {
    const next = roomReducer(state, { type: "projection", projection: update });
    notStrictEqual(next, state);
    strictEqual(next.projection, update);
  }
});
