"use client";

import { useEffect, useReducer, useRef } from "react";
import {
  connectRoom,
  createRoom as createRoomRequest,
  joinRoom as joinRoomRequest,
  leaveRoom as leaveRoomRequest,
  leaveRoomOnPageHide,
  type RoomSocket,
  type RoomSocketEvent,
} from "./room";
import { isRoomIdentity, type ClientCommand, type RoomIdentity, type RoomProjection } from "./protocol";

const STORAGE_KEY = "triple-agent-room";
const RECONNECT_GRACE_PERIOD_MS = 300_000;
const INITIAL_RECONNECT_DELAY_MS = 500;
const MAX_RECONNECT_DELAY_MS = 8_000;
const RECONNECT_JITTER_MS = 250;
const SESSION_EXPIRED_MESSAGE = "This room session has expired";
const OFFLINE_SESSION_EXPIRED_MESSAGE = "The room session expired after five minutes offline";
const KICKED_MESSAGE = "You have been removed from the lobby by the host.";

export type RoomStatus = "idle" | "connecting" | "online" | "reconnecting" | "leaving";

export type RoomNotice =
  | { kind: "kicked"; message: string; joinCode: string }
  | { kind: "session-expired"; message: string };

export type PendingCommand = {
  requestId: string;
  kind: ClientCommand["kind"];
};

export type UseRoomResult = {
  identity: RoomIdentity | null;
  projection: RoomProjection | null;
  status: RoomStatus;
  error: string | null;
  notice: RoomNotice | null;
  pending: PendingCommand | null;
  create(playerName: string): Promise<void>;
  join(joinCode: string, playerName: string): Promise<void>;
  leave(): Promise<void>;
  send(command: ClientCommand): void;
  dismissNotice(): void;
};

type RoomState = {
  identity: RoomIdentity | null;
  projection: RoomProjection | null;
  status: RoomStatus;
  error: string | null;
  notice: RoomNotice | null;
  pending: PendingCommand | null;
};

type RoomStateAction =
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

const INITIAL_STATE: RoomState = {
  identity: null,
  projection: null,
  status: "idle",
  error: null,
  notice: null,
  pending: null,
};

function roomReducer(state: RoomState, action: RoomStateAction): RoomState {
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
      return {
        ...state,
        pending: null,
        error: action.error ?? null,
      };
    case "connection-lost":
      return { ...state, status: "reconnecting", pending: null };
    case "error":
      return { ...state, error: action.message };
    case "session-ended":
      return {
        ...INITIAL_STATE,
        notice: action.notice,
      };
    case "leave-started":
      return { ...state, status: "leaving", pending: null, error: null };
    case "left":
      return { ...INITIAL_STATE, error: action.error ?? null };
    case "notice-dismissed":
      return { ...state, notice: null };
  }
}

function reconnectDelay(attempt: number, randomValue = Math.random()): number {
  const exponential = Math.min(MAX_RECONNECT_DELAY_MS, INITIAL_RECONNECT_DELAY_MS * 2 ** attempt);
  return exponential + Math.floor(randomValue * RECONNECT_JITTER_MS);
}

function loadRoomIdentity(): RoomIdentity | null {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const value: unknown = JSON.parse(raw);
    if (!isRoomIdentity(value)) {
      window.localStorage.removeItem(STORAGE_KEY);
      return null;
    }
    return value;
  } catch {
    return null;
  }
}

function saveRoomIdentity(identity: RoomIdentity): void {
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(identity));
  } catch {
    // The active room still works if storage is unavailable; only reload restore is lost.
  }
}

function clearRoomIdentity(): void {
  try {
    window.localStorage.removeItem(STORAGE_KEY);
  } catch {
    // Nothing else can be done when storage is blocked.
  }
}

function errorMessage(cause: unknown, fallback: string): string {
  return cause instanceof Error && cause.message ? cause.message : fallback;
}

