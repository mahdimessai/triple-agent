import { operationCatalog, type OperationId } from "@/components/triple-agent/operation-catalog";
import type { ClientCommand } from "@/features/triple-agent/protocol";

export type ScreenId = "title" | "join" | "lobby" | "settings" | "mission" | "role" | "operation" | "interlude" | "discussion" | "accusation" | "results";

export const screens: Array<{ id: ScreenId; label: string; shortLabel: string }> = [
  { id: "title", label: "Title", shortLabel: "01" },
  { id: "join", label: "Join", shortLabel: "02" },
  { id: "lobby", label: "Lobby", shortLabel: "03" },
  { id: "settings", label: "Settings", shortLabel: "04" },
  { id: "mission", label: "Briefing", shortLabel: "05" },
  { id: "role", label: "Role reveal", shortLabel: "06" },
  { id: "operation", label: "Operation", shortLabel: "07" },
  { id: "discussion", label: "Discussion", shortLabel: "08" },
  { id: "accusation", label: "Accusation", shortLabel: "09" },
  { id: "results", label: "Results", shortLabel: "10" },
];

/**
 * Transitional UI payload shape used by screens that predate ClientCommand.
 * It is intentionally kept at this boundary so screen components do not know
 * about request ids or expected versions.
 */
export type CommandPayload = {
  targetId?: string;
  targetIds?: string[];
  operationKind?: string;
  operationEnabled?: boolean;
  discussionTimerEnabled?: boolean;
  discussionSeconds?: number;
  virusCount?: number;
  roleId?: string;
  roleEnabled?: boolean;
  choice?: string;
  name?: string;
};

export type CommandSender = {
  (command: ClientCommand): void;
  (kind: ClientCommand["kind"], payload?: CommandPayload): void;
};

export function clientCommandFromLegacy(kind: ClientCommand["kind"], payload: CommandPayload = {}): ClientCommand {
  switch (kind) {
    case "lobby.ready": return { kind };
    case "match.start": return { kind, ...(payload.operationKind ? { operation_kind: payload.operationKind } : {}) };
    case "role.acknowledge": return { kind };
    case "operation.resolve": return {
      kind,
      ...(payload.targetIds ? { target_ids: payload.targetIds } : {}),
      ...(payload.choice ? { choice: payload.choice } : {}),
    };
    case "operation.select_target": return { kind, target_id: payload.targetId ?? "" };
    case "operation.explain_done": return { kind };
    case "interlude.advance": return { kind };
    case "discussion.advance": return { kind };
    case "vote.submit": return { kind, target_id: payload.targetId ?? "" };
    case "results.continue": return { kind };
    case "match.rematch": return { kind };
    case "lobby.operation_enabled": return { kind, operation_kind: payload.operationKind ?? "", operation_enabled: payload.operationEnabled ?? false };
    case "lobby.discussion_timer": return {
      kind,
      discussion_timer_enabled: payload.discussionTimerEnabled ?? true,
      ...(payload.discussionSeconds ? { discussion_seconds: payload.discussionSeconds } : {}),
    };
    case "lobby.virus_count": return { kind, virus_count: payload.virusCount ?? 0 };
    case "lobby.role_enabled": return { kind, role_id: payload.roleId ?? "", role_enabled: payload.roleEnabled ?? false };
  }
}

export function operationIdForServerKind(kind: string | undefined): OperationId {
  if (kind === "Swap" || kind === "SpyTransfer") return "Swap";
  if (kind && operationCatalog.some((operation) => operation.id === kind)) return kind as OperationId;
  return "OneRandom";
}
