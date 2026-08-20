import {
  isRoomIdentity,
  parseRoomServerMessage,
  type ClientCommand,
  type RoomIdentity,
  type RoomServerMessage,
  type SessionError,
  type WireCommand,
} from "./protocol";

const DEFAULT_API_BASE_URL = "http://localhost:8080";

function apiBaseUrl(): string {
  return (process.env.NEXT_PUBLIC_TRIPLE_AGENT_API_URL ?? DEFAULT_API_BASE_URL).replace(/\/$/, "");
}

function apiUrl(path: string): string {
  return `${apiBaseUrl()}${path}`;
}

async function postJson(path: string, body: unknown): Promise<unknown> {
  const response = await fetch(apiUrl(path), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });

  let payload: unknown;
  try {
    payload = await response.json();
  } catch {
    payload = null;
  }

  if (!response.ok) {
    const message = typeof payload === "object"
      && payload !== null
      && "error" in payload
      && typeof payload.error === "string"
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
  return roomIdentityRequest("/api/lobbies/join", {
    join_code: joinCode,
    player_name: playerName,
  });
}

export async function leaveRoom(identity: RoomIdentity): Promise<void> {
  await postJson("/api/lobbies/leave", {
    room_id: identity.room_id,
    player_id: identity.player_id,
    reconnect_token: identity.reconnect_token,
  });
}

export function leaveRoomOnPageHide(identity: RoomIdentity): void {
  const body = JSON.stringify({
    room_id: identity.room_id,
    player_id: identity.player_id,
    reconnect_token: identity.reconnect_token,
  });
  const url = apiUrl("/api/lobbies/leave");

  if (typeof navigator !== "undefined" && typeof navigator.sendBeacon === "function") {
    const accepted = navigator.sendBeacon(url, new Blob([body], { type: "text/plain;charset=UTF-8" }));
    if (accepted) return;
  }

  void fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body,
    keepalive: true,
  });
}

export type RoomSocketEvent =
  | { type: "open" }
  | { type: "message"; message: RoomServerMessage }
  | { type: "invalid-message" }
  | { type: "closed"; terminal: boolean; status?: number; message?: string; code?: string };

export type RoomSocket = {
  send(command: ClientCommand, expectedVersion: number): string;
  resync(): void;
  close(): void;
};

function websocketUrl(identity: RoomIdentity): string {
  const url = new URL(apiBaseUrl());
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  url.pathname = "/ws";
  url.search = new URLSearchParams({
    room_id: identity.room_id,
    player_id: identity.player_id,
  }).toString();
  return url.toString();
}

function nextRequestId(): string {
  return crypto.randomUUID();
}

function serializeCommand(command: ClientCommand, expectedVersion: number, requestId: string): WireCommand {
  return {
    ...command,
    request_id: requestId,
    expected_version: expectedVersion,
  };
}

function closeDetails(error: SessionError): Omit<Extract<RoomSocketEvent, { type: "closed" }>, "type"> {
  return {
    terminal: error.status === 401 || error.status === 410,
    status: error.status,
    message: error.error,
    ...(error.code ? { code: error.code } : {}),
  };
}

export function connectRoom(identity: RoomIdentity, onEvent: (event: RoomSocketEvent) => void): RoomSocket {
  const socket = new WebSocket(websocketUrl(identity));
  let disposed = false;
  let closed = false;

  function emit(event: RoomSocketEvent): void {
    if (!disposed) onEvent(event);
  }

  function emitClosed(details: Omit<Extract<RoomSocketEvent, { type: "closed" }>, "type">): void {
    if (disposed || closed) return;
    closed = true;
    onEvent({ type: "closed", ...details });
  }

  socket.addEventListener("open", () => {
    if (disposed) return;
    socket.send(JSON.stringify({ type: "room.auth", reconnect_token: identity.reconnect_token }));
  });

  socket.addEventListener("message", (event) => {
    if (disposed) return;

    let parsed: unknown;
    try {
      parsed = JSON.parse(String(event.data));
    } catch {
      emit({ type: "invalid-message" });
      return;
    }

    const message = parseRoomServerMessage(parsed);
    if (!message) {
      emit({ type: "invalid-message" });
      return;
    }

    if (message.type === "session.authenticated") {
      emit({ type: "open" });
      return;
    }

    emit({ type: "message", message });

    if (message.type === "session.error") {
      emitClosed(closeDetails(message));
      socket.close();
    }
  });

  socket.addEventListener("error", () => {
    emitClosed({ terminal: false, message: "The room connection failed" });
    socket.close();
  });

  socket.addEventListener("close", () => {
    emitClosed({ terminal: false });
  });

  return {
    send(command: ClientCommand, expectedVersion: number): string {
      if (disposed || socket.readyState !== WebSocket.OPEN) {
        throw new Error("Room connection is not open");
      }
      const requestId = nextRequestId();
      socket.send(JSON.stringify(serializeCommand(command, expectedVersion, requestId)));
      return requestId;
    },

    resync(): void {
      if (disposed || socket.readyState !== WebSocket.OPEN) return;
      socket.send(JSON.stringify({ kind: "room.resync" } satisfies WireCommand));
    },

    close(): void {
      if (disposed) return;
      disposed = true;
      socket.close();
    },
  };
}
