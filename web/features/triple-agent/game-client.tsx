"use client";

import dynamic from "next/dynamic";
import { useEffect, useRef, useState } from "react";
import type { OperationId } from "@/components/triple-agent/operation-catalog";
import { InkButton } from "@/components/ui/ink-button";
import { GameShell } from "@/features/triple-agent/components/game-shell";
import { TitleScreen } from "@/features/triple-agent/screens/title-screen";
import { operationIdForServerKind, type CommandPayload, type CommandSender, type ScreenId } from "@/features/triple-agent/model/screen";
import type { ClientCommand, RoomProjection } from "@/features/triple-agent/protocol/types";
import { useRoomSession } from "@/features/triple-agent/session/use-room-session";

function ScreenLoading() {
  return <div className="ta-rise ta-screen items-center justify-center text-center"><p className="ta-condensed">Loading…</p></div>;
}

const AccusationScreen = dynamic(() => import("@/features/triple-agent/screens/accusation-screen").then((module) => module.AccusationScreen), { loading: ScreenLoading });
const DiscussionScreen = dynamic(() => import("@/features/triple-agent/screens/discussion-screen").then((module) => module.DiscussionScreen), { loading: ScreenLoading });
const InterludeScreen = dynamic(() => import("@/features/triple-agent/screens/interlude-screen").then((module) => module.InterludeScreen), { loading: ScreenLoading });
const LobbyScreen = dynamic(() => import("@/features/triple-agent/screens/lobby-screen").then((module) => module.LobbyScreen), { loading: ScreenLoading });
const MissionScreen = dynamic(() => import("@/features/triple-agent/screens/mission-screen").then((module) => module.MissionScreen), { loading: ScreenLoading });
const OperationScreen = dynamic(() => import("@/features/triple-agent/screens/operation-screen").then((module) => module.OperationScreen), { loading: ScreenLoading });
const ResultsScreen = dynamic(() => import("@/features/triple-agent/screens/results-screen").then((module) => module.ResultsScreen), { loading: ScreenLoading });
const RoleScreen = dynamic(() => import("@/features/triple-agent/screens/role-screen").then((module) => module.RoleScreen), { loading: ScreenLoading });
const SettingsScreen = dynamic(() => import("@/features/triple-agent/screens/settings-screen").then((module) => module.SettingsScreen), { loading: ScreenLoading });

function ScreenContent({ screen, playerName, setPlayerName, roomCode, setRoomCode, roomCodeCopied, copyRoomCode, timerEnabled, setTimerEnabled, navigate, operationId, setOperationId, projection, liveSession, onCommand, pendingCommand, roomAction, onCreateRoom, onJoinRoom, onLeave, leaving, error }: {
  screen: ScreenId;
  playerName: string;
  setPlayerName: (value: string) => void;
  roomCode: string;
  setRoomCode: (value: string) => void;
  roomCodeCopied: boolean;
  copyRoomCode: () => void;
  timerEnabled: boolean;
  setTimerEnabled: (value: boolean) => void;
  navigate: (screen: ScreenId) => void;
  operationId: OperationId;
  setOperationId: (id: OperationId) => void;
  projection?: RoomProjection;
  liveSession?: boolean;
  onCommand?: CommandSender;
  pendingCommand?: ClientCommand["kind"];
  roomAction?: "creating" | "joining";
  onCreateRoom: () => void;
  onJoinRoom: () => void;
  onLeave: () => void;
  leaving: boolean;
  error?: string;
}) {
  switch (screen) {
    case "title":
    case "join": return <TitleScreen playerName={playerName} setPlayerName={setPlayerName} roomCode={roomCode} setRoomCode={setRoomCode} joining={screen === "join"} roomAction={roomAction} onStart={onCreateRoom} onJoin={onJoinRoom} onOpenJoin={() => navigate("join")} onCancelJoin={() => navigate("title")} error={error} />;
    case "lobby": return <LobbyScreen roomCode={roomCode} roomCodeCopied={roomCodeCopied} copyRoomCode={copyRoomCode} livePlayers={projection?.public.players} hostId={projection?.public.host_id} selfId={projection?.private.player_id} minPlayers={projection?.public.settings.min_players} liveSession={liveSession} isHost={projection?.public.host_id === projection?.private.player_id} canReady={Boolean(projection)} isReady={projection?.public.players.find((player) => player.id === projection.private.player_id)?.ready} readyLoading={pendingCommand === "lobby.ready"} startLoading={pendingCommand === "match.start"} onReady={() => onCommand?.("lobby.ready")} onStart={() => projection ? onCommand?.("match.start") : navigate("mission")} onTransferHost={(targetId) => onCommand?.("lobby.transfer_host", { targetId })} onKickPlayer={(targetId) => onCommand?.("lobby.kick_player", { targetId })} onLeave={onLeave} leaving={leaving} error={error} />;
    case "settings": return <SettingsScreen timerEnabled={timerEnabled} setTimerEnabled={setTimerEnabled} projection={projection} liveSession={liveSession} isHost={projection?.public.host_id === projection?.private.player_id} onCommand={onCommand} pending={Boolean(pendingCommand)} error={error} />;
    case "mission": return <MissionScreen onNext={() => navigate("role")} />;
    case "role": return <RoleScreen faction={projection?.private.faction} roleId={projection?.private.role} roleName={projection?.private.role_name} roleDescription={projection?.private.role_description} roleEffect={projection?.private.role_effect} virusRoster={projection?.private.virus_roster} virusTeamSize={projection?.private.virus_team_size} canSubmit={projection?.private.can_submit} waitingOn={projection?.public.pending_role_acks} loading={pendingCommand === "role.acknowledge"} onNext={() => projection ? onCommand?.("role.acknowledge") : navigate("operation")} />;
    case "operation": return <OperationScreen key={`${operationId}:${projection?.public.operation?.kind ?? ""}:${projection?.public.phase ?? ""}`} operationId={operationId} projection={projection} loading={pendingCommand === "operation.resolve" || pendingCommand === "operation.explain_done"} onNext={(targetIds, choice) => { if (!projection) navigate("discussion"); else if (projection.public.phase === "OPERATION_INPUT") onCommand?.("operation.resolve", { targetIds, choice }); else if (projection.public.phase === "OPERATION_RESULT") onCommand?.("operation.explain_done"); }} />;
    case "interlude": return <InterludeScreen deadline={projection?.public.discussion_deadline} seconds={projection?.public.settings.interlude_seconds} isHost={projection?.public.host_id === projection?.private.player_id} loading={pendingCommand === "interlude.advance"} onSkip={() => projection ? onCommand?.("interlude.advance") : navigate("discussion")} />;
    case "discussion": return <DiscussionScreen timerEnabled={timerEnabled} projection={projection} canAdvance={!projection || projection.private.can_submit} loading={pendingCommand === "discussion.advance"} onNext={() => projection ? onCommand?.("discussion.advance") : navigate("accusation")} />;
    case "accusation": return <AccusationScreen projection={projection} loading={pendingCommand === "vote.submit"} onNext={(targetId) => projection ? onCommand?.("vote.submit", { targetId }) : navigate("results")} />;
    case "results": return <ResultsScreen projection={projection} loading={pendingCommand === "match.rematch"} onRestart={() => projection ? onCommand?.("match.rematch") : navigate("lobby")} />;
  }
}

