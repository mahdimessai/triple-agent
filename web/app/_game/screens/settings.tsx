"use client";

import { memo, useCallback, useEffect, useMemo, useState, useSyncExternalStore } from "react";
import { createPortal } from "react-dom";
import type { ClientCommand, RoomProjection } from "../protocol";
import type { PendingCommand } from "../use-room";
import {
  getOperation,
  hiddenAgendaMemberIds,
  liveOperationIds,
  operationBrief,
  operations,
  packOperationIds,
  type OperationDefinition,
} from "../operations";
import { roles, type RoleDefinition } from "../roles";
import { ArtStamp, InkButton, PaperTitle } from "../ui";

const MIN_DISCUSSION_SECONDS = 60;
const MAX_DISCUSSION_SECONDS = 900;
const DISCUSSION_STEP_SECONDS = 30;
const MIN_VIRUS_COUNT = 0;
const MAX_VIRUS_COUNT = 4;

export type SettingsScreenProps = {
  projection: RoomProjection;
  pending: PendingCommand | null;
  onSend(command: ClientCommand): void;
  onClose(): void;
  error?: string | null;
};

function formatDuration(seconds: number): string {
  const minutes = Math.floor(seconds / 60);
  const remainder = seconds % 60;
  return `${minutes}:${String(remainder).padStart(2, "0")}`;
}

const Stepper = memo(function Stepper({
  label,
  value,
  hint,
  disabled,
  canDecrease,
  canIncrease,
  onDecrease,
  onIncrease,
}: {
  label: string;
  value: string;
  hint: string;
  disabled?: boolean;
  canDecrease: boolean;
  canIncrease: boolean;
  onDecrease: () => void;
  onIncrease: () => void;
}) {
  return (
    <div className="ta-stepper">
      <p className="ta-condensed text-xs tracking-[0.16em] uppercase text-black/65">{label}</p>
      <div className="mt-2 flex items-stretch gap-2">
        <button
          className="ta-stepper-arrow"
          type="button"
          aria-label={`Decrease ${label.toLowerCase()}`}
          disabled={disabled || !canDecrease}
          onClick={onDecrease}
        >
          &#9664;
        </button>
        <output className="ta-stepper-value ta-display">{value}</output>
        <button
          className="ta-stepper-arrow"
          type="button"
          aria-label={`Increase ${label.toLowerCase()}`}
          disabled={disabled || !canIncrease}
          onClick={onIncrease}
        >
          &#9654;
        </button>
      </div>
      <p className="ta-sans mt-2 text-xs leading-snug text-black/70">{hint}</p>
    </div>
  );
});

type ToggleHandler = (id: string, enabled: boolean) => void;

type InspectedItem =
  | {
      type: "operation";
      operation: OperationDefinition;
      label: string;
      enabled: boolean;
      disabled: boolean;
    }
  | {
      type: "role";
      role: RoleDefinition;
      enabled: boolean;
      disabled: boolean;
    }
  | null;

const CompactOperationCard = memo(function CompactOperationCard({
  operation,
  label,
  enabled,
  disabled,
  onToggle,
  onInspect,
}: {
  operation: OperationDefinition;
  label: string;
  enabled: boolean;
  disabled: boolean;
  onToggle: ToggleHandler;
  onInspect: (op: OperationDefinition, label: string, enabled: boolean, disabled: boolean) => void;
}) {
  const handleInspectClick = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      onInspect(operation, label, enabled, disabled);
    },
    [onInspect, operation, label, enabled, disabled]
  );

  const handleToggleClick = useCallback(() => {
    onToggle(operation.id, !enabled);
  }, [onToggle, operation.id, enabled]);

  return (
    <div
      className="ta-compact-card group"
      data-enabled={enabled}
      data-disabled={disabled ? "true" : undefined}
    >
      <button
        type="button"
        className="flex min-w-0 flex-1 items-center gap-2.5 text-left py-0.5"
        disabled={disabled}
        onClick={handleToggleClick}
        aria-pressed={enabled}
        aria-label={`${enabled ? "Disable" : "Enable"} ${operation.name}`}
      >
        <span className="ta-compact-card-art shrink-0" aria-hidden="true">
          <ArtStamp artName={operation.artName} alt="" className="h-6 w-7 object-contain" />
        </span>
        <span className="min-w-0 flex-1 pr-1">
          <span className="ta-narrow block text-sm sm:text-base leading-tight text-ta-ink uppercase">
            {operation.name}
          </span>
          <span className="ta-narrow block text-[0.62rem] tracking-[0.14em] text-black/55 uppercase">
            {label}
          </span>
        </span>
      </button>
      <div className="flex items-center gap-1.5 shrink-0 pl-1">
        <button
          type="button"
          className="ta-compact-info-btn"
          aria-label={`Inspect ${operation.name} briefing`}
          title="View tactical briefing"
          onClick={handleInspectClick}
        >
          i
        </button>
        <button
          type="button"
          disabled={disabled}
          onClick={handleToggleClick}
          className={`ta-compact-status-badge ${enabled ? "ta-badge-enabled" : "ta-badge-disabled"}`}
          aria-label={`Toggle ${operation.name}`}
        >
          {enabled ? "ON" : "OFF"}
        </button>
      </div>
    </div>
  );
});

