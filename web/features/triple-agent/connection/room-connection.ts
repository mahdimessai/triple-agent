import type {
  ClientCommand,
  ConnectionState,
  ConnectionStateDetails,
  LobbySession,
  RoomProjection,
  RoomServerMessage,
  WireCommand,
} from "../protocol";
import { apiBaseUrl } from "../api/lobbies";

function websocketUrl(session: LobbySession) {
  const url = new URL(apiBaseUrl());
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  url.pathname = "/ws";
  url.search = new URLSearchParams({
    room_id: session.room_id,
    player_id: session.player_id,
  }).toString();
  return url.toString();
}

export type RoomConnection = {
  send(command: ClientCommand, expectedVersion: number): string;
  close(): void;
};

export function connectRoom(
  session: LobbySession,
  onProjection: (projection: RoomProjection) => void,
  onAck: (requestId: string, ok: boolean, error?: string) => void,
  onState: (state: ConnectionState, details?: ConnectionStateDetails) => void,
): RoomConnection {
  const socket = new WebSocket(websocketUrl(session));
  let terminalFailure = false;
  let latestVersion = -1;
  let resyncPending = false;
  onState("connecting");

  socket.addEventListener("open", () => {
    socket.send(JSON.stringify({ type: "room.auth", reconnect_token: session.reconnect_token }));
  });
  socket.addEventListener("close", () => {
    if (!terminalFailure) onState("closed");
  });
  socket.addEventListener("error", () => {
    if (!terminalFailure) onState("closed");
  });
  socket.addEventListener("message", (event) => {
    let message: RoomServerMessage;
    try {
      message = JSON.parse(String(event.data)) as RoomServerMessage;
    } catch {
      return;
    }
    if (message.type === "room.projection") {
      if (message.public.version < latestVersion) return;
      if (resyncPending) {
        resyncPending = false;
        latestVersion = message.public.version;
        onProjection(message);
        return;
      }
      if (latestVersion >= 0 && message.public.version > latestVersion + 1 && socket.readyState === WebSocket.OPEN) {
        resyncPending = true;
        socket.send(JSON.stringify({ kind: "room.resync" }));
        return;
      }
      latestVersion = message.public.version;
      onProjection(message);
    } else if (message.type === "command.ack") {
      onAck(message.request_id, message.ok, message.error);
    } else if (message.type === "session.authenticated") {
      onState("open");
    } else if (message.type === "session.error") {
      terminalFailure = message.status === 401 || message.status === 410;
      onState("closed", { terminal: terminalFailure, message: message.error, status: message.status });
      socket.close();
    }
  });

  return {
    send(command: ClientCommand, expectedVersion: number) {
      if (socket.readyState !== WebSocket.OPEN) {
        throw new Error("Room connection is not open");
      }
      const requestId = crypto.randomUUID();
      const wireCommand: WireCommand = {
        ...command,
        request_id: requestId,
        expected_version: expectedVersion,
      };
      socket.send(JSON.stringify(wireCommand));
      return requestId;
    },
    close() {
      socket.close();
    },
  };
}
