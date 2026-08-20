import type {
  CommandAck,
  Faction,
  OperationResult,
  Phase,
  PlayerProjection,
  RoomIdentity,
  RoomProjection,
  RoomServerMessage,
  SessionError,
} from "./types";

const PHASES: ReadonlySet<string> = new Set<Phase>([
  "LOBBY", "ROLE_REVEAL", "OPERATION_INPUT", "OPERATION_RESULT", "OPERATION_INTERLUDE",
  "DISCUSSION", "VOTE_INPUT", "RESULTS_INTRO", "VOTE_RESULTS", "IMPRISONMENT_REVEAL",
  "AGENCY_REVEAL", "OUTCOME_REVEAL", "LEADERBOARD", "OUT_OF_LOOP", "END",
]);

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isString(value: unknown): value is string { return typeof value === "string"; }
function isNumber(value: unknown): value is number { return typeof value === "number" && Number.isFinite(value); }
function isBoolean(value: unknown): value is boolean { return typeof value === "boolean"; }
function isStringArray(value: unknown): value is string[] { return Array.isArray(value) && value.every(isString); }
function isFaction(value: unknown): value is Faction { return value === "SERVICE" || value === "VIRUS" || value === "NONE"; }

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
  if (!isBoolean(value.discussion_timer_enabled) || !isNumber(value.discussion_seconds) || !isStringArray(value.enabled_operations)) return false;
  if (value.enabled_roles !== undefined && !isStringArray(value.enabled_roles)) return false;
  for (const key of ["min_players", "max_players", "interlude_seconds", "virus_count"] as const) {
    if (value[key] !== undefined && !isNumber(value[key])) return false;
  }
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
  if (!isRecord(value) || !isString(value.kind) || !isString(value.name)) return false;
  if (!["NONE", "ONE_TARGET", "TWO_TARGETS", "CHOICE", "PRIVATE_INFO"].includes(String(value.input_kind))) return false;
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
    if (!isRecord(entry) || !isString(entry.player_id) || !isString(entry.name) || !isFaction(entry.faction) || !isNumber(entry.votes)) return false;
    if (entry.role !== undefined && !isString(entry.role)) return false;
    if (entry.defection !== undefined && entry.defection !== "BLUE_DEFECTOR" && entry.defection !== "RED_DEFECTOR") return false;
    return entry.result === "WINNER" || entry.result === "LOSER" || entry.result === "DRAW";
  });
}

function isVirusRoster(value: unknown): value is NonNullable<RoomProjection["private"]["virus_roster"]> {
  return Array.isArray(value) && value.every((entry) => isRecord(entry)
    && isString(entry.id) && isString(entry.name) && isNumber(entry.seat) && isBoolean(entry.connected));
}

function isPrivateProjection(value: unknown): value is RoomProjection["private"] {
  if (!isRecord(value) || !isString(value.player_id) || !isBoolean(value.vote_submitted) || !isBoolean(value.can_submit)) return false;
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
  return value.type === "command.ack" && isString(value.request_id) && isBoolean(value.ok)
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
    case "room.projection": return isRoomProjection(value) ? value : null;
    case "command.ack": return isCommandAck(value) ? value : null;
    case "session.authenticated": return { type: "session.authenticated" };
    case "session.error": return isSessionError(value) ? value : null;
    default: return null;
  }
}

export function isRoomIdentity(value: unknown): value is RoomIdentity {
  return isRecord(value)
    && isString(value.room_id)
    && isString(value.join_code)
    && isString(value.player_id)
    && isString(value.reconnect_token);
}