export function useRoom(): UseRoomResult {
  const [state, dispatch] = useReducer(roomReducer, INITIAL_STATE);
  const socketRef = useRef<RoomSocket | null>(null);
  const reconnectTimerRef = useRef<number | null>(null);
  const reconnectAttemptRef = useRef(0);
  const reconnectDeadlineRef = useRef<number | null>(null);
  const connectionGenerationRef = useRef(0);
  const projectionVersionRef = useRef(-1);
  const phaseRef = useRef<RoomProjection["public"]["phase"] | null>(null);
  const actionLockRef = useRef(false);

  function cancelReconnect(): void {
    if (reconnectTimerRef.current !== null) {
      window.clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = null;
    }
  }

  function disconnect(): void {
    connectionGenerationRef.current += 1;
    cancelReconnect();
    reconnectAttemptRef.current = 0;
    reconnectDeadlineRef.current = null;
    socketRef.current?.close();
    socketRef.current = null;
  }

  function endSession(notice: RoomNotice): void {
    disconnect();
    projectionVersionRef.current = -1;
    phaseRef.current = null;
    clearRoomIdentity();
    dispatch({ type: "session-ended", notice });
  }

  function scheduleReconnect(identity: RoomIdentity): void {
    if (reconnectTimerRef.current !== null) return;

    const now = Date.now();
    const deadline = reconnectDeadlineRef.current ?? now + RECONNECT_GRACE_PERIOD_MS;
    reconnectDeadlineRef.current = deadline;
    const remaining = deadline - now;

    if (remaining <= 0) {
      endSession({ kind: "session-expired", message: OFFLINE_SESSION_EXPIRED_MESSAGE });
      return;
    }

    dispatch({ type: "connection-lost" });
    const delay = Math.min(reconnectDelay(reconnectAttemptRef.current++), remaining);
    reconnectTimerRef.current = window.setTimeout(() => {
      reconnectTimerRef.current = null;
      if (Date.now() >= deadline) {
        endSession({ kind: "session-expired", message: OFFLINE_SESSION_EXPIRED_MESSAGE });
        return;
      }
      if (!socketRef.current) openConnection(identity, true);
    }, delay);
  }

  function openConnection(identity: RoomIdentity, reconnecting: boolean): void {
    cancelReconnect();
    socketRef.current?.close();
    socketRef.current = null;

    const generation = ++connectionGenerationRef.current;
    let latestPhase: RoomProjection["public"]["phase"] | null = null;
    let resyncPending = false;

    dispatch({ type: "connect-started", identity, reconnecting });

    const socket = connectRoom(identity, (event: RoomSocketEvent) => {
      if (generation !== connectionGenerationRef.current) return;

      if (event.type === "open") {
        reconnectAttemptRef.current = 0;
        reconnectDeadlineRef.current = null;
        dispatch({ type: "connected" });
        return;
      }

      if (event.type === "invalid-message") {
        dispatch({ type: "error", message: "The room server sent an invalid update" });
        return;
      }

      if (event.type === "message") {
        const message = event.message;
        if (message.type === "command.ack") {
          dispatch({
            type: "command-acked",
            requestId: message.request_id,
            error: message.ok ? undefined : message.error ?? "The server rejected that action",
          });
          return;
        }

        if (message.type === "session.error") {
          return;
        }

        if (message.type !== "room.projection") return;

        const incomingVersion = message.public.version;
        const currentVersion = projectionVersionRef.current;
        if (incomingVersion < currentVersion) return;

        if (resyncPending) {
          resyncPending = false;
        } else if (currentVersion >= 0 && incomingVersion > currentVersion + 1) {
          resyncPending = true;
          socket.resync();
          return;
        }

        projectionVersionRef.current = incomingVersion;
        latestPhase = message.public.phase;
        phaseRef.current = message.public.phase;
        dispatch({ type: "projection", projection: message });
        return;
      }

      socketRef.current = null;
      if (event.terminal) {
        if (event.status === 401 && latestPhase === "LOBBY") {
          endSession({ kind: "kicked", message: KICKED_MESSAGE, joinCode: identity.join_code });
          return;
        }
        endSession({
          kind: "session-expired",
          message: event.message ?? SESSION_EXPIRED_MESSAGE,
        });
        return;
      }

      scheduleReconnect(identity);
    });

    socketRef.current = socket;
  }

  function resumePersistedConnection(force = false): void {
    if (socketRef.current && !force) return;
    const identity = loadRoomIdentity();
    if (!identity) return;
    if (reconnectDeadlineRef.current !== null && Date.now() >= reconnectDeadlineRef.current) {
      endSession({ kind: "session-expired", message: OFFLINE_SESSION_EXPIRED_MESSAGE });
      return;
    }
    openConnection(identity, true);
  }

  useEffect(() => {
    const storedIdentity = loadRoomIdentity();
    if (storedIdentity) {
      // A reload can happen before the first projection arrives. Treat a
      // persisted session as a lobby until the server tells us otherwise so
      // a fast reload still sends the cleanup request for an empty lobby.
      phaseRef.current = "LOBBY";
      openConnection(storedIdentity, true);
    }

    const handleVisibilityChange = () => {
      if (document.visibilityState === "visible") resumePersistedConnection(true);
    };
    const handlePageShow = () => resumePersistedConnection(true);
    const handlePageHide = (event: PageTransitionEvent) => {
      if (event.persisted || phaseRef.current !== "LOBBY") return;
      const identity = loadRoomIdentity();
      if (identity) leaveRoomOnPageHide(identity);
    };

    document.addEventListener("visibilitychange", handleVisibilityChange);
    window.addEventListener("pageshow", handlePageShow);
    window.addEventListener("pagehide", handlePageHide);

    return () => {
      document.removeEventListener("visibilitychange", handleVisibilityChange);
      window.removeEventListener("pageshow", handlePageShow);
      window.removeEventListener("pagehide", handlePageHide);
      disconnect();
    };
    // Room connection resources intentionally live for the hook lifetime.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function create(playerName: string): Promise<void> {
    if (actionLockRef.current || state.status !== "idle") return;
    const name = playerName.trim();
    if (!name) {
      dispatch({ type: "error", message: "Enter your name first" });
      return;
    }

    actionLockRef.current = true;
    dispatch({ type: "request-started" });
    try {
      const identity = await createRoomRequest(name);
      projectionVersionRef.current = -1;
      phaseRef.current = "LOBBY";
      saveRoomIdentity(identity);
      openConnection(identity, false);
    } catch (cause) {
      dispatch({ type: "request-failed", message: errorMessage(cause, "Could not create the room") });
    } finally {
      actionLockRef.current = false;
    }
  }

  async function join(joinCode: string, playerName: string): Promise<void> {
    if (actionLockRef.current || state.status !== "idle") return;
    const name = playerName.trim();
    const code = joinCode.trim().toUpperCase();
    if (!name) {
      dispatch({ type: "error", message: "Enter your name first" });
      return;
    }
    if (!code) {
      dispatch({ type: "error", message: "Enter a room code first" });
      return;
    }

    actionLockRef.current = true;
    dispatch({ type: "request-started" });
    try {
      const identity = await joinRoomRequest(code, name);
      projectionVersionRef.current = -1;
      phaseRef.current = "LOBBY";
      saveRoomIdentity(identity);
      openConnection(identity, false);
    } catch (cause) {
      dispatch({ type: "request-failed", message: errorMessage(cause, "Could not join the room") });
    } finally {
      actionLockRef.current = false;
    }
  }

  async function leave(): Promise<void> {
    const identity = state.identity;
    if (!identity) {
      clearRoomIdentity();
      dispatch({ type: "left" });
      return;
    }

    dispatch({ type: "leave-started" });
    disconnect();
    projectionVersionRef.current = -1;
    phaseRef.current = null;
    clearRoomIdentity();

    try {
      await leaveRoomRequest(identity);
      dispatch({ type: "left" });
    } catch (cause) {
      dispatch({ type: "left", error: errorMessage(cause, "Could not notify the server that you left") });
    }
  }

  function send(command: ClientCommand): void {
    const socket = socketRef.current;
    const projection = state.projection;
    if (!socket || !projection || state.status !== "online") {
      dispatch({ type: "error", message: "Connect to a room before sending an action" });
      return;
    }
    if (state.pending) return;

    try {
      const requestId = socket.send(command, projection.public.version);
      dispatch({ type: "command-sent", pending: { requestId, kind: command.kind } });
    } catch (cause) {
      dispatch({ type: "error", message: errorMessage(cause, "The room connection is not ready") });
    }
  }

  function dismissNotice(): void {
    dispatch({ type: "notice-dismissed" });
  }

  return {
    identity: state.identity,
    projection: state.projection,
    status: state.status,
    error: state.error,
    notice: state.notice,
    pending: state.pending,
    create,
    join,
    leave,
    send,
    dismissNotice,
  };
}
