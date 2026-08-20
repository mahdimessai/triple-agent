"use client";

import { useState, type ReactNode } from "react";
import { useRoom } from "./use-room";
import { useRoomInvite } from "./invite/use-room-invite";
import { GameShell } from "./shell";
import type { RoomProjection } from "./protocol";
import { TitleScreen, type EntryMode } from "./screens/title";
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
  const roomInvite = useRoomInvite(room.identity, room.projection);
  const [entryMode, setEntryMode] = useState<EntryMode>("create");
  const [playerName, setPlayerName] = useState("");
  const [joinCode, setJoinCode] = useState("");
  const [settingsRoomId, setSettingsRoomId] = useState<string | null>(null);

  const settingsOpen = Boolean(room.identity && settingsRoomId === room.identity.room_id);
  const visibleEntryMode = room.notice?.kind === "kicked" ? "join" : entryMode;
  const visibleJoinCode = room.notice?.kind === "kicked"
    ? room.notice.joinCode
    : roomInvite.invite?.code ?? joinCode;

  function dismissInvite(): void {
    roomInvite.dismissInvite();
    setJoinCode("");
    setEntryMode("create");
  }

  function handleLeave() {
    const code = room.identity?.join_code ?? "";
    roomInvite.resetFeedback();
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
        invite={roomInvite.invite}
        onDismissInvite={dismissInvite}
        onModeChange={setEntryMode}
        onPlayerNameChange={setPlayerName}
        onJoinCodeChange={setJoinCode}
        onCreate={() => {
          roomInvite.resetFeedback();
          void room.create(playerName);
        }}
        onJoin={() => {
          roomInvite.resetFeedback();
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
  // Errors are reported next to whatever the player just touched. The shell
  // banner is only the fallback for phases that have no inline slot.
  const roomError = room.error ?? roomInvite.error;
  const errorShownInline = projection.public.phase === "LOBBY" && !settingsOpen;
  let content: ReactNode;
  if (settingsOpen) {
    content = <SettingsScreen projection={projection} pending={room.pending} onSend={room.send} onClose={() => setSettingsRoomId(null)} error={roomError} />;
  } else if (projection.public.phase === "LOBBY") {
    content = (
      <LobbyScreen
        projection={projection}
        joinCode={room.identity?.join_code ?? ""}
        copied={roomInvite.copiedRoomCode}
        pending={room.pending}
        onSend={room.send}
        onCopyRoomCode={() => void roomInvite.copyRoomCode()}
        onShareLink={() => void roomInvite.shareRoomLink()}
        linkShared={roomInvite.linkShared}
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
    >
      {content}
    </GameShell>
  );
}
