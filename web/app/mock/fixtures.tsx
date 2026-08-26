"use client";

import { useMemo, useState } from "react";
import type { ClientCommand, Phase, RoomProjection } from "@/features/triple-agent/protocol";
import { LobbyScreen } from "@/features/triple-agent/screens/lobby";
import { SettingsScreen } from "@/features/triple-agent/screens/settings";
import { RoleScreen } from "@/features/triple-agent/screens/role";
import { OperationScreen } from "@/features/triple-agent/screens/operation";
import { InterludeScreen } from "@/features/triple-agent/screens/interlude";
import { DiscussionScreen } from "@/features/triple-agent/screens/discussion";
import { AccusationScreen } from "@/features/triple-agent/screens/accusation";
import { ResultsScreen } from "@/features/triple-agent/screens/results";

const phases: Phase[] = [
  "LOBBY", "ROLE_REVEAL", "OPERATION_INPUT", "OPERATION_RESULT", "OPERATION_INTERLUDE",
  "DISCUSSION", "VOTE_INPUT", "RESULTS_INTRO", "VOTE_RESULTS", "IMPRISONMENT_REVEAL",
  "AGENCY_REVEAL", "OUTCOME_REVEAL", "LEADERBOARD", "OUT_OF_LOOP", "END",
];

function fixture(phase: Phase): RoomProjection {
  const players = [
    { id: "p1", name: "AGENT A", seat: 1, ready: true, connected: true, vote_submitted: false },
    { id: "p2", name: "AGENT B", seat: 2, ready: true, connected: true, vote_submitted: true },
    { id: "p3", name: "AGENT C", seat: 3, ready: true, connected: true, vote_submitted: false },
    { id: "p4", name: "AGENT D", seat: 4, ready: true, connected: true, vote_submitted: true },
    { id: "p5", name: "AGENT E", seat: 5, ready: true, connected: true, vote_submitted: false },
  ];
  return {
    type: "room.projection",
    public: {
      room_id: "room-fixture", host_id: "p1", phase, version: 42, players,
      settings: {
        discussion_timer_enabled: true, discussion_seconds: 300, min_players: 5, max_players: 10,
        interlude_seconds: 7, virus_count: 2,
        enabled_operations: ["OneRandom", "Detector", "Swap", "Strain"],
        enabled_roles: ["FAKE_RED", "LOYAL_BLUE"],
      },
      active_player_id: "p1",
      operation: {
        kind: "Detector", name: "Secret Intel", input_kind: "TWO_TARGETS", target_count: 2,
        active_player_id: "p1", active_player_name: "AGENT A", input_owner_id: "p1",
        public_instruction: "Choose two players and learn whether either is VIRUS.",
      },
      discussion_deadline: new Date(Date.now() + 180_000).toISOString(),
      vote_totals: { p1: 0, p2: 3, p3: 1, p4: 0, p5: 1 },
      imprisoned_player_id: "p2",
      revealed_faction: "VIRUS", winner: "SERVICE", pending_role_acks: 2, discussion_ready_count: 2,
      leaderboard: [
        { player_id: "p1", name: "AGENT A", faction: "SERVICE", role: "FAKE_RED", votes: 0, result: "WINNER" },
        { player_id: "p2", name: "AGENT B", faction: "VIRUS", votes: 3, result: "LOSER" },
        { player_id: "p3", name: "AGENT C", faction: "SERVICE", votes: 1, result: "WINNER" },
        { player_id: "p4", name: "AGENT D", faction: "SERVICE", votes: 0, result: "WINNER" },
        { player_id: "p5", name: "AGENT E", faction: "VIRUS", votes: 1, result: "LOSER" },
      ],
    },
    private: {
      player_id: "p1", faction: "SERVICE", initial_faction: "SERVICE", role: "FAKE_RED",
      role_name: "Triple Agent", role_description: "The VIRUS agents think you are one of them.",
      role_effect: "You actually work for SERVICE.",
      virus_roster: [{ id: "p2", name: "AGENT B", seat: 2, connected: true }], virus_team_size: 2,
      legal_target_ids: ["p2", "p3", "p4", "p5"],
      operation_instruction: "Choose two players.",
      operation_result: { code: "ONE_VIRUS_ONE_SERVICE", target_player_ids: ["p2", "p3"], message: "Intel resolved." },
      vote_submitted: false, can_submit: true,
    },
  };
}

export function MockFixtures() {
  const [phase, setPhase] = useState<Phase>("LOBBY");
  const [settings, setSettings] = useState(false);
  const [lastCommand, setLastCommand] = useState<string>("none");
  const projection = useMemo(() => fixture(phase), [phase]);
  const onSend = (command: ClientCommand) => setLastCommand(JSON.stringify(command));
  const common = { projection, pending: null, onSend };

  let screen;
  if (settings) screen = <SettingsScreen {...common} onClose={() => setSettings(false)} />;
  else {
    switch (phase) {
      case "LOBBY": screen = <LobbyScreen {...common} joinCode="XZ04NW" copied={false} onShareLink={() => setLastCommand("share link")} onCopyRoomCode={() => {}} />; break;
      case "ROLE_REVEAL": screen = <RoleScreen {...common} />; break;
      case "OPERATION_INPUT": case "OPERATION_RESULT": screen = <OperationScreen {...common} />; break;
      case "OPERATION_INTERLUDE": screen = <InterludeScreen {...common} />; break;
      case "DISCUSSION": screen = <DiscussionScreen {...common} />; break;
      case "VOTE_INPUT": screen = <AccusationScreen {...common} />; break;
      default: screen = <ResultsScreen {...common} />;
    }
  }

  return (
    <main className="ta-viewport">
      <section className="ta-device">
        <header className="ta-header border-b-4 border-black bg-ta-orange-deep px-3 py-3 text-ta-paper">
          <div className="ta-header-inner flex-wrap">
            <label className="ta-condensed text-xs">Fixture <select className="ml-2 bg-white p-2 text-black" value={phase} onChange={(event) => { setPhase(event.target.value as Phase); setSettings(false); }}>{phases.map((item) => <option key={item}>{item}</option>)}</select></label>
            <button className="ta-tab" type="button" onClick={() => setSettings((open) => !open)}>{settings ? "CLOSE SETTINGS" : "SETTINGS"}</button>
          </div>
          <p className="ta-condensed mt-2 break-all text-xs">LAST COMMAND: {lastCommand}</p>
        </header>
        <div className="ta-stage"><div className="ta-stage-inner">{screen}</div></div>
      </section>
    </main>
  );
}
