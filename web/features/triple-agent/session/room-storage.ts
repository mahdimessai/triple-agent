import { isRoomIdentity, type RoomIdentity } from "../protocol";

const STORAGE_KEY = "triple-agent-room";

export function loadRoomIdentity(): RoomIdentity | null {
  try {
    const raw = window.sessionStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const value: unknown = JSON.parse(raw);
    if (!isRoomIdentity(value)) { window.sessionStorage.removeItem(STORAGE_KEY); return null; }
    return { join_code: value.join_code, reconnect_token: value.reconnect_token };
  } catch { return null; }
}

export function saveRoomIdentity(identity: RoomIdentity): void {
  try { window.sessionStorage.setItem(STORAGE_KEY, JSON.stringify(identity)); } catch {
    // The active room still works if storage is unavailable; only reload restore is lost.
  }
}

export function clearRoomIdentity(): void {
  try { window.sessionStorage.removeItem(STORAGE_KEY); } catch {
    // Nothing else can be done when storage is blocked.
  }
}
