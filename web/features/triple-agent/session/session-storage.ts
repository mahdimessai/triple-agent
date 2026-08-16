import type { LobbySession } from "../protocol";

const lobbySessionStorageKey = "triple-agent:session:v1";

export function saveLobbySession(session: LobbySession) {
  if (typeof window !== "undefined") {
    try {
      window.sessionStorage.setItem(lobbySessionStorageKey, JSON.stringify(session));
    } catch {
      // Private browsing and storage-disabled environments can reject writes.
    }
  }
}

export function loadLobbySession(): LobbySession | undefined {
  if (typeof window === "undefined") return undefined;
  try {
    const stored = window.sessionStorage.getItem(lobbySessionStorageKey);
    if (!stored) return undefined;
    const session = JSON.parse(stored) as Partial<LobbySession>;
    if (!session.room_id || !session.join_code || !session.player_id || !session.reconnect_token) return undefined;
    return session as LobbySession;
  } catch {
    return undefined;
  }
}

export function clearLobbySession() {
  if (typeof window !== "undefined") {
    try {
      window.sessionStorage.removeItem(lobbySessionStorageKey);
    } catch {
      // Storage may be unavailable even when the session is still in memory.
    }
  }
}
