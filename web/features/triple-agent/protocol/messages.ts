import type { RoomProjection } from "./projections";

export type CommandAck = {
  type: "command.ack";
  request_id: string;
  ok: boolean;
  error?: string;
};

export type SessionAuthenticated = {
  type: "session.authenticated";
};

export type SessionError = {
  type: "session.error";
  status?: number;
  error?: string;
};

export type RoomServerMessage = RoomProjection | CommandAck | SessionAuthenticated | SessionError;

export type ConnectionState = "connecting" | "open" | "closed";

export type ConnectionStateDetails = {
  terminal?: boolean;
  message?: string;
  status?: number;
};
