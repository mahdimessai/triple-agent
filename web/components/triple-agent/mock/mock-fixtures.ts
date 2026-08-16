export type MockPlayer = {
  id: string;
  name: string;
  seat: number;
  ready: boolean;
  connected: boolean;
  isHost?: boolean;
};

export type MockActivityEntry = {
  label: string;
  detail: string;
  tone: "neutral" | "warning" | "success";
};

export type MockFixture = {
  roomCode: string;
  playerName: string;
  players: readonly MockPlayer[];
  agencyCounts: {
    service: number;
    virus: number;
  };
  serviceTeammates: readonly string[];
  virusTeammates: readonly string[];
  anonymousTip: {
    operationName: string;
    activePlayer: string;
    target: string;
    targetFaction: "SERVICE" | "VIRUS";
    publicDescription: string;
    privateInstruction: string;
    privateResult: string;
    publicExplanation: string;
  };
  hiddenAgenda: {
    title: string;
    body: string;
    outcomes: readonly string[];
  };
  discussion: {
    timerLabel: string;
    entries: readonly MockActivityEntry[];
  };
  accusation: {
    prompt: string;
    targets: readonly string[];
    selectedTarget: string;
  };
  results: {
    voteTotals: readonly { player: string; votes: number }[];
    imprisonedPlayer: string;
    imprisonedFaction: "SERVICE" | "VIRUS";
    agencyReveal: string;
    winner: "SERVICE" | "VIRUS";
    winnerReason: string;
    leaderboard: readonly {
      player: string;
      score: number;
      faction: string;
      role?: string;
      defection?: "BLUE_DEFECTOR" | "RED_DEFECTOR";
    }[];
    outOfLoop: readonly string[];
  };
  activity: readonly MockActivityEntry[];
};

/**
 * This fixture is deliberately plain data. It never crosses into the room
 * projection or WebSocket client, so a visual mock cannot accidentally become
 * an authority for multiplayer state.
 */
export const mockFixture = {
  roomCode: "V-019",
  playerName: "PLAYER C",
  players: [
    { id: "player-a", name: "PLAYER A", seat: 1, ready: true, connected: true, isHost: true },
    { id: "player-b", name: "PLAYER B", seat: 2, ready: true, connected: true },
    { id: "player-c", name: "PLAYER C", seat: 3, ready: true, connected: true },
    { id: "player-d", name: "PLAYER D", seat: 4, ready: true, connected: true },
    { id: "player-e", name: "PLAYER E", seat: 5, ready: false, connected: true },
  ],
  agencyCounts: {
    service: 3,
    virus: 2,
  },
  serviceTeammates: ["PLAYER A", "PLAYER B"],
  virusTeammates: ["PLAYER D"],
  anonymousTip: {
    operationName: "ANONYMOUS TIP",
    activePlayer: "PLAYER C",
    target: "PLAYER D",
    targetFaction: "SERVICE",
    publicDescription: "A source has revealed the agency of one other player.",
    privateInstruction: "Your source reveals the agency of one other player.",
    privateResult: "PLAYER D works for the SERVICE.",
    publicExplanation: "I received an Anonymous Tip. The source says PLAYER D works for the SERVICE.",
  },
  hiddenAgenda: {
    title: "New orders from up top",
    body: "The room is told only that a hidden agenda arrived. Which one it is stays with the recipient until the reveal.",
    outcomes: [
      "Sleeper Agent: your agency switches between Service and VIRUS",
      "Scapegoat or Grudge: you win only if you, or your target, are imprisoned",
      "Infatuation: you win only if the agent you are bound to wins",
      "Secret Tip: you learn one other agent's agency",
    ],
  },
  discussion: {
    timerLabel: "01:42 remaining",
    entries: [
      { label: "PLAYER C", detail: "claimed the first operation result", tone: "neutral" },
      { label: "PLAYER D", detail: "challenged the anonymous description", tone: "warning" },
      { label: "PLAYER A", detail: "asked the table to compare evidence", tone: "success" },
    ],
  },
  accusation: {
    prompt: "Select one player to accuse. Your choice is private until every player submits.",
    targets: ["PLAYER A", "PLAYER B", "PLAYER D", "PLAYER E"],
    selectedTarget: "PLAYER D",
  },
  results: {
    voteTotals: [
      { player: "PLAYER D", votes: 3 },
      { player: "PLAYER B", votes: 1 },
      { player: "PLAYER E", votes: 1 },
    ],
    imprisonedPlayer: "PLAYER D",
    imprisonedFaction: "SERVICE",
    agencyReveal: "The Agency held the majority, but the imprisoned role decides the round.",
    winner: "VIRUS",
    winnerReason: "A Service player was imprisoned while the Virus remained hidden.",
    leaderboard: [
      { player: "PLAYER B", score: 4, faction: "VIRUS", role: "LOYAL_RED" },
      { player: "PLAYER E", score: 3, faction: "VIRUS", role: "LYING_RED" },
      { player: "PLAYER A", score: 2, faction: "SERVICE", role: "FAKE_RED" },
      { player: "PLAYER C", score: 2, faction: "VIRUS", role: "LYING_BLUE", defection: "BLUE_DEFECTOR" },
      { player: "PLAYER D", score: 1, faction: "SERVICE", role: "FAKE_BLUE", defection: "RED_DEFECTOR" },
    ],
    outOfLoop: [
      "PLAYER D is out of the loop for the next operation.",
      "The table can still see the public result.",
      "Final role and defection markers are now public.",
    ],
  },
  activity: [
    { label: "ROOM", detail: "V-019 · 5 players connected", tone: "success" },
    { label: "ROUND", detail: "Operation 02 · Anonymous Tip", tone: "neutral" },
    { label: "WAITING", detail: "Server is resolving private submissions", tone: "warning" },
    { label: "PUBLIC", detail: "The next reveal will be visible to everyone", tone: "neutral" },
  ],
} as const satisfies MockFixture;
