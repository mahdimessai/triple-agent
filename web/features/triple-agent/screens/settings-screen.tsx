import { memo, useCallback, useEffect, useMemo, useState } from "react";
import { createPortal } from "react-dom";
import { ArtStamp } from "@/components/ui/art-stamp";
import { PaperTitle } from "@/components/ui/paper-title";
import {
  operationCatalog,
  liveOperationIDs,
  hiddenAgendaMemberIDs,
  packOperationIDs,
  getOperation,
  type OperationDefinition,
} from "@/components/triple-agent/operation-catalog";
import { roleCatalog, type RoleDefinition } from "@/components/triple-agent/role-catalog";
import type { RoomProjection } from "@/components/triple-agent/server-client";
import type { CommandSender } from "@/features/triple-agent/model/screen";
import { operationBrief } from "@/features/triple-agent/operations/presentation";

const MIN_DISCUSSION_SECONDS = 60;
const MAX_DISCUSSION_SECONDS = 900;
const DISCUSSION_STEP_SECONDS = 30;
const MIN_VIRUS_COUNT = 0;
const MAX_VIRUS_COUNT = 4;

function formatDuration(seconds: number) {
  const minutes = Math.floor(seconds / 60);
  const remainder = seconds % 60;
  return remainder === 0 ? `${minutes}:00` : `${minutes}:${String(remainder).padStart(2, "0")}`;
}