const CompactRoleCard = memo(function CompactRoleCard({
  role,
  enabled,
  disabled,
  onToggle,
  onInspect,
}: {
  role: RoleDefinition;
  enabled: boolean;
  disabled: boolean;
  onToggle: ToggleHandler;
  onInspect: (role: RoleDefinition, enabled: boolean, disabled: boolean) => void;
}) {
  const handleInspectClick = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      onInspect(role, enabled, disabled);
    },
    [onInspect, role, enabled, disabled]
  );

  const handleToggleClick = useCallback(() => {
    onToggle(role.id, !enabled);
  }, [onToggle, role.id, enabled]);

  return (
    <div
      className="ta-compact-card group"
      data-enabled={enabled}
      data-disabled={disabled ? "true" : undefined}
    >
      <button
        type="button"
        className="flex min-w-0 flex-1 items-center gap-2.5 text-left py-0.5"
        disabled={disabled}
        onClick={handleToggleClick}
        aria-pressed={enabled}
        aria-label={`${enabled ? "Remove" : "Add"} ${role.name}`}
      >
        <span className="ta-compact-card-art shrink-0" aria-hidden="true">
          <ArtStamp artName={role.artName} alt="" className="h-6 w-7 object-contain" />
        </span>
        <span className="min-w-0 flex-1 pr-1">
          <span className="ta-narrow block text-sm sm:text-base leading-tight text-ta-ink uppercase">
            {role.name}
          </span>
          <span
            className={`ta-narrow block text-[0.62rem] tracking-[0.14em] uppercase ${
              role.faction === "VIRUS" ? "text-ta-red" : "text-[#1d5b79]"
            }`}
          >
            {role.faction}
          </span>
        </span>
      </button>
      <div className="flex items-center gap-1.5 shrink-0 pl-1">
        <button
          type="button"
          className="ta-compact-info-btn"
          aria-label={`Inspect ${role.name} dossier`}
          title="View role dossier"
          onClick={handleInspectClick}
        >
          i
        </button>
        <button
          type="button"
          disabled={disabled}
          onClick={handleToggleClick}
          className={`ta-compact-status-badge ${enabled ? "ta-badge-enabled" : "ta-badge-disabled"}`}
          aria-label={`Toggle ${role.name}`}
        >
          {enabled ? "ON" : "OFF"}
        </button>
      </div>
    </div>
  );
});

const emptySubscribe = () => () => {};

function useIsMounted(): boolean {
  return useSyncExternalStore(
    emptySubscribe,
    () => true,
    () => false
  );
}

