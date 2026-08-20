import {
  parseRoomServerMessage,
  type ClientCommand,
  type RoomIdentity,
  type RoomServerMessage,
  type SessionError,
  type WireCommand,
} from "../protocol";
import { apiBaseUrl } from "./api-base";

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
  url.search = new URLSearchParams({ room_id: identity.room_id, player_id: identity.player_id }).toString();
  return url.toString();
}

function serializeCommand(command: ClientCommand, expectedVersion: number, requestId: string): WireCommand {
  return { ...command, request_id: requestId, expected_version: expectedVersion };
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
  const emit = (event: RoomSocketEvent) => { if (!disposed) onEvent(event); };
  const emitClosed = (details: Omit<Extract<RoomSocketEvent, { type: "closed" }>, "type">) => {
    if (disposed || closed) return;
    closed = true;
    onEvent({ type: "closed", ...details });
  };

  socket.addEventListener("open", () => {
    if (!disposed) socket.send(JSON.stringify({ type: "room.auth", reconnect_token: identity.reconnect_token }));
  });
  socket.addEventListener("message", (event) => {
    if (disposed) return;
    let parsed: unknown;
    try { parsed = JSON.parse(String(event.data)); } catch { emit({ type: "invalid-message" }); return; }
    const message = parseRoomServerMessage(parsed);
    if (!message) { emit({ type: "invalid-message" }); return; }
    if (message.type === "session.authenticated") { emit({ type: "open" }); return; }
    emit({ type: "message", message });
    if (message.type === "session.error") { emitClosed(closeDetails(message)); socket.close(); }
  });
  socket.addEventListener("error", () => { emitClosed({ terminal: false, message: "The room connection failed" }); socket.close(); });
  socket.addEventListener("close", () => emitClosed({ terminal: false }));

  return {
    send(command, expectedVersion) {
      if (disposed || socket.readyState !== WebSocket.OPEN) throw new Error("Room connection is not open");
      const requestId = crypto.randomUUID();
      socket.send(JSON.stringify(serializeCommand(command, expectedVersion, requestId)));
      return requestId;
    },
    resync() {
      if (!disposed && socket.readyState === WebSocket.OPEN) socket.send(JSON.stringify({ kind: "room.resync" } satisfies WireCommand));
    },
    close() {
      if (disposed) return;
      disposed = true;
      socket.close();
    },
  };
}
