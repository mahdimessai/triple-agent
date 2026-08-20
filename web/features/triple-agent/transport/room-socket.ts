import {
  parseRoomServerMessage,
  type ClientCommand,
  type RoomIdentity,
  type RoomServerMessage,
  type SessionError,
  type WireCommand,
} from "../protocol";
import { websocketUrl } from "./endpoints";

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