function DossierModal({
  item,
  onClose,
  canToggle,
  onToggle,
}: {
  item: InspectedItem;
  onClose: () => void;
  canToggle?: boolean;
  onToggle?: (id: string, enabled: boolean) => void;
}) {
  const mounted = useIsMounted();

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [onClose]);

  if (!item || !mounted) return null;

  const isOp = item.type === "operation";
  const title = isOp ? item.operation.name : item.role.name;
  const artName = isOp ? item.operation.artName : item.role.artName;
  const tag = isOp ? item.label : item.role.faction;
  const tagColor = !isOp && item.role.faction === "VIRUS" ? "text-ta-red" : "text-[#1d5b79]";
  const description = isOp ? operationBrief(item.operation) : item.role.effect;
  const id = isOp ? item.operation.id : item.role.id;

  const modalContent = (
    <div className="ta-modal-portal-backdrop" role="presentation" onClick={onClose}>
      <div
        className="ta-paper relative w-full max-w-md p-5 text-left border-4 border-black shadow-[8px_8px_0_var(--ta-shadow)]"
        role="dialog"
        aria-modal="true"
        aria-labelledby="dossier-dialog-title"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start justify-between gap-3 border-b-2 border-black/25 pb-3.5">
          <div className="flex items-center gap-3">
            <div className="h-16 w-20 flex-shrink-0 border-2 border-black bg-white p-1 flex items-center justify-center shadow-[2px_2px_0_var(--ta-shadow)]">
              <ArtStamp artName={artName} alt="" className="h-full w-full object-contain" />
            </div>
            <div>
              <p className={`ta-condensed text-xs tracking-[0.16em] uppercase ${tagColor}`}>{tag}</p>
              <h3 id="dossier-dialog-title" className="ta-display text-2xl leading-none text-ta-ink">
                {title}
              </h3>
            </div>
          </div>
          <button
            type="button"
            className="ta-secondary-button !min-h-0 border-2 border-black px-2.5 py-1 text-xs uppercase tracking-wider"
            onClick={onClose}
          >
            ✕
          </button>
        </div>

        <div className="my-4 space-y-3">
          <div>
            <p className="ta-condensed text-[0.65rem] tracking-[0.16em] text-black/60 uppercase mb-0.5">
              {isOp ? "TACTICAL BRIEFING" : "ROLE PROFILE"}
            </p>
            <p className="ta-sans text-sm leading-relaxed text-ta-ink">{description}</p>
          </div>
          {isOp && (
            <div>
              <p className="ta-condensed text-[0.65rem] tracking-[0.16em] text-black/60 uppercase mb-0.5">
                INPUT MECHANISM
              </p>
              <p className="ta-sans text-xs leading-relaxed text-black/80">{item.operation.input}</p>
            </div>
          )}
          {!isOp && (
            <div>
              <p className="ta-condensed text-[0.65rem] tracking-[0.16em] text-black/60 uppercase mb-0.5">
                BACKGROUND
              </p>
              <p className="ta-sans text-xs leading-relaxed text-black/80">{item.role.description}</p>
            </div>
          )}
        </div>

        <div className="flex items-center justify-between border-t-2 border-black/25 pt-3.5">
          <span className="ta-sans text-xs text-black/60">
            Status: <strong className="text-ta-ink">{item.enabled ? "ACTIVE" : "EXCLUDED"}</strong>
          </span>
          {canToggle && !item.disabled ? (
            <button
              type="button"
              className={`ta-secondary-button !min-h-0 border-2 border-black px-4 py-1 text-xs uppercase tracking-wider ${
                item.enabled ? "bg-ta-red text-ta-paper" : "bg-ta-teal text-ta-ink"
              }`}
              onClick={() => onToggle?.(id, !item.enabled)}
            >
              {item.enabled ? "Disable" : "Enable"}
            </button>
          ) : (
            <button
              type="button"
              className="ta-secondary-button !min-h-0 border-2 border-black px-4 py-1 text-xs uppercase tracking-wider"
              onClick={onClose}
            >
              Close
            </button>
          )}
        </div>
      </div>
    </div>
  );

  return createPortal(modalContent, document.body);
}

