import type { ReactNode } from "react";
import { AccusationScreen } from "@/features/triple-agent/screens/accusation-screen";
import { DiscussionScreen } from "@/features/triple-agent/screens/discussion-screen";
import { LobbyScreen } from "@/features/triple-agent/screens/lobby-screen";
import { MissionScreen } from "@/features/triple-agent/screens/mission-screen";
import { OperationScreen } from "@/features/triple-agent/screens/operation-screen";
import { ResultsScreen } from "@/features/triple-agent/screens/results-screen";
import { RoleScreen } from "@/features/triple-agent/screens/role-screen";
import { TitleScreen } from "@/features/triple-agent/screens/title-screen";
import { getOperation, type OperationId } from "@/components/triple-agent/operation-catalog";
import type { Faction, Phase, RoomProjection } from "@/components/triple-agent/server-client";
import type { MockFixture } from "@/components/triple-agent/mock/mock-fixtures";
import type { MockScreenDefinition } from "@/components/triple-agent/mock/mock-screen-registry";

const noop = () => {};
const noopString = (_value: string) => {};
const noopOperation = (_operationId: OperationId) => {};

function fixtureProjection(fixture: MockFixture, phase: Phase, operationId?: OperationId): RoomProjection {
  const players = fixture.players.map((player) => ({
    id: player.id,
    name: player.name,
    seat: player.seat,
    ready: player.ready,
    connected: player.connected,
    vote_submitted: false,
  }));
  const host = fixture.players.find((player) => player.isHost) ?? fixture.players[0];
  const currentPlayer = fixture.players.find((player) => player.name === fixture.playerName) ?? fixture.players[0];
  const currentFaction: Faction = fixture.virusTeammates.includes(currentPlayer.name) ? "VIRUS" : "SERVICE";
  const selectedOperation = operationId ? getOperation(operationId) : undefined;
  const operationInputKind = selectedOperation
    ? selectedOperation.category === "choice"
      ? selectedOperation.input.includes("two")
        ? "TWO_TARGETS"
        : selectedOperation.input === "Defect or stay"
          ? "CHOICE"
          : "ONE_TARGET"
      : selectedOperation.category === "information"
        ? "PRIVATE_INFO"
        : "NONE"
    : undefined;
  const playerIDByName = new Map(players.map((player) => [player.name, player.id]));
  const voteTotals = Object.fromEntries(
    fixture.results.voteTotals.map((entry) => [playerIDByName.get(entry.player) ?? entry.player, entry.votes]),
  );
  const leaderboard = fixture.results.leaderboard.map((entry) => ({
    player_id: playerIDByName.get(entry.player) ?? entry.player,
    name: entry.player,
    faction: (entry.faction === "VIRUS" ? "VIRUS" : "SERVICE") as Faction,
    role: entry.role,
    defection: entry.defection,
    votes: fixture.results.voteTotals.find((vote) => vote.player === entry.player)?.votes ?? 0,
    result: entry.faction === fixture.results.winner ? "WINNER" as const : "LOSER" as const,
  }));

  return {
    type: "room.projection",
    public: {
      room_id: fixture.roomCode,
      host_id: host.id,
      phase,
      version: 1,
      players,
      settings: {
        discussion_timer_enabled: true,
        discussion_seconds: 102,
        enabled_operations: ["Swap", "Detector", "Share", "OneRandom"],
      },
      active_player_id: currentPlayer.id,
      operation: selectedOperation
        ? {
            kind: selectedOperation.id,
            name: selectedOperation.name,
            input_kind: operationInputKind ?? "NONE",
            target_count: operationInputKind === "TWO_TARGETS" ? 2 : operationInputKind === "ONE_TARGET" ? 1 : undefined,
            active_player_id: currentPlayer.id,
            active_player_name: currentPlayer.name,
            public_instruction: selectedOperation.publicUpdate,
          }
        : undefined,
      vote_totals: voteTotals,
      imprisoned_player_id: playerIDByName.get(fixture.results.imprisonedPlayer),
      revealed_faction: fixture.results.imprisonedFaction,
      winner: fixture.results.winner,
      leaderboard,
      activity: fixture.activity.map((entry) => `${entry.label}: ${entry.detail}`).join(" · "),
    },
    private: {
      player_id: currentPlayer.id,
      role: currentFaction,
      initial_faction: currentFaction,
      faction: currentFaction,
      apparent_faction: currentFaction,
      operation_result: {
        target_player_id: playerIDByName.get(fixture.anonymousTip.target),
        target_faction: fixture.anonymousTip.targetFaction,
        your_faction: currentFaction,
        message: fixture.anonymousTip.privateResult,
      },
      operation_instruction: selectedOperation?.privatePrompt,
      legal_target_ids: players.filter((player) => player.id !== currentPlayer.id).map((player) => player.id),
      choices: ["STAY", "DEFECT"],
      vote_submitted: false,
      can_submit: true,
    },
  };
}

