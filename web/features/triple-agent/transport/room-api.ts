import { isRoomIdentity, type RoomIdentity } from "../protocol";
import { apiUrl } from "./api-base";

async function postJson(path: string, body: unknown): Promise<unknown> {
  const response = await fetch(apiUrl(path), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });

  let payload: unknown;
  try { payload = await response.json(); } catch { payload = null; }

  if (!response.ok) {
    const message = typeof payload === "object" && payload !== null && "error" in payload && typeof payload.error === "string"
      ? payload.error
      : `Request failed (${response.status})`;
    throw new Error(message);
  }
  return payload;
}

async function roomIdentityRequest(path: string, body: unknown): Promise<RoomIdentity> {
  const payload = await postJson(path, body);
  if (!isRoomIdentity(payload)) throw new Error("The room server returned an invalid session");
  return payload;
}

export function createRoom(playerName: string): Promise<RoomIdentity> {
  return roomIdentityRequest("/api/lobbies", { player_name: playerName });
}

export function joinRoom(joinCode: string, playerName: string): Promise<RoomIdentity> {
  return roomIdentityRequest("/api/lobbies/join", { join_code: joinCode, player_name: playerName });
}

export async function leaveRoom(identity: RoomIdentity): Promise<void> {
  await postJson("/api/lobbies/leave", {
    room_id: identity.room_id,
    player_id: identity.player_id,
    reconnect_token: identity.reconnect_token,
  });
}

export function leaveRoomOnPageHide(identity: RoomIdentity): void {
  const body = JSON.stringify({ room_id: identity.room_id, player_id: identity.player_id, reconnect_token: identity.reconnect_token });
  const url = apiUrl("/api/lobbies/leave");
  if (typeof navigator !== "undefined" && typeof navigator.sendBeacon === "function") {
    const accepted = navigator.sendBeacon(url, new Blob([body], { type: "text/plain;charset=UTF-8" }));
    if (accepted) return;
  }
  void fetch(url, { method: "POST", headers: { "Content-Type": "application/json" }, body, keepalive: true });
}