// 1. Isolated Match Setup Section
const MatchSetupSection = memo(function MatchSetupSection({
  isTimerActive,
  currentSeconds,
  currentVirusCount,
  controlsLocked,
  isHost,
  onSetDuration,
  onSetVirusCount,
}: {
  isTimerActive: boolean;
  currentSeconds: number;
  currentVirusCount: number;
  controlsLocked: boolean;
  isHost: boolean;
  onSetDuration: (seconds: number) => void;
  onSetVirusCount: (count: number) => void;
}) {
  return (
    <div className="ta-paper p-4">
      <div className="mb-3 flex items-center justify-between border-b-2 border-black/15 pb-2.5">
        <div>
          <p className="ta-condensed text-xs tracking-[0.18em] uppercase text-ta-ink">MATCH SETUP</p>
          <p className="ta-sans mt-0.5 text-xs leading-snug text-black/70">
            {isHost
              ? "Configure the round discussion timer and initial VIRUS agents."
              : "Set the discussion length and how many agents start as VIRUS."}
          </p>
        </div>
        {!isHost ? (
          <span className="ta-condensed text-[0.65rem] tracking-wider uppercase px-2 py-0.5 bg-black/10 border-2 border-black/25">
            HOST CONTROLLED
          </span>
        ) : null}
      </div>
      <div className="grid gap-3 sm:grid-cols-2">
        <Stepper
          label="DISCUSSION TIMER"
          value={isTimerActive ? formatDuration(currentSeconds) : "OFF"}
          hint={
            isTimerActive
              ? `Step below ${formatDuration(MIN_DISCUSSION_SECONDS)} for untimed. Max ${formatDuration(MAX_DISCUSSION_SECONDS)}.`
              : "Discussion runs untimed until the host advances."
          }
          disabled={controlsLocked}
          canDecrease={isTimerActive}
          canIncrease={!isTimerActive || currentSeconds < MAX_DISCUSSION_SECONDS}
          onDecrease={() => onSetDuration(currentSeconds - DISCUSSION_STEP_SECONDS)}
          onIncrease={() =>
            onSetDuration(
              isTimerActive
                ? currentSeconds + DISCUSSION_STEP_SECONDS
                : Math.max(MIN_DISCUSSION_SECONDS, currentSeconds)
            )
          }
        />
        <Stepper
          label="VIRUS TEAM SIZE"
          value={currentVirusCount === 0 ? "AUTO" : String(currentVirusCount)}
          hint={
            currentVirusCount === 0
              ? "Auto-scales: 2 for 5-6 players, 3 for 7+."
              : `${currentVirusCount} agent${currentVirusCount === 1 ? "" : "s"} start as VIRUS.`
          }
          disabled={controlsLocked}
          canDecrease={currentVirusCount > MIN_VIRUS_COUNT}
          canIncrease={currentVirusCount < MAX_VIRUS_COUNT}
          onDecrease={() => onSetVirusCount(currentVirusCount - 1)}
          onIncrease={() => onSetVirusCount(currentVirusCount + 1)}
        />
      </div>
    </div>
  );
});

// 2. Isolated Special Roles Section
const SpecialRolesSection = memo(function SpecialRolesSection({
  enabledRoleIDs,
  controlsLocked,
  onToggleRole,
  onInspectRole,
}: {
  enabledRoleIDs: Set<string>;
  controlsLocked: boolean;
  onToggleRole: ToggleHandler;
  onInspectRole: (role: RoleDefinition, enabled: boolean, disabled: boolean) => void;
}) {
  const activeRoleCount = useMemo(
    () => roles.filter((role) => enabledRoleIDs.has(role.id)).length,
    [enabledRoleIDs]
  );

  return (
    <div className="ta-paper p-4">
      <div className="mb-3 flex items-end justify-between gap-3 border-b-2 border-black/15 pb-2.5">
        <div>
          <p className="ta-condensed text-xs tracking-[0.18em] uppercase text-ta-ink">SPECIAL ROLES</p>
          <p className="ta-sans mt-0.5 text-xs leading-snug text-black/70">
            Secret roles dealt on top of agency. Click card to toggle, <strong>i</strong> for dossier.
          </p>
        </div>
        <span className="ta-condensed shrink-0 whitespace-nowrap text-xs tracking-[0.12em] bg-black/10 px-2 py-0.5 border border-black/20">
          {activeRoleCount} / {roles.length} IN POOL
        </span>
      </div>
      <div className="grid gap-2.5 sm:grid-cols-2">
        {roles.map((role) => (
          <CompactRoleCard
            key={role.id}
            role={role}
            enabled={enabledRoleIDs.has(role.id)}
            disabled={controlsLocked}
            onToggle={onToggleRole}
            onInspect={onInspectRole}
          />
        ))}
      </div>
    </div>
  );
});