const Stepper = memo(function Stepper({
  label,
  hint,
  value,
  onDecrease,
  onIncrease,
  canDecrease,
  canIncrease,
  disabled,
}: {
  label: string;
  hint: string;
  value: string;
  onDecrease: () => void;
  onIncrease: () => void;
  canDecrease: boolean;
  canIncrease: boolean;
  disabled: boolean;
}) {
  return (
    <div className="ta-stepper">
      <p className="ta-condensed text-xs font-bold tracking-[0.16em] text-black/75 uppercase">{label}</p>
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
      <p className="ta-condensed mt-2 text-xs leading-tight text-black/70">{hint}</p>
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
          <span className="ta-condensed block text-xs sm:text-sm font-bold leading-snug tracking-tight text-ta-ink uppercase">
            {operation.name}
          </span>
          <span className="ta-condensed block text-[0.6rem] font-bold tracking-wider text-black/60 uppercase">
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
          <span className="ta-condensed block text-xs sm:text-sm font-bold leading-snug tracking-tight text-ta-ink uppercase">
            {role.name}
          </span>
          <span
            className={`ta-condensed block text-[0.6rem] font-bold tracking-wider uppercase ${
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
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
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
              <p className={`ta-condensed text-xs font-bold tracking-[0.16em] uppercase ${tagColor}`}>{tag}</p>
              <h3 id="dossier-dialog-title" className="ta-display text-2xl leading-none text-ta-ink">
                {title}
              </h3>
            </div>
          </div>
          <button
            type="button"
            className="ta-secondary-button !min-h-0 px-2.5 py-1 text-sm font-bold hover:!bg-ta-red"
            onClick={onClose}
            aria-label="Close dossier"
          >
            ✕
          </button>
        </div>

        <div className="my-4">
          <p className="ta-condensed text-xs font-bold tracking-[0.16em] text-black/60 uppercase">TACTICAL BRIEFING</p>
          <p className="ta-condensed mt-1.5 text-base leading-relaxed text-black/90">{description}</p>
        </div>

        <div className="flex items-center justify-between border-t-2 border-black/25 pt-3.5">
          <span className="ta-condensed text-xs font-bold tracking-[0.14em] text-black/75">
            STATUS: {item.enabled ? "ENABLED IN MATCH" : "DISABLED"}
          </span>
          {canToggle && onToggle && !item.disabled ? (
            <button
              type="button"
              className={`ta-secondary-button !min-h-0 border-2 border-black px-4 py-1.5 text-xs font-bold uppercase tracking-wider ${
                item.enabled ? "hover:!bg-ta-red" : "hover:!bg-ta-teal"
              }`}
              onClick={() => {
                onToggle(id, !item.enabled);
                onClose();
              }}
            >
              {item.enabled ? "Disable" : "Enable"}
            </button>
          ) : (
            <span className="ta-condensed text-xs font-bold text-black/50 tracking-wider">HOST CONTROLLED</span>
          )}
        </div>
      </div>
    </div>
  );

  return createPortal(modalContent, document.body);
}

// 1. Isolated & Memoized Match Setup Steppers
const MatchSetupSection = memo(function MatchSetupSection({
  isTimerActive,
  currentSeconds,
  currentVirusCount,
  controlsLocked,
  pending,
  liveSession,
  isHost,
  onSetDuration,
  onSetVirusCount,
}: {
  isTimerActive: boolean;
  currentSeconds: number;
  currentVirusCount: number;
  controlsLocked: boolean;
  pending: boolean;
  liveSession: boolean;
  isHost: boolean;
  onSetDuration: (seconds: number) => void;
  onSetVirusCount: (count: number) => void;
}) {
  return (
    <div className="ta-paper p-4">
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2 border-b-2 border-black/15 pb-2.5">
        <div>
          <p className="ta-condensed text-xs font-bold tracking-[0.16em] uppercase text-ta-ink">MATCH SETUP</p>
          <p className="ta-condensed text-xs text-black/70">
            {pending
              ? "Saving the room setting…"
              : controlsLocked
              ? "Live settings (Host controlled)"
              : "Set the discussion length and how many agents start as VIRUS."}
          </p>
        </div>
        {liveSession && !isHost ? (
          <span className="ta-condensed text-[0.65rem] font-bold tracking-wider uppercase px-2 py-0.5 bg-black/10 border-2 border-black/25">
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

// 2. Isolated & Memoized Special Roles Section
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
    () => roleCatalog.filter((role) => enabledRoleIDs.has(role.id)).length,
    [enabledRoleIDs]
  );

  return (
    <div className="ta-paper p-4">
      <div className="mb-3 flex items-end justify-between gap-3 border-b-2 border-black/15 pb-2.5">
        <div>
          <p className="ta-condensed text-xs font-bold tracking-[0.16em] uppercase text-ta-ink">SPECIAL ROLES</p>
          <p className="ta-condensed mt-0.5 text-xs leading-tight text-black/70">
            Secret roles dealt on top of agency. Click card to toggle, <strong>i</strong> for dossier.
          </p>
        </div>
        <span className="ta-condensed shrink-0 text-xs font-bold tracking-[0.12em] bg-black/10 px-2 py-0.5 border border-black/20">
          {activeRoleCount} / {roleCatalog.length} IN POOL
        </span>
      </div>
      <div className="grid gap-2.5 sm:grid-cols-2">
        {roleCatalog.map((role) => (
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

// 3. Isolated & Memoized Operations Deck Section
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
    () => operationCatalog.filter((op) => op.status !== "recovered-only" && liveOperationIDs.has(op.id)),
    []
  );

  const hiddenAgendaCover = useMemo(() => getOperation("HiddenAgenda"), []);
  const hiddenAgendaMembers = useMemo(
    () => configuredOperations.filter((op) => hiddenAgendaMemberIDs.has(op.id)),
    [configuredOperations]
  );
  const deckOperations = useMemo(
    () => configuredOperations.filter((op) => !hiddenAgendaMemberIDs.has(op.id) && op.id !== "HiddenAgenda"),
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
          <p className="ta-condensed text-xs font-bold tracking-[0.16em] uppercase text-ta-ink">OPERATIONS DECK</p>
          <p className="ta-condensed text-xs text-black/70">
            Dealt from one shuffled global deck. Click card to toggle, <strong>i</strong> for tactical briefing.
          </p>
        </div>
        <span className="ta-condensed text-xs font-bold tracking-[0.12em] bg-black/10 px-2 py-0.5 border border-black/20">
          {activeCount} / {deckSize} ACTIVE
        </span>
      </div>
      <div className="grid gap-2.5 sm:grid-cols-2">
        {deckOperations.map((operation) => (
          <CompactOperationCard
            key={operation.id}
            operation={operation}
            label={packOperationIDs.has(operation.id) ? "PACK 01" : operation.category.toUpperCase()}
            enabled={enabledIDs.has(operation.id)}
            disabled={lockedOff(operation.id)}
            onToggle={onToggleOperation}
            onInspect={onInspectOperation}
          />
        ))}
      </div>

      {/* Hidden Agenda Group */}
      <div className="mt-4 border-2 border-black/25 bg-black/[0.03] p-3.5">
        <p className="ta-condensed text-[0.68rem] font-bold tracking-[0.18em] text-black/70 uppercase mb-2.5">
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
  timerEnabled,
  setTimerEnabled,
  projection,
  liveSession = false,
  isHost = false,
  onCommand,
  pending = false,
  error,
  showHeader = true,
}: {
  timerEnabled: boolean;
  setTimerEnabled: (value: boolean) => void;
  projection?: RoomProjection;
  liveSession?: boolean;
  isHost?: boolean;
  onCommand?: CommandSender;
  pending?: boolean;
  error?: string;
  showHeader?: boolean;
}) {
  const canEdit = Boolean(liveSession && isHost && projection?.public.phase === "LOBBY");
  const controlsLocked = liveSession && (!canEdit || pending);

  const [inspectedItem, setInspectedItem] = useState<InspectedItem>(null);

  // Stable memoized sets for enabled IDs
  const enabledIDs = useMemo(() => {
    if (projection?.public.settings.enabled_operations) {
      return new Set(projection.public.settings.enabled_operations);
    }
    return new Set(
      operationCatalog.filter((operation) => operation.status === "enabled").map((operation) => operation.id)
    );
  }, [projection?.public.settings.enabled_operations]);

  const [localRoles, setLocalRoles] = useState<Set<string>>(() => new Set());
  const enabledRoleIDs = useMemo(() => {
    if (projection?.public.settings.enabled_roles) {
      return new Set(projection.public.settings.enabled_roles);
    }
    return localRoles;
  }, [projection?.public.settings.enabled_roles, localRoles]);

  const [localSeconds, setLocalSeconds] = useState(300);
  const [localVirusCount, setLocalVirusCount] = useState(0);
  const currentSeconds = projection?.public.settings.discussion_seconds ?? localSeconds;
  const currentVirusCount = projection?.public.settings.virus_count ?? localVirusCount;
  const isTimerActive = projection ? projection.public.settings.discussion_timer_enabled : timerEnabled;

  const handleSetDuration = useCallback(
    (seconds: number) => {
      if (seconds < MIN_DISCUSSION_SECONDS) {
        if (canEdit) {
          onCommand?.("lobby.discussion_timer", {
            discussionTimerEnabled: false,
            discussionSeconds: currentSeconds,
          });
        } else if (!liveSession) {
          setTimerEnabled(false);
        }
        return;
      }
      const clamped = Math.min(MAX_DISCUSSION_SECONDS, seconds);
      if (canEdit) {
        onCommand?.("lobby.discussion_timer", {
          discussionTimerEnabled: true,
          discussionSeconds: clamped,
        });
      } else if (!liveSession) {
        setTimerEnabled(true);
        setLocalSeconds(clamped);
      }
    },
    [canEdit, currentSeconds, liveSession, onCommand, setTimerEnabled]
  );

  const handleSetVirusCount = useCallback(
    (count: number) => {
      const clamped = Math.min(MAX_VIRUS_COUNT, Math.max(MIN_VIRUS_COUNT, count));
      if (canEdit) {
        onCommand?.("lobby.virus_count", { virusCount: clamped });
      } else if (!liveSession) {
        setLocalVirusCount(clamped);
      }
    },
    [canEdit, liveSession, onCommand]
  );

  const toggleOperation = useCallback<ToggleHandler>(
    (operationID, enabled) => {
      if (!canEdit) return;
      onCommand?.("lobby.operation_enabled", { operationKind: operationID, operationEnabled: enabled });
    },
    [canEdit, onCommand]
  );

  const toggleRole = useCallback<ToggleHandler>(
    (roleID, enabled) => {
      if (canEdit) {
        onCommand?.("lobby.role_enabled", { roleId: roleID, roleEnabled: enabled });
      } else if (!liveSession) {
        setLocalRoles((current) => {
          const next = new Set(current);
          if (next.has(roleID)) next.delete(roleID);
          else next.add(roleID);
          return next;
        });
      }
    },
    [canEdit, liveSession, onCommand]
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
        pending={pending}
        liveSession={liveSession}
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

      {error ? <p className="ta-condensed text-sm text-white">{error}</p> : null}

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

export function SettingsScreen(props: {
  timerEnabled: boolean;
  setTimerEnabled: (value: boolean) => void;
  projection?: RoomProjection;
  liveSession?: boolean;
  isHost?: boolean;
  onCommand?: CommandSender;
  pending?: boolean;
  error?: string;
}) {
  return (
    <div className="ta-rise ta-screen ta-screen--wide ta-screen--settings space-y-4">
      <SettingsPanel {...props} showHeader={true} />
    </div>
  );
}
