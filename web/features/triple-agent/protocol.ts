export type Phase =
  | "LOBBY"
  | "ROLE_REVEAL"
  | "OPERATION_INPUT"
  | "OPERATION_RESULT"
  | "OPERATION_INTERLUDE"
  | "DISCUSSION"
  | "VOTE_INPUT"
  | "RESULTS_INTRO"
  | "VOTE_RESULTS"
  | "IMPRISONMENT_REVEAL"
  | "AGENCY_REVEAL"
  | "OUTCOME_REVEAL"
  | "LEADERBOARD"
  | "OUT_OF_LOOP"
  | "END";

export type Faction = "SERVICE" | "VIRUS" | "NONE";

export type PlayerProjection = {
  id: string;
  name: string;
  seat: number;
  ready: boolean;
  connected: boolean;
  vote_submitted: boolean;
};

export type OperationResult = {
  code?: string;
  target_player_id?: string;
  target_player_ids?: string[];
  target_faction?: Faction;
  other_player_id?: string;
  other_faction?: Faction;
  your_faction?: Faction;
  message: string;
};

export type OperationInputKind = "NONE" | "ONE_TARGET" | "TWO_TARGETS" | "CHOICE" | "PRIVATE_INFO";

export type RoomProjection = {
  type: "room.projection";
  public: {
    room_id: string;
    host_id: string;
    phase: Phase;
    version: number;
    players: PlayerProjection[];
    settings: {
      discussion_timer_enabled: boolean;
      discussion_seconds: number;
      enabled_operations: string[];
      min_players?: number;
      max_players?: number;
      interlude_seconds?: number;
      virus_count?: number;
      enabled_roles?: string[];
    };
    active_player_id?: string;
    operation?: {
      kind: string;
      name: string;
      input_kind: OperationInputKind;
      target_count?: number;
      active_player_id: string;
      active_player_name: string;
      input_owner_id?: string;
      step?: number;
      public_instruction: string;
    };
    discussion_deadline?: string;
    vote_totals?: Record<string, number>;
    imprisoned_player_id?: string;
    revealed_faction?: Faction;
    winner?: Faction;
    leaderboard?: Array<{
      player_id: string;
      name: string;
      faction: Faction;
      role?: string;
      defection?: "BLUE_DEFECTOR" | "RED_DEFECTOR";
      votes: number;
      result: "WINNER" | "LOSER" | "DRAW";
    }>;
    activity?: string;
    pending_role_acks?: number;
    discussion_ready_count?: number;
  };
  private: {
    player_id: string;
    role?: string;
    initial_faction?: Faction;
    faction?: Faction;
    apparent_faction?: Faction;
    operation_result?: OperationResult;
    operation_instruction?: string;
    role_name?: string;
    role_description?: string;
    role_effect?: string;
    virus_roster?: Array<{ id: string; name: string; seat: number; connected: boolean }>;
    virus_team_size?: number;
    operation_kind?: string;
    operation_name?: string;
    legal_target_ids?: string[];
    choices?: string[];
    vote_submitted: boolean;
    can_submit: boolean;
  };
};

export type RoomIdentity = {
  room_id: string;
  join_code: string;
  player_id: string;
  reconnect_token: string;
};

export type ClientCommand =
  | { kind: "lobby.ready" }
  | { kind: "match.start"; operation_kind?: string }
  | { kind: "role.acknowledge" }
  | { kind: "operation.resolve"; target_ids?: string[]; choice?: string }
  | { kind: "operation.select_target"; target_id: string }
  | { kind: "operation.explain_done" }
  | { kind: "interlude.advance" }
  | { kind: "discussion.advance" }
  | { kind: "vote.submit"; target_id: string }
  | { kind: "results.continue" }
  | { kind: "match.rematch" }
  | { kind: "lobby.operation_enabled"; operation_kind: string; operation_enabled: boolean }
  | { kind: "lobby.discussion_timer"; discussion_timer_enabled: boolean; discussion_seconds?: number }
  | { kind: "lobby.virus_count"; virus_count: number }
  | { kind: "lobby.role_enabled"; role_id: string; role_enabled: boolean }
  | { kind: "lobby.transfer_host"; target_id: string }
  | { kind: "lobby.kick_player"; target_id: string };

export type ResyncCommand = { kind: "room.resync" };

export type CommandMetadata = {
  request_id: string;
  expected_version: number;
};

export type WireCommand = (ClientCommand & CommandMetadata) | ResyncCommand;

