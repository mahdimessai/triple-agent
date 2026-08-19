"use client";

import { useEffect, useRef, useState, type ReactNode } from "react";
import { useRoom } from "./use-room";
import { GameShell } from "./shell";
import type { RoomProjection } from "./protocol";
import { TitleScreen, type EntryMode, type RoomInvite } from "./screens/title";
import { LobbyScreen } from "./screens/lobby";
import { SettingsScreen } from "./screens/settings";
import { RoleScreen } from "./screens/role";
import { OperationScreen } from "./screens/operation";
import { InterludeScreen } from "./screens/interlude";
import { DiscussionScreen } from "./screens/discussion";
import { AccusationScreen } from "./screens/accusation";
import { ResultsScreen } from "./screens/results";

function assertNever(value: never): never {
  throw new Error(`Unhandled phase: ${String(value)}`);
}

function renderPhase(projection: RoomProjection, room: ReturnType<typeof useRoom>): ReactNode {
  const props = { projection, pending: room.pending, onSend: room.send };
  const phase = projection.public.phase;
  switch (phase) {
    case "LOBBY":
      return null;
    case "ROLE_REVEAL":
      return <RoleScreen {...props} />;
    case "OPERATION_INPUT":
    case "OPERATION_RESULT":
      return <OperationScreen {...props} />;
    case "OPERATION_INTERLUDE":
      return <InterludeScreen {...props} />;
    case "DISCUSSION":
      return <DiscussionScreen {...props} />;
    case "VOTE_INPUT":
      return <AccusationScreen {...props} />;
    case "RESULTS_INTRO":
    case "VOTE_RESULTS":
    case "IMPRISONMENT_REVEAL":
    case "AGENCY_REVEAL":
    case "OUTCOME_REVEAL":
    case "LEADERBOARD":
    case "OUT_OF_LOOP":
    case "END":
      return <ResultsScreen {...props} />;
  }
  return assertNever(phase);
}

