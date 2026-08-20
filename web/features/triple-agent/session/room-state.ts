import type { ClientCommand, RoomIdentity, RoomProjection } from "../protocol";

export type RoomStatus = "idle" | "connecting" | "online" | "reconnecting" | "leaving";

export type RoomNotice =
  | { kind: "kicked"; message: string; joinCode: string }
  | { kind: "session-expired"; message: string };

export type PendingCommand = {
  requestId: string;
  kind: ClientCommand["kind"];
};

export type RoomState = {
  identity: RoomIdentity | null;
  projection: RoomProjection | null;
  status: RoomStatus;
  error: string | null;
  notice: RoomNotice | null;
  pending: PendingCommand | null;
};

export type RoomStateAction =
  | { type: "request-started" }
  | { type: "request-failed"; message: string }
  | { type: "connect-started"; identity: RoomIdentity; reconnecting: boolean }
  | { type: "connected" }
  | { type: "projection"; projection: RoomProjection }
  | { type: "command-sent"; pending: PendingCommand }
  | { type: "command-acked"; requestId: string; error?: string }
  | { type: "connection-lost" }
  | { type: "error"; message: string }
  | { type: "session-ended"; notice: RoomNotice }
  | { type: "leave-started" }
  | { type: "left"; error?: string }
  | { type: "notice-dismissed" };

export const INITIAL_ROOM_STATE: RoomState = {
  identity: null,
  projection: null,
  status: "idle",
  error: null,
  notice: null,
  pending: null,
};

export function roomReducer(state: RoomState, action: RoomStateAction): RoomState {
  switch (action.type) {
    case "request-started":
      return { ...state, status: "connecting", error: null, notice: null };
    case "request-failed":
      return { ...state, status: "idle", error: action.message };
    case "connect-started":
      return {
        ...state,
        identity: action.identity,
        status: action.reconnecting ? "reconnecting" : "connecting",
        error: null,
        pending: null,
      };
    case "connected":
      return { ...state, status: "online", error: null };
    case "projection":
      return { ...state, projection: action.projection, error: null };
    case "command-sent":
      return { ...state, pending: action.pending, error: null };
    case "command-acked":
      if (state.pending?.requestId !== action.requestId) return state;
      return { ...state, pending: null, error: action.error ?? null };
    case "connection-lost":
      return { ...state, status: "reconnecting", pending: null };
    case "error":
      return { ...state, error: action.message };
    case "session-ended":
      return { ...INITIAL_ROOM_STATE, notice: action.notice };
    case "leave-started":
      return { ...state, status: "leaving", pending: null, error: null };
    case "left":
      return { ...INITIAL_ROOM_STATE, error: action.error ?? null };
    case "notice-dismissed":
      return { ...state, notice: null };
  }
}