function operationForRender(render: MockScreenDefinition["render"]): OperationId {
  switch (render) {
    case "hidden-agenda":
      return "HiddenAgenda";
    case "anonymous-description":
    case "anonymous-private-result":
    case "anonymous-public-explanation":
      return "OneRandom";
    case "operations-intro":
    case "operation-waiting":
    default:
      return "OneRandom";
  }
}

function renderOperation(render: MockScreenDefinition["render"]) {
  const operationId = operationForRender(render);
  return <OperationScreen operationId={operationId} onNext={noop} />;
}

function renderResults(fixture: MockFixture, phase: Phase) {
  return <ResultsScreen projection={fixtureProjection(fixture, phase)} onRestart={noop} />;
}

/**
 * The workbench owns only fixture selection. Every game screen below is the
 * same component used by the live client, with deterministic fixture data and
 * no-op actions standing in for server commands.
 */
export function renderMockScreen(definition: MockScreenDefinition, fixture: MockFixture): ReactNode {
  switch (definition.render) {
    case "title":
      return <TitleScreen playerName="AGENT A" setPlayerName={noopString} roomCode="" setRoomCode={noopString} joining={false} onStart={noop} onJoin={noop} onOpenJoin={noop} onCancelJoin={noop} />;
    case "setup":
      return (
        <LobbyScreen
          livePlayers={fixtureProjection(fixture, "LOBBY").public.players}
          hostId={fixture.players.find((player) => player.isHost)?.id}
          liveSession
          isHost
          canReady={false}
          onReady={noop}
          onStart={noop}
        />
      );
    case "briefing":
      return <MissionScreen onNext={noop} />;
    case "role-service":
      return <RoleScreen faction="SERVICE" onNext={noop} />;
    case "role-virus":
      return <RoleScreen faction="VIRUS" onNext={noop} />;
    case "operations-intro":
    case "hidden-agenda":
    case "operation-waiting":
    case "anonymous-description":
    case "anonymous-private-result":
    case "anonymous-public-explanation":
      return renderOperation(definition.render);
    case "discussion":
      return <DiscussionScreen timerEnabled canAdvance projection={fixtureProjection(fixture, "DISCUSSION")} onNext={noop} />;
    case "accusation-guard":
    case "accusation-input":
      return <AccusationScreen projection={fixtureProjection(fixture, "VOTE_INPUT")} onNext={noop} />;
    case "results-intro":
      return renderResults(fixture, "RESULTS_INTRO");
    case "vote-results":
      return renderResults(fixture, "VOTE_RESULTS");
    case "imprisonment-reveal":
      return renderResults(fixture, "IMPRISONMENT_REVEAL");
    case "agency-reveal":
      return renderResults(fixture, "AGENCY_REVEAL");
    case "outcome-reveal":
      return renderResults(fixture, "OUTCOME_REVEAL");
    case "leaderboard":
      return renderResults(fixture, "LEADERBOARD");
    case "out-of-loop":
      return renderResults(fixture, "OUT_OF_LOOP");
    case "end":
      return renderResults(fixture, "END");
    case "planned":
      return <p className="ta-condensed p-5 text-lg">This fixture phase is not modeled by the production client yet.</p>;
  }
}
