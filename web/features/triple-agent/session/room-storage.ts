import { isRoomIdentity, type RoomIdentity } from "../protocol";

const STORAGE_KEY = "triple-agent-room";

export function loadRoomIdentity(): RoomIdentity | null {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const value: unknown = JSON.parse(raw);
    if (!isRoomIdentity(value)) { window.localStorage.removeItem(STORAGE_KEY); return null; }
    return value;
  } catch { return null; }
}

export function saveRoomIdentity(identity: RoomIdentity): void {
  try { window.localStorage.setItem(STORAGE_KEY, JSON.stringify(identity)); } catch {
    // The active room still works if storage is unavailable; only reload restore is lost.
  }
}

export function clearRoomIdentity(): void {
  try { window.localStorage.removeItem(STORAGE_KEY); } catch {
    // Nothing else can be done when storage is blocked.
  }
}