// 3. Isolated Operations Deck Section
const OperationsDeckSection = memo(function OperationsDeckSection({
  enabledIDs,
  canEdit,
  onToggleOperation,
  onInspectOperation,
}: {
  enabledIDs: Set<string>;
  canEdit: boolean;
  onToggleOperation: ToggleHandler;
  onInspectOperation: (op: OperationDefinition, label: string, enabled: boolean, disabled: boolean) => void;
}) {
  const configuredOperations = useMemo(
    () => operations.filter((op) => op.status !== "recovered-only" && liveOperationIds.has(op.id)),
    []
  );

  const hiddenAgendaCover = useMemo(() => getOperation("HiddenAgenda"), []);
  const hiddenAgendaMembers = useMemo(
    () => configuredOperations.filter((op) => hiddenAgendaMemberIds.has(op.id)),
    [configuredOperations]
  );
  const deckOperations = useMemo(
    () => configuredOperations.filter((op) => !hiddenAgendaMemberIds.has(op.id) && op.id !== "HiddenAgenda"),
    [configuredOperations]
  );

  const hiddenAgendaEnabled = useMemo(
    () => hiddenAgendaMembers.some((op) => enabledIDs.has(op.id)),
    [hiddenAgendaMembers, enabledIDs]
  );
  const deckSize = deckOperations.length + 1;
  const activeCount = useMemo(
    () => deckOperations.filter((op) => enabledIDs.has(op.id)).length + (hiddenAgendaEnabled ? 1 : 0),
    [deckOperations, enabledIDs, hiddenAgendaEnabled]
  );

  const lockedOff = useCallback(
    (opId: string) => !canEdit || (enabledIDs.has(opId) && enabledIDs.size <= 1),
    [canEdit, enabledIDs]
  );

  return (
    <div className="ta-paper p-4">
      <div className="mb-3 flex items-end justify-between gap-3 border-b-2 border-black/15 pb-2.5">
        <div>
          <p className="ta-condensed text-xs tracking-[0.18em] uppercase text-ta-ink">OPERATIONS DECK</p>
          <p className="ta-sans text-xs leading-snug text-black/70">
            Dealt from one shuffled global deck. Click card to toggle, <strong>i</strong> for tactical briefing.
          </p>
        </div>
        <span className="ta-condensed shrink-0 whitespace-nowrap text-xs tracking-[0.12em] bg-black/10 px-2 py-0.5 border border-black/20">
          {activeCount} / {deckSize} ACTIVE
        </span>
      </div>
      <div className="grid gap-2.5 sm:grid-cols-2">
        {deckOperations.map((operation) => (
          <CompactOperationCard
            key={operation.id}
            operation={operation}
            label={packOperationIds.has(operation.id) ? "PACK 01" : operation.category.toUpperCase()}
            enabled={enabledIDs.has(operation.id)}
            disabled={lockedOff(operation.id)}
            onToggle={onToggleOperation}
            onInspect={onInspectOperation}
          />
        ))}
      </div>

      {/* Hidden Agenda Group */}
      <div className="mt-4 border-2 border-black/25 bg-black/[0.03] p-3.5">
        <p className="ta-condensed text-[0.68rem] tracking-[0.18em] text-black/70 uppercase mb-2.5">
          HIDDEN AGENDA EXPANSION
        </p>
        <div className="flex flex-col items-center">
          <div className="w-full max-w-sm mb-3">
            <CompactOperationCard
              operation={hiddenAgendaCover}
              label="MASTER COVER"
              enabled={hiddenAgendaEnabled}
              disabled={!canEdit}
              onToggle={onToggleOperation}
              onInspect={onInspectOperation}
            />
          </div>

          {/* Envelopes */}
          <div className="grid w-full gap-2.5 sm:grid-cols-2">
            {hiddenAgendaMembers.map((operation) => (
              <CompactOperationCard
                key={operation.id}
                operation={operation}
                label="ENVELOPE"
                enabled={enabledIDs.has(operation.id)}
                disabled={lockedOff(operation.id)}
                onToggle={onToggleOperation}
                onInspect={onInspectOperation}
              />
            ))}
          </div>
        </div>
      </div>
    </div>
  );
});

