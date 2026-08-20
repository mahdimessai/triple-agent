import type { RoomIdentity } from "../protocol";

const DEFAULT_API_BASE_URL = "http://localhost:8080";

export function apiBaseUrl(): string {
  return (process.env.NEXT_PUBLIC_TRIPLE_AGENT_API_URL ?? DEFAULT_API_BASE_URL).replace(/\/$/, "");
}

export function apiUrl(path: string): string {
  return `${apiBaseUrl()}${path}`;
}

export function websocketUrl(identity: RoomIdentity): string {
  const url = new URL(apiBaseUrl());
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  url.pathname = "/ws";
  url.search = new URLSearchParams({
    room_id: identity.room_id,
    player_id: identity.player_id,
  }).toString();
  return url.toString();
}