export type CommandAck = {
  type: "command.ack";
  request_id: string;
  ok: boolean;
  error?: string;
};

export type SessionAuthenticated = {
  type: "session.authenticated";
};

export type SessionError = {
  type: "session.error";
  status?: number;
  error?: string;
  code?: string;
};

export type RoomServerMessage = RoomProjection | CommandAck | SessionAuthenticated | SessionError;

const PHASES: ReadonlySet<string> = new Set<Phase>([
  "LOBBY",
  "ROLE_REVEAL",
  "OPERATION_INPUT",
  "OPERATION_RESULT",
  "OPERATION_INTERLUDE",
  "DISCUSSION",
  "VOTE_INPUT",
  "RESULTS_INTRO",
  "VOTE_RESULTS",
  "IMPRISONMENT_REVEAL",
  "AGENCY_REVEAL",
  "OUTCOME_REVEAL",
  "LEADERBOARD",
  "OUT_OF_LOOP",
  "END",
]);

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isString(value: unknown): value is string {
  return typeof value === "string";
}

function isNumber(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value);
}

function isBoolean(value: unknown): value is boolean {
  return typeof value === "boolean";
}

function isStringArray(value: unknown): value is string[] {
  return Array.isArray(value) && value.every(isString);
}

function isFaction(value: unknown): value is Faction {
  return value === "SERVICE" || value === "VIRUS" || value === "NONE";
}

function isPlayerProjection(value: unknown): value is PlayerProjection {
  return isRecord(value)
    && isString(value.id)
    && isString(value.name)
    && isNumber(value.seat)
    && isBoolean(value.ready)
    && isBoolean(value.connected)
    && isBoolean(value.vote_submitted);
}

function isSettings(value: unknown): value is RoomProjection["public"]["settings"] {
  if (!isRecord(value)) return false;
  if (!isBoolean(value.discussion_timer_enabled)) return false;
  if (!isNumber(value.discussion_seconds)) return false;
  if (!isStringArray(value.enabled_operations)) return false;
  if (value.enabled_roles !== undefined && !isStringArray(value.enabled_roles)) return false;
  if (value.min_players !== undefined && !isNumber(value.min_players)) return false;
  if (value.max_players !== undefined && !isNumber(value.max_players)) return false;
  if (value.interlude_seconds !== undefined && !isNumber(value.interlude_seconds)) return false;
  if (value.virus_count !== undefined && !isNumber(value.virus_count)) return false;
  return true;
}

function isOperationResult(value: unknown): value is OperationResult {
  if (!isRecord(value) || !isString(value.message)) return false;
  if (value.code !== undefined && !isString(value.code)) return false;
  if (value.target_player_id !== undefined && !isString(value.target_player_id)) return false;
  if (value.target_player_ids !== undefined && !isStringArray(value.target_player_ids)) return false;
  if (value.target_faction !== undefined && !isFaction(value.target_faction)) return false;
  if (value.other_player_id !== undefined && !isString(value.other_player_id)) return false;
  if (value.other_faction !== undefined && !isFaction(value.other_faction)) return false;
  if (value.your_faction !== undefined && !isFaction(value.your_faction)) return false;
  return true;
}

function isOperation(value: unknown): value is NonNullable<RoomProjection["public"]["operation"]> {
  if (!isRecord(value)) return false;
  if (!isString(value.kind) || !isString(value.name)) return false;
  if (value.input_kind !== "NONE" && value.input_kind !== "ONE_TARGET" && value.input_kind !== "TWO_TARGETS" && value.input_kind !== "CHOICE" && value.input_kind !== "PRIVATE_INFO") return false;
  if (!isString(value.active_player_id) || !isString(value.active_player_name) || !isString(value.public_instruction)) return false;
  if (value.target_count !== undefined && !isNumber(value.target_count)) return false;
  if (value.input_owner_id !== undefined && !isString(value.input_owner_id)) return false;
  if (value.step !== undefined && !isNumber(value.step)) return false;
  return true;
}

function isVoteTotals(value: unknown): value is Record<string, number> {
  return isRecord(value) && Object.values(value).every(isNumber);
}

function isLeaderboard(value: unknown): value is NonNullable<RoomProjection["public"]["leaderboard"]> {
  return Array.isArray(value) && value.every((entry) => {
    if (!isRecord(entry)) return false;
    if (!isString(entry.player_id) || !isString(entry.name) || !isFaction(entry.faction) || !isNumber(entry.votes)) return false;
    if (entry.role !== undefined && !isString(entry.role)) return false;
    if (entry.defection !== undefined && entry.defection !== "BLUE_DEFECTOR" && entry.defection !== "RED_DEFECTOR") return false;
    return entry.result === "WINNER" || entry.result === "LOSER" || entry.result === "DRAW";
  });
}