export function SettingsPanel({
  projection,
  pending = null,
  onSend,
  showHeader = true,
  error,
}: {
  projection: RoomProjection;
  pending?: PendingCommand | null;
  onSend(command: ClientCommand): void;
  showHeader?: boolean;
  error?: string | null;
}) {
  const isHost = projection.public.host_id === projection.private.player_id;
  const canEdit = isHost && projection.public.phase === "LOBBY";
  const controlsLocked = !canEdit || Boolean(pending);

  const [inspectedItem, setInspectedItem] = useState<InspectedItem>(null);

  const enabledIDs = useMemo(() => {
    return new Set(projection.public.settings.enabled_operations ?? []);
  }, [projection.public.settings.enabled_operations]);

  const enabledRoleIDs = useMemo(() => {
    return new Set(projection.public.settings.enabled_roles ?? []);
  }, [projection.public.settings.enabled_roles]);

  const currentSeconds = projection.public.settings.discussion_seconds ?? 300;
  const currentVirusCount = projection.public.settings.virus_count ?? 0;
  const isTimerActive = projection.public.settings.discussion_timer_enabled ?? true;

  const handleSetDuration = useCallback(
    (seconds: number) => {
      if (!canEdit) return;
      if (seconds < MIN_DISCUSSION_SECONDS) {
        onSend({
          kind: "lobby.discussion_timer",
          discussion_timer_enabled: false,
          discussion_seconds: currentSeconds,
        });
        return;
      }
      const clamped = Math.min(MAX_DISCUSSION_SECONDS, seconds);
      onSend({
        kind: "lobby.discussion_timer",
        discussion_timer_enabled: true,
        discussion_seconds: clamped,
      });
    },
    [canEdit, currentSeconds, onSend]
  );

  const handleSetVirusCount = useCallback(
    (count: number) => {
      if (!canEdit) return;
      const clamped = Math.min(MAX_VIRUS_COUNT, Math.max(MIN_VIRUS_COUNT, count));
      onSend({ kind: "lobby.virus_count", virus_count: clamped });
    },
    [canEdit, onSend]
  );

  const toggleOperation = useCallback<ToggleHandler>(
    (operationID, enabled) => {
      if (!canEdit) return;
      onSend({
        kind: "lobby.operation_enabled",
        operation_kind: operationID,
        operation_enabled: enabled,
      });
    },
    [canEdit, onSend]
  );

  const toggleRole = useCallback<ToggleHandler>(
    (roleID, enabled) => {
      if (!canEdit) return;
      onSend({
        kind: "lobby.role_enabled",
        role_id: roleID,
        role_enabled: enabled,
      });
    },
    [canEdit, onSend]
  );

  const handleInspectOperation = useCallback(
    (op: OperationDefinition, label: string, enabled: boolean, disabled: boolean) => {
      setInspectedItem({ type: "operation", operation: op, label, enabled, disabled });
    },
    []
  );

  const handleInspectRole = useCallback(
    (role: RoleDefinition, enabled: boolean, disabled: boolean) => {
      setInspectedItem({ type: "role", role, enabled, disabled });
    },
    []
  );

  const handleCloseModal = useCallback(() => {
    setInspectedItem(null);
  }, []);

  return (
    <div className="space-y-4">
      {showHeader ? <PaperTitle>Room settings</PaperTitle> : null}

      <MatchSetupSection
        isTimerActive={isTimerActive}
        currentSeconds={currentSeconds}
        currentVirusCount={currentVirusCount}
        controlsLocked={controlsLocked}
        isHost={isHost}
        onSetDuration={handleSetDuration}
        onSetVirusCount={handleSetVirusCount}
      />

      <SpecialRolesSection
        enabledRoleIDs={enabledRoleIDs}
        controlsLocked={controlsLocked}
        onToggleRole={toggleRole}
        onInspectRole={handleInspectRole}
      />

      <OperationsDeckSection
        enabledIDs={enabledIDs}
        canEdit={canEdit}
        onToggleOperation={toggleOperation}
        onInspectOperation={handleInspectOperation}
      />

      {error ? (
        <p className="ta-paper ta-sans border-l-4 border-ta-red px-3 py-2 text-sm leading-snug text-ta-red" role="alert">
          {error}
        </p>
      ) : null}

      <DossierModal
        item={inspectedItem}
        onClose={handleCloseModal}
        canToggle={canEdit}
        onToggle={(id, en) => {
          if (inspectedItem?.type === "operation") toggleOperation(id, en);
          else if (inspectedItem?.type === "role") toggleRole(id, en);
        }}
      />
    </div>
  );
}

export function SettingsScreen({ projection, pending, onSend, onClose, error = null }: SettingsScreenProps) {
  return (
    <div className="ta-rise ta-screen ta-screen--wide ta-screen--settings space-y-4">
      <SettingsPanel projection={projection} pending={pending} onSend={onSend} showHeader={true} error={error} />
      <InkButton className="w-full" onClick={onClose}>
        Back to game
      </InkButton>
    </div>
  );
}
