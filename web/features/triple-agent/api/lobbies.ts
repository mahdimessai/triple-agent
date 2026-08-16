import type { LobbySession } from "../protocol";

export function apiBaseUrl() {
  return process.env.NEXT_PUBLIC_TRIPLE_AGENT_API_URL ?? "http://localhost:8080";
}

async function postJson<T>(path: string, body: unknown): Promise<T> {
  const response = await fetch(`${apiBaseUrl()}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const payload = (await response.json()) as unknown;
  if (!response.ok) {
    const message = typeof payload === "object" && payload !== null && "error" in payload && typeof payload.error === "string"
      ? payload.error
      : `Request failed (${response.status})`;
    throw new Error(message);
  }
  return payload as T;
}

export function createLobby(playerName = "") {
  return postJson<LobbySession>("/api/lobbies", {
    player_name: playerName,
  });
}

export function joinLobby(joinCode: string, playerName = "") {
  return postJson<LobbySession>("/api/lobbies/join", {
    join_code: joinCode,
    player_name: playerName,
  });
}

export async function leaveLobby(session: LobbySession) {
  return postJson<{ left: boolean }>("/api/lobbies/leave", {
    room_id: session.room_id,
    player_id: session.player_id,
    reconnect_token: session.reconnect_token,
  });
}