function isVirusRoster(value: unknown): value is NonNullable<RoomProjection["private"]["virus_roster"]> {
  return Array.isArray(value) && value.every((entry) => isRecord(entry)
    && isString(entry.id)
    && isString(entry.name)
    && isNumber(entry.seat)
    && isBoolean(entry.connected));
}

function isPrivateProjection(value: unknown): value is RoomProjection["private"] {
  if (!isRecord(value)) return false;
  if (!isString(value.player_id)) return false;
  if (!isBoolean(value.vote_submitted) || !isBoolean(value.can_submit)) return false;
  for (const key of ["role", "operation_instruction", "role_name", "role_description", "role_effect", "operation_kind", "operation_name"] as const) {
    if (value[key] !== undefined && !isString(value[key])) return false;
  }
  if (value.faction !== undefined && !isFaction(value.faction)) return false;
  if (value.initial_faction !== undefined && !isFaction(value.initial_faction)) return false;
  if (value.apparent_faction !== undefined && !isFaction(value.apparent_faction)) return false;
  if (value.legal_target_ids !== undefined && !isStringArray(value.legal_target_ids)) return false;
  if (value.choices !== undefined && !isStringArray(value.choices)) return false;
  if (value.operation_result !== undefined && !isOperationResult(value.operation_result)) return false;
  if (value.virus_roster !== undefined && !isVirusRoster(value.virus_roster)) return false;
  if (value.virus_team_size !== undefined && !isNumber(value.virus_team_size)) return false;
  return true;
}

function isRoomProjection(value: Record<string, unknown>): value is RoomProjection {
  if (value.type !== "room.projection" || !isRecord(value.public) || !isPrivateProjection(value.private)) return false;
  const room = value.public;
  if (!isString(room.room_id) || !isString(room.host_id) || !isString(room.phase) || !PHASES.has(room.phase)) return false;
  if (!isNumber(room.version) || !Array.isArray(room.players) || !room.players.every(isPlayerProjection) || !isSettings(room.settings)) return false;
  if (room.active_player_id !== undefined && !isString(room.active_player_id)) return false;
  if (room.operation !== undefined && !isOperation(room.operation)) return false;
  if (room.discussion_deadline !== undefined && !isString(room.discussion_deadline)) return false;
  if (room.vote_totals !== undefined && !isVoteTotals(room.vote_totals)) return false;
  if (room.imprisoned_player_id !== undefined && !isString(room.imprisoned_player_id)) return false;
  if (room.revealed_faction !== undefined && !isFaction(room.revealed_faction)) return false;
  if (room.winner !== undefined && !isFaction(room.winner)) return false;
  if (room.leaderboard !== undefined && !isLeaderboard(room.leaderboard)) return false;
  if (room.activity !== undefined && !isString(room.activity)) return false;
  if (room.pending_role_acks !== undefined && !isNumber(room.pending_role_acks)) return false;
  if (room.discussion_ready_count !== undefined && !isNumber(room.discussion_ready_count)) return false;
  return true;
}

function isCommandAck(value: Record<string, unknown>): value is CommandAck {
  return value.type === "command.ack"
    && isString(value.request_id)
    && isBoolean(value.ok)
    && (value.error === undefined || isString(value.error));
}

function isSessionError(value: Record<string, unknown>): value is SessionError {
  return value.type === "session.error"
    && (value.status === undefined || isNumber(value.status))
    && (value.error === undefined || isString(value.error))
    && (value.code === undefined || isString(value.code));
}

export function parseRoomServerMessage(value: unknown): RoomServerMessage | null {
  if (!isRecord(value) || !isString(value.type)) return null;

  switch (value.type) {
    case "room.projection":
      return isRoomProjection(value) ? value : null;
    case "command.ack":
      return isCommandAck(value) ? value : null;
    case "session.authenticated":
      return { type: "session.authenticated" };
    case "session.error":
      return isSessionError(value) ? value : null;
    default:
      return null;
  }
}

export function isRoomIdentity(value: unknown): value is RoomIdentity {
  return isRecord(value)
    && isString(value.room_id)
    && isString(value.join_code)
    && isString(value.player_id)
    && isString(value.reconnect_token);
}
