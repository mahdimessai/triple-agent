"use client";

import { useState, type ReactNode } from "react";
import { useRoomInvite } from "./invite/use-room-invite";
import type { RoomProjection } from "./protocol";
import { AccusationScreen } from "./screens/accusation";
import { DiscussionScreen } from "./screens/discussion";
import { InterludeScreen } from "./screens/interlude";
import { LobbyScreen } from "./screens/lobby";
import { OperationScreen } from "./screens/operation";
import { ResultsScreen } from "./screens/results";
import { RoleScreen } from "./screens/role";
import { SettingsScreen } from "./screens/settings";
import { TitleScreen, type EntryMode } from "./screens/title";
import { GameShell } from "./shell";
import { useRoom } from "./use-room";

function assertNever(value: never): never {
  throw new Error(`Unhandled phase: ${String(value)}`);
}

function renderGamePhase(projection: RoomProjection, room: ReturnType<typeof useRoom>): ReactNode {
  const props = { projection, pending: room.pending, onSend: room.send };
  switch (projection.public.phase) {
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
  return assertNever(projection.public.phase);
}

export function Game() {
  const room = useRoom();
  const [entryMode, setEntryMode] = useState<EntryMode>("create");
  const [playerName, setPlayerName] = useState("");
  const [joinCode, setJoinCode] = useState("");
  const [settingsRoomId, setSettingsRoomId] = useState<string | null>(null);
  const invite = useRoomInvite({ room, setJoinCode });

  const settingsOpen = Boolean(room.identity && settingsRoomId === room.identity.room_id);
  const visibleEntryMode = room.notice?.kind === "kicked" ? "join" : entryMode;
  const visibleJoinCode = room.notice?.kind === "kicked" ? room.notice.joinCode : joinCode;

  function handleLeave(): void {
    const code = room.identity?.join_code ?? "";
    invite.resetFeedback();
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
        invite={invite.invite}
        onDismissInvite={() => {
          invite.dismissInvite();
          setEntryMode("create");
        }}
        onModeChange={setEntryMode}
        onPlayerNameChange={setPlayerName}
        onJoinCodeChange={setJoinCode}
        onCreate={() => {
          invite.resetFeedback();
          void room.create(playerName);
        }}
        onJoin={() => {
          invite.resetFeedback();
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
  const roomError = room.error ?? invite.copyError;
  const errorShownInline = projection.public.phase === "LOBBY" && !settingsOpen;
  let content: ReactNode;

  if (settingsOpen) {
    content = (
      <SettingsScreen
        projection={projection}
        pending={room.pending}
        onSend={room.send}
        onClose={() => setSettingsRoomId(null)}
        error={roomError}
      />
    );
  } else if (projection.public.phase === "LOBBY") {
    content = (
      <LobbyScreen
        projection={projection}
        joinCode={room.identity?.join_code ?? ""}
        copied={invite.copiedRoomCode}
        pending={room.pending}
        onSend={room.send}
        onCopyRoomCode={() => void invite.copyRoomCode()}
        onShareLink={() => void invite.shareRoomLink()}
        linkShared={invite.linkShared}
        error={roomError}
      />
    );
  } else {
    content = renderGamePhase(projection, room);
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
