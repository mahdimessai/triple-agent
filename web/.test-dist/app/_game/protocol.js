"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.parseRoomServerMessage = parseRoomServerMessage;
exports.isRoomIdentity = isRoomIdentity;
const PHASES = new Set([
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
function isRecord(value) {
    return typeof value === "object" && value !== null && !Array.isArray(value);
}
function isString(value) {
    return typeof value === "string";
}
function isNumber(value) {
    return typeof value === "number" && Number.isFinite(value);
}
function isBoolean(value) {
    return typeof value === "boolean";
}
function isStringArray(value) {
    return Array.isArray(value) && value.every(isString);
}
function isFaction(value) {
    return value === "SERVICE" || value === "VIRUS" || value === "NONE";
}
function isPlayerProjection(value) {
    return isRecord(value)
        && isString(value.id)
        && isString(value.name)
        && isNumber(value.seat)
        && isBoolean(value.ready)
        && isBoolean(value.connected)
        && isBoolean(value.vote_submitted);
}
function isSettings(value) {
    if (!isRecord(value))
        return false;
    if (!isBoolean(value.discussion_timer_enabled))
        return false;
    if (!isNumber(value.discussion_seconds))
        return false;
    if (!isStringArray(value.enabled_operations))
        return false;
    if (value.enabled_roles !== undefined && !isStringArray(value.enabled_roles))
        return false;
    if (value.min_players !== undefined && !isNumber(value.min_players))
        return false;
    if (value.max_players !== undefined && !isNumber(value.max_players))
        return false;
    if (value.interlude_seconds !== undefined && !isNumber(value.interlude_seconds))
        return false;
    if (value.virus_count !== undefined && !isNumber(value.virus_count))
        return false;
    return true;
}
function isOperationResult(value) {
    if (!isRecord(value) || !isString(value.message))
        return false;
    if (value.code !== undefined && !isString(value.code))
        return false;
    if (value.target_player_id !== undefined && !isString(value.target_player_id))
        return false;
    if (value.target_player_ids !== undefined && !isStringArray(value.target_player_ids))
        return false;
    if (value.target_faction !== undefined && !isFaction(value.target_faction))
        return false;
    if (value.other_player_id !== undefined && !isString(value.other_player_id))
        return false;
    if (value.other_faction !== undefined && !isFaction(value.other_faction))
        return false;
    if (value.your_faction !== undefined && !isFaction(value.your_faction))
        return false;
    return true;
}
function isOperation(value) {
    if (!isRecord(value))
        return false;
    if (!isString(value.kind) || !isString(value.name))
        return false;
    if (value.input_kind !== "NONE" && value.input_kind !== "ONE_TARGET" && value.input_kind !== "TWO_TARGETS" && value.input_kind !== "CHOICE" && value.input_kind !== "PRIVATE_INFO")
        return false;
    if (!isString(value.active_player_id) || !isString(value.active_player_name) || !isString(value.public_instruction))
        return false;
    if (value.target_count !== undefined && !isNumber(value.target_count))
        return false;
    if (value.input_owner_id !== undefined && !isString(value.input_owner_id))
        return false;
    if (value.step !== undefined && !isNumber(value.step))
        return false;
    return true;
}
function isVoteTotals(value) {
    return isRecord(value) && Object.values(value).every(isNumber);
}
function isLeaderboard(value) {
    return Array.isArray(value) && value.every((entry) => {
        if (!isRecord(entry))
            return false;
        if (!isString(entry.player_id) || !isString(entry.name) || !isFaction(entry.faction) || !isNumber(entry.votes))
            return false;
        if (entry.role !== undefined && !isString(entry.role))
            return false;
        if (entry.defection !== undefined && entry.defection !== "BLUE_DEFECTOR" && entry.defection !== "RED_DEFECTOR")
            return false;
        return entry.result === "WINNER" || entry.result === "LOSER" || entry.result === "DRAW";
    });
}
function isVirusRoster(value) {
    return Array.isArray(value) && value.every((entry) => isRecord(entry)
        && isString(entry.id)
        && isString(entry.name)
        && isNumber(entry.seat)
        && isBoolean(entry.connected));
}
function isPrivateProjection(value) {
    if (!isRecord(value))
        return false;
    if (!isString(value.player_id))
        return false;
    if (!isBoolean(value.vote_submitted) || !isBoolean(value.can_submit))
        return false;
    for (const key of ["role", "operation_instruction", "role_name", "role_description", "role_effect", "operation_kind", "operation_name"]) {
        if (value[key] !== undefined && !isString(value[key]))
            return false;
    }
    if (value.faction !== undefined && !isFaction(value.faction))
        return false;
    if (value.initial_faction !== undefined && !isFaction(value.initial_faction))
        return false;
    if (value.apparent_faction !== undefined && !isFaction(value.apparent_faction))
        return false;
    if (value.legal_target_ids !== undefined && !isStringArray(value.legal_target_ids))
        return false;
    if (value.choices !== undefined && !isStringArray(value.choices))
        return false;
    if (value.operation_result !== undefined && !isOperationResult(value.operation_result))
        return false;
    if (value.virus_roster !== undefined && !isVirusRoster(value.virus_roster))
        return false;
    if (value.virus_team_size !== undefined && !isNumber(value.virus_team_size))
        return false;
    return true;
}
function isRoomProjection(value) {
    if (value.type !== "room.projection" || !isRecord(value.public) || !isPrivateProjection(value.private))
        return false;
    const room = value.public;
    if (!isString(room.room_id) || !isString(room.host_id) || !isString(room.phase) || !PHASES.has(room.phase))
        return false;
    if (!isNumber(room.version) || !Array.isArray(room.players) || !room.players.every(isPlayerProjection) || !isSettings(room.settings))
        return false;
    if (room.active_player_id !== undefined && !isString(room.active_player_id))
        return false;
    if (room.operation !== undefined && !isOperation(room.operation))
        return false;
    if (room.discussion_deadline !== undefined && !isString(room.discussion_deadline))
        return false;
    if (room.vote_totals !== undefined && !isVoteTotals(room.vote_totals))
        return false;
    if (room.imprisoned_player_id !== undefined && !isString(room.imprisoned_player_id))
        return false;
    if (room.revealed_faction !== undefined && !isFaction(room.revealed_faction))
        return false;
    if (room.winner !== undefined && !isFaction(room.winner))
        return false;
    if (room.leaderboard !== undefined && !isLeaderboard(room.leaderboard))
        return false;
    if (room.activity !== undefined && !isString(room.activity))
        return false;
    if (room.pending_role_acks !== undefined && !isNumber(room.pending_role_acks))
        return false;
    if (room.discussion_ready_count !== undefined && !isNumber(room.discussion_ready_count))
        return false;
    return true;
}
function isCommandAck(value) {
    return value.type === "command.ack"
        && isString(value.request_id)
        && isBoolean(value.ok)
        && (value.error === undefined || isString(value.error));
}
function isSessionError(value) {
    return value.type === "session.error"
        && (value.status === undefined || isNumber(value.status))
        && (value.error === undefined || isString(value.error))
        && (value.code === undefined || isString(value.code));
}
function parseRoomServerMessage(value) {
    if (!isRecord(value) || !isString(value.type))
        return null;
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
function isRoomIdentity(value) {
    return isRecord(value)
        && isString(value.room_id)
        && isString(value.join_code)
        && isString(value.player_id)
        && isString(value.reconnect_token);
}
