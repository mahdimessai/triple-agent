export type RoomInvite = {
  code: string;
  host?: string;
};

export function parseRoomInvite(search: string): RoomInvite | null {
  const params = new URLSearchParams(search);
  const code = (params.get("join") ?? "")
    .toUpperCase()
    .replace(/[^A-Z0-9]/g, "")
    .slice(0, 6);
  if (code.length !== 6) return null;

  const host = (params.get("host") ?? "")
    .replace(/[^\p{L}\p{N} '._-]/gu, "")
    .trim()
    .slice(0, 24);

  return { code, ...(host ? { host } : {}) };
}

export function buildRoomInviteUrl(origin: string, pathname: string, code: string, hostName?: string): string {
  const params = new URLSearchParams({ join: code });
  if (hostName) params.set("host", hostName);
  return `${origin}${pathname}?${params.toString()}`;
}