export function Game() {
  const room = useRoom();
  const [entryMode, setEntryMode] = useState<EntryMode>("create");
  const [playerName, setPlayerName] = useState("");
  const [joinCode, setJoinCode] = useState("");
  const [settingsRoomId, setSettingsRoomId] = useState<string | null>(null);
  const [copiedRoomCode, setCopiedRoomCode] = useState(false);
  const [linkShared, setLinkShared] = useState(false);
  const [invite, setInvite] = useState<RoomInvite | null>(null);
  const shareTimerRef = useRef<number | null>(null);
  const [copyError, setCopyError] = useState<string | null>(null);
  const copyTimerRef = useRef<number | null>(null);

  const settingsOpen = Boolean(room.identity && settingsRoomId === room.identity.room_id);
  const visibleEntryMode = room.notice?.kind === "kicked" ? "join" : entryMode;
  const visibleJoinCode = room.notice?.kind === "kicked" ? room.notice.joinCode : joinCode;

  useEffect(() => () => {
    if (copyTimerRef.current !== null) window.clearTimeout(copyTimerRef.current);
    if (shareTimerRef.current !== null) window.clearTimeout(shareTimerRef.current);
  }, []);

  /* Arriving on an invite link: the code comes from the URL and is never shown as a
     form to fill in. The host name is display-only text supplied by whoever shared
     the link, so it is sanitised and length-capped before it reaches the screen. */
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const code = (params.get("join") ?? "").toUpperCase().replace(/[^A-Z0-9]/g, "").slice(0, 6);
    if (code.length !== 6) return;
    const host = (params.get("host") ?? "").replace(/[^\p{L}\p{N} '._-]/gu, "").trim().slice(0, 24);
    setJoinCode(code);
    setInvite({ code, host: host || undefined });
  }, []);

  function dismissInvite(): void {
    setInvite(null);
    setJoinCode("");
    setEntryMode("create");
    window.history.replaceState(null, "", window.location.pathname);
  }

  /* The async clipboard API is not only sometimes absent, it also rejects when
     the document is not focused or the permission is denied. Both cases have to
     fall through to the selection-based copy, or the button reports failure for
     a copy the browser would happily have made. */
  function legacyCopy(code: string): boolean {
    try {
      const input = document.createElement("textarea");
      input.value = code;
      input.setAttribute("readonly", "true");
      input.style.position = "fixed";
      input.style.opacity = "0";
      document.body.appendChild(input);
      input.select();
      const copied = document.execCommand("copy");
      input.remove();
      return copied;
    } catch {
      return false;
    }
  }

  async function copyRoomCode(): Promise<void> {
    const code = room.identity?.join_code;
    if (!code) return;

    let copied = false;
    if (navigator.clipboard?.writeText) {
      try {
        await navigator.clipboard.writeText(code);
        copied = true;
      } catch {
        copied = false;
      }
    }
    if (!copied) copied = legacyCopy(code);

    if (!copied) {
      setCopiedRoomCode(false);
      setCopyError("Could not copy the room code");
      return;
    }

    setCopyError(null);
    setCopiedRoomCode(true);
    if (copyTimerRef.current !== null) window.clearTimeout(copyTimerRef.current);
    copyTimerRef.current = window.setTimeout(() => {
      copyTimerRef.current = null;
      setCopiedRoomCode(false);
    }, 1600);
  }

  function inviteUrl(code: string): string {
    const me = room.projection?.public.players.find((player) => player.id === room.projection?.private.player_id);
    const host = me?.name ? `&host=${encodeURIComponent(me.name)}` : "";
    return `${window.location.origin}${window.location.pathname}?join=${code}${host}`;
  }

  /* Where the platform offers a share sheet the link goes straight into it;
     everywhere else the fallback is to copy the link and say so on the button. */
  async function shareRoomLink(): Promise<void> {
    const code = room.identity?.join_code;
    if (!code) return;
    const url = inviteUrl(code);

    if (navigator.share) {
      try {
        await navigator.share({ title: "Triple Agent", text: `Join my Triple Agent room: ${code}`, url });
        return;
      } catch {
        // A cancelled or unavailable share sheet falls through to copying.
      }
    }

    let copied = false;
    if (navigator.clipboard?.writeText) {
      try {
        await navigator.clipboard.writeText(url);
        copied = true;
      } catch {
        copied = false;
      }
    }
    if (!copied) copied = legacyCopy(url);

    if (!copied) {
      setCopyError("Could not copy the invite link");
      return;
    }
    setCopyError(null);
    setLinkShared(true);
    if (shareTimerRef.current !== null) window.clearTimeout(shareTimerRef.current);
    shareTimerRef.current = window.setTimeout(() => {
      shareTimerRef.current = null;
      setLinkShared(false);
    }, 1600);
  }

  function handleLeave() {
    const code = room.identity?.join_code ?? "";
    setCopyError(null);
    setCopiedRoomCode(false);
    void room.leave().then(() => {
      setJoinCode(code);
      setEntryMode("join");
    });
  }

  if (!room.projection) {
    return (
      <TitleScreen
        mode={visibleEntryMode}
        playerName={playerName}
        joinCode={visibleJoinCode}
        busy={room.status === "connecting" || room.status === "reconnecting" || room.status === "leaving"}
        error={room.error}
        notice={room.notice}
        invite={invite}
        onDismissInvite={dismissInvite}
        onModeChange={setEntryMode}
        onPlayerNameChange={setPlayerName}
        onJoinCodeChange={setJoinCode}
        onCreate={() => {
          setCopyError(null);
          setCopiedRoomCode(false);
          void room.create(playerName);
        }}
        onJoin={() => {
          setCopyError(null);
          setCopiedRoomCode(false);
          void room.join(visibleJoinCode, playerName);
        }}
        onDismissNotice={() => {
          if (room.notice?.kind === "kicked") {
            setJoinCode(room.notice.joinCode);
            setEntryMode("join");
          }
          room.dismissNotice();
        }}
      />
    );
  }

  const projection = room.projection;
  /* Errors are reported next to whatever the player just touched. The shell
     banner is only the fallback for phases that have no inline slot, so an
     error never shows up twice. */
  const roomError = room.error ?? copyError;
  const errorShownInline = projection.public.phase === "LOBBY" && !settingsOpen;
  let content: ReactNode;
  if (settingsOpen) {
    content = <SettingsScreen projection={projection} pending={room.pending} onSend={room.send} onClose={() => setSettingsRoomId(null)} error={roomError} />;
  } else if (projection.public.phase === "LOBBY") {
    content = (
      <LobbyScreen
        projection={projection}
        joinCode={room.identity?.join_code ?? ""}
        copied={copiedRoomCode}
        pending={room.pending}
        onSend={room.send}
        onCopyRoomCode={() => void copyRoomCode()}
        onShareLink={() => void shareRoomLink()}
        linkShared={linkShared}
        error={roomError}
      />
    );
  } else {
    content = renderPhase(projection, room);
  }

  return (
    <GameShell
      projection={projection}
      status={room.status}
      error={errorShownInline ? null : roomError}
      settingsOpen={settingsOpen}
      onToggleSettings={() => setSettingsRoomId(settingsOpen ? null : room.identity?.room_id ?? null)}
      onLeave={handleLeave}
      onHome={() => {
        if (settingsOpen) {
          setSettingsRoomId(null);
          return;
        }
        if (projection.public.phase === "LOBBY") {
          if (window.confirm("Leave this lobby? Your seat will be given up.")) {
            handleLeave();
          }
        }
      }}
    >
      {content}
    </GameShell>
  );
}
