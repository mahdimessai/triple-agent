"use client";

import { useEffect, useRef, useState } from "react";
import { createLobby, joinLobby, leaveLobby } from "../api/lobbies";
import { connectRoom, type RoomConnection } from "../connection/room-connection";
import {
  reconnectDeadline,
  reconnectDelay,
  RECONNECT_GRACE_PERIOD_MS,
} from "../connection/reconnect-policy";
import { clientCommandFromLegacy, type CommandPayload, type ScreenId } from "../model/screen";
import type { ClientCommand, LobbySession, Phase, RoomProjection } from "../protocol";
import { clearLobbySession, loadLobbySession, saveLobbySession } from "./session-storage";

const SESSION_EXPIRED_MESSAGE = "This room session has expired";
const OFFLINE_SESSION_EXPIRED_MESSAGE = "The room session expired after five minutes offline";
type RoomAction = "creating" | "joining";
type PendingCommand = { requestId: string; kind: ClientCommand["kind"] };

function screenForPhase(phase: Phase): ScreenId {
  switch (phase) {
    case "LOBBY": return "lobby";
    case "ROLE_REVEAL": return "role";
    case "OPERATION_INPUT":
    case "OPERATION_RESULT": return "operation";
    case "OPERATION_INTERLUDE": return "interlude";
    case "DISCUSSION": return "discussion";
    case "VOTE_INPUT": return "accusation";
    case "RESULTS_INTRO":
    case "VOTE_RESULTS":
    case "IMPRISONMENT_REVEAL":
    case "AGENCY_REVEAL":
    case "OUTCOME_REVEAL":
    case "LEADERBOARD":
    case "OUT_OF_LOOP":
    case "END": return "results";
  }
}

export type UseRoomSessionOptions = {
  screen: ScreenId;
  onScreenChange: (screen: ScreenId) => void;
};

