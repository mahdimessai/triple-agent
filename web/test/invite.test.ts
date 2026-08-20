import test from "node:test";
import assert from "node:assert/strict";
import { buildRoomInviteUrl, parseRoomInvite } from "../features/triple-agent/invite/invite";

test("invite parser rejects incomplete codes and sanitizes display-only host", () => {
  assert.equal(parseRoomInvite("?join=abc12"), null);
  assert.deepEqual(
    parseRoomInvite("?join=ab12c3&host=Mahdi%20%F0%9F%98%88%20Messai"),
    { code: "AB12C3", host: "Mahdi  Messai" },
  );
});

test("invite URL uses URLSearchParams encoding", () => {
  assert.equal(
    buildRoomInviteUrl("https://game.example", "/play", "ABC123", "A & B"),
    "https://game.example/play?join=ABC123&host=A+%26+B",
  );
});