export function GameClient() {
  const [screen, setScreen] = useState<ScreenId>("title");
  const [playerName, setPlayerName] = useState("");
  const [operationId, setOperationId] = useState<OperationId>("OneRandom");
  const [roomCodeCopied, setRoomCodeCopied] = useState(false);
  const roomCodeCopiedTimerRef = useRef<number | undefined>(undefined);
  const {
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
    kickedFromLobby,
    dismissKickedPopup,
    createRoom,
    joinRoom,
    leaveRoom,
    leaving,
    sendCommand,
    reportError,
  } = useRoomSession({ screen, onScreenChange: setScreen });

  useEffect(() => () => {
    if (roomCodeCopiedTimerRef.current !== undefined) window.clearTimeout(roomCodeCopiedTimerRef.current);
  }, []);

  async function copyRoomCode() {
    if (!roomCode) return;
    try {
      let copied = false;
      if (navigator.clipboard?.writeText) {
        try {
          await navigator.clipboard.writeText(roomCode);
          copied = true;
        } catch {
          copied = false;
        }
      }
      if (!copied) {
        const input = document.createElement("textarea");
        input.value = roomCode;
        input.setAttribute("readonly", "true");
        input.style.position = "fixed";
        input.style.opacity = "0";
        document.body.appendChild(input);
        input.select();
        document.execCommand("copy");
        input.remove();
      }
      setRoomCodeCopied(true);
      if (roomCodeCopiedTimerRef.current !== undefined) window.clearTimeout(roomCodeCopiedTimerRef.current);
      roomCodeCopiedTimerRef.current = window.setTimeout(() => {
        roomCodeCopiedTimerRef.current = undefined;
        setRoomCodeCopied(false);
      }, 1600);
    } catch {
      reportError("Could not copy the room code");
    }
  }

  const liveOperationId = operationIdForServerKind(projection?.public.operation?.kind);
  return (
      <GameShell screen={screen} setScreen={setScreen} session={session} connectionState={connectionState} reconnecting={reconnecting}>
        <ScreenContent
            screen={screen}
            playerName={playerName}
            setPlayerName={setPlayerName}
            roomCode={roomCode}
            setRoomCode={setRoomCode}
            roomCodeCopied={roomCodeCopied}
            copyRoomCode={() => void copyRoomCode()}
            timerEnabled={timerEnabled}
            setTimerEnabled={setTimerEnabled}
            navigate={setScreen}
            operationId={liveOperationId}
            setOperationId={setOperationId}
            projection={projection}
            liveSession={Boolean(session)}
            onCommand={sendCommand}
            pendingCommand={pendingCommand}
            roomAction={roomAction}
            onCreateRoom={() => void createRoom(playerName)}
            onJoinRoom={() => void joinRoom(playerName)}
            onLeave={() => void leaveRoom()}
            leaving={leaving}
            error={error}
        />
        {kickedFromLobby ? (
            <div className="ta-join-backdrop" role="presentation" onClick={dismissKickedPopup}>
              <div
                  className="ta-paper ta-join-dialog text-center"
                  role="dialog"
                  aria-modal="true"
                  aria-labelledby="kicked-dialog-title"
                  onClick={(e) => e.stopPropagation()}
              >
                <p className="ta-condensed text-xs tracking-[0.2em] text-black/60">NOTICE</p>
                <h2 id="kicked-dialog-title" className="ta-display mt-1 text-3xl text-ta-red">
                  Kicked from Lobby
                </h2>
                <p className="ta-condensed mt-4 text-base leading-tight">
                  You have been removed from the lobby by the host.
                </p>
                <div className="mt-6">
                  <InkButton variant="orange" className="w-full" onClick={dismissKickedPopup}>
                    Understood
                  </InkButton>
                </div>
              </div>
            </div>
        ) : null}
      </GameShell>
  );
}