export function useRoomSession({ screen, onScreenChange }: UseRoomSessionOptions) {
  const screenRef = useRef(screen);
  const [roomCode, setRoomCode] = useState("");
  const [timerEnabled, setTimerEnabled] = useState(false);
  const [session, setSession] = useState<LobbySession>();
  const [projection, setProjection] = useState<RoomProjection>();
  const [connectionState, setConnectionState] = useState<"connecting" | "open" | "closed">("closed");
  const [reconnecting, setReconnecting] = useState(false);
  const [error, setError] = useState<string>();
  const [leaving, setLeaving] = useState(false);
  const [roomAction, setRoomAction] = useState<RoomAction>();
  const [pendingCommand, setPendingCommand] = useState<ClientCommand["kind"]>();
  const socketRef = useRef<RoomConnection | null>(null);
  const sessionRef = useRef<LobbySession | undefined>(undefined);
  const roomActionRef = useRef<RoomAction | undefined>(undefined);
  const pendingCommandRef = useRef<PendingCommand | undefined>(undefined);
  const connectionAttemptRef = useRef(0);
  const reconnectTimerRef = useRef<number | undefined>(undefined);
  const reconnectAttemptRef = useRef(0);
  const reconnectDeadlineRef = useRef<number | undefined>(undefined);
  const phaseRef = useRef<Phase | undefined>(undefined);

  useEffect(() => {
    screenRef.current = screen;
  }, [screen]);

  function cancelReconnect() {
    if (reconnectTimerRef.current !== undefined) {
      window.clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = undefined;
    }
  }

  function clearRoomAction() {
    roomActionRef.current = undefined;
    setRoomAction(undefined);
  }

  function clearPendingCommand() {
    pendingCommandRef.current = undefined;
    setPendingCommand(undefined);
  }

  function closeSessionSocket(updateState = true) {
    connectionAttemptRef.current++;
    cancelReconnect();
    reconnectDeadlineRef.current = undefined;
    socketRef.current?.close();
    socketRef.current = null;
    clearPendingCommand();
    setReconnecting(false);
    if (updateState) setConnectionState("closed");
  }

  function expireSession(message = SESSION_EXPIRED_MESSAGE) {
    connectionAttemptRef.current++;
    cancelReconnect();
    reconnectDeadlineRef.current = undefined;
    socketRef.current?.close();
    socketRef.current = null;
    reconnectAttemptRef.current = 0;
    phaseRef.current = undefined;
    clearRoomAction();
    clearPendingCommand();
    sessionRef.current = undefined;
    clearLobbySession();
    setSession(undefined);
    setProjection(undefined);
    setRoomCode("");
    setConnectionState("closed");
    setError(message);
    onScreenChange("title");
  }

  function releaseLobbySession(nextSession: LobbySession) {
    connectionAttemptRef.current++;
    cancelReconnect();
    reconnectDeadlineRef.current = undefined;
    reconnectAttemptRef.current = 0;
    phaseRef.current = undefined;
    clearRoomAction();
    clearPendingCommand();
    sessionRef.current = undefined;
    clearLobbySession();
    setSession(undefined);
    setProjection(undefined);
    setRoomCode(nextSession.join_code);
    setConnectionState("closed");
    setReconnecting(false);
    setError("Your lobby seat was released; join again to return to the room.");
    onScreenChange("join");
  }

  function openSessionSocket(nextSession: LobbySession) {
    const attempt = ++connectionAttemptRef.current;
    const connection = connectRoom(nextSession, (nextProjection) => {
      if (attempt !== connectionAttemptRef.current) return;
      if (roomActionRef.current) clearRoomAction();
      setProjection(nextProjection);
      phaseRef.current = nextProjection.public.phase;
      setTimerEnabled(nextProjection.public.settings.discussion_timer_enabled);
      const nextScreen = screenRef.current === "settings" && nextProjection.public.phase === "LOBBY"
        ? "settings"
        : screenForPhase(nextProjection.public.phase);
      onScreenChange(nextScreen);
    }, (requestId, ok, commandError) => {
      if (attempt !== connectionAttemptRef.current) return;
      if (pendingCommandRef.current?.requestId !== requestId) return;
      clearPendingCommand();
      if (!ok) setError(commandError ?? "The server rejected that action");
      else setError(undefined);
    }, (state, details) => {
      if (attempt !== connectionAttemptRef.current) return;
      setConnectionState(state);
      if (state === "open") {
        setReconnecting(false);
        reconnectAttemptRef.current = 0;
        reconnectDeadlineRef.current = undefined;
        cancelReconnect();
      } else if (state === "closed") {
        clearPendingCommand();
        socketRef.current = null;
        if (phaseRef.current === "LOBBY") releaseLobbySession(nextSession);
        else if (details?.terminal) expireSession(details.message ?? SESSION_EXPIRED_MESSAGE);
        else scheduleReconnect(nextSession);
      }
    });
    socketRef.current = connection;
  }

  function scheduleReconnect(nextSession: LobbySession) {
    if (reconnectTimerRef.current !== undefined) return;
    if (sessionRef.current?.room_id !== nextSession.room_id) return;
    setReconnecting(true);
    // Reconnect scheduling runs from a socket event, not during render.
    // eslint-disable-next-line react-hooks/purity
    const now = Date.now();
    const deadline = reconnectDeadline(now, reconnectDeadlineRef.current, RECONNECT_GRACE_PERIOD_MS);
    reconnectDeadlineRef.current = deadline;
    const remaining = deadline - now;
    if (remaining <= 0) {
      expireSession(OFFLINE_SESSION_EXPIRED_MESSAGE);
      return;
    }
    const attempt = reconnectAttemptRef.current++;
    reconnectTimerRef.current = window.setTimeout(() => {
      reconnectTimerRef.current = undefined;
      if (Date.now() >= deadline) {
        expireSession(OFFLINE_SESSION_EXPIRED_MESSAGE);
        return;
      }
      if (sessionRef.current?.room_id === nextSession.room_id && !socketRef.current) {
        openSessionSocket(nextSession);
      }
    }, Math.min(reconnectDelay(attempt), remaining));
  }

  function connectSession(nextSession: LobbySession) {
    sessionRef.current = nextSession;
    phaseRef.current = undefined;
    saveLobbySession(nextSession);
    reconnectAttemptRef.current = 0;
    reconnectDeadlineRef.current = undefined;
    setReconnecting(false);
    clearPendingCommand();
    closeSessionSocket();
    setSession(nextSession);
    setRoomCode(nextSession.join_code);
    setProjection(undefined);
    setError(undefined);
    openSessionSocket(nextSession);
  }

  useEffect(() => {
    let disposed = false;
    const ensureSocketConnected = () => {
      if (reconnectDeadlineRef.current !== undefined && Date.now() >= reconnectDeadlineRef.current) {
        expireSession(OFFLINE_SESSION_EXPIRED_MESSAGE);
        return;
      }
      if (sessionRef.current && !socketRef.current) {
        cancelReconnect();
        openSessionSocket(sessionRef.current);
      }
    };
    const handleVisibilityChange = () => {
      if (document.visibilityState === "visible") ensureSocketConnected();
    };
    const handlePageShow = () => ensureSocketConnected();

    document.addEventListener("visibilitychange", handleVisibilityChange);
    window.addEventListener("pageshow", handlePageShow);

    const storedSession = loadLobbySession();
    if (storedSession) {
      window.setTimeout(() => {
        if (disposed) return;
        sessionRef.current = storedSession;
        setSession(storedSession);
        setRoomCode(storedSession.join_code);
        ensureSocketConnected();
      }, 0);
    }
    return () => {
      disposed = true;
      document.removeEventListener("visibilitychange", handleVisibilityChange);
      window.removeEventListener("pageshow", handlePageShow);
      closeSessionSocket();
    };
    // The session callbacks intentionally close over the one-time lifecycle handlers above.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function createRoom(playerName = "") {
    if (roomActionRef.current) return;
    const name = playerName.trim();
    if (!name) {
      setError("Enter your name first");
      return;
    }
    roomActionRef.current = "creating";
    setRoomAction("creating");
    try {
      connectSession(await createLobby(name));
    } catch (cause) {
      clearRoomAction();
      setError(cause instanceof Error ? cause.message : "Could not create the room");
    }
  }

  async function joinRoom(playerName = "") {
    if (roomActionRef.current) return;
    const name = playerName.trim();
    if (!name) {
      setError("Enter your name first");
      return;
    }
    const code = roomCode.trim().toUpperCase();
    if (!code) {
      setError("Enter a room code first");
      return;
    }
    roomActionRef.current = "joining";
    setRoomAction("joining");
    try {
      connectSession(await joinLobby(code, name));
    } catch (cause) {
      clearRoomAction();
      setError(cause instanceof Error ? cause.message : "Could not join the room");
    }
  }

  async function leaveRoom() {
    const currentSession = sessionRef.current;
    if (!currentSession) {
      onScreenChange("title");
      return;
    }
    setLeaving(true);
    try {
      await leaveLobby(currentSession);
      closeSessionSocket();
      sessionRef.current = undefined;
      clearLobbySession();
      setSession(undefined);
      setProjection(undefined);
      // Keep the code in the join form so a former host can immediately join
      // the lobby again as a new, ordinary player.
      setRoomCode(currentSession.join_code);
      setError(undefined);
      onScreenChange("join");
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Could not leave the room");
    } finally {
      setLeaving(false);
    }
  }

  function sendCommand(command: ClientCommand): void;
  function sendCommand(kind: ClientCommand["kind"], payload?: CommandPayload): void;
  function sendCommand(commandOrKind: ClientCommand | ClientCommand["kind"], payload?: CommandPayload) {
    const command = typeof commandOrKind === "string"
      ? clientCommandFromLegacy(commandOrKind, payload)
      : commandOrKind;
    if (!projection || !socketRef.current) {
      setError("Connect to a room before sending an action");
      return;
    }
    if (pendingCommandRef.current) return;
    try {
      const requestId = socketRef.current.send(command, projection.public.version);
      pendingCommandRef.current = { requestId, kind: command.kind };
      setPendingCommand(command.kind);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "The room connection is not ready");
    }
  }

  function reportError(message: string) {
    setError(message);
  }

  return {
    roomCode,
    setRoomCode,
    timerEnabled,
    setTimerEnabled,
    session,
    projection,
    connectionState,
    reconnecting,
    error,
    roomAction,
    pendingCommand,
    createRoom,
    joinRoom,
    leaveRoom,
    leaving,
    sendCommand,
    reportError,
  };
}
