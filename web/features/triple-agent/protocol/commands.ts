/**
 * Commands exposed to the feature UI. Connection metadata is deliberately
 * absent: the session supplies the current expected version and the
 * connection allocates a request id when it serializes the command.
 */
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
  | { kind: "lobby.role_enabled"; role_id: string; role_enabled: boolean };

/** The websocket-only resync control frame is not a domain ClientCommand. */
export type ResyncCommand = { kind: "room.resync" };

export type CommandMetadata = {
  request_id: string;
  expected_version: number;
};

export type WireCommand = (ClientCommand & CommandMetadata) | ResyncCommand;
