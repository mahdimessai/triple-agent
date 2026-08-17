import { memo, useCallback, useState } from "react";
import { ArtStamp } from "@/components/ui/art-stamp";
import { PaperTitle } from "@/components/ui/paper-title";
import { operationCatalog, liveOperationIDs, hiddenAgendaMemberIDs, packOperationIDs, getOperation, type OperationDefinition } from "@/components/triple-agent/operation-catalog";
import { roleCatalog } from "@/components/triple-agent/role-catalog";
import type { RoomProjection } from "@/components/triple-agent/server-client";
import type { CommandSender } from "@/features/triple-agent/model/screen";
import { operationBrief } from "@/features/triple-agent/operations/presentation";

// The server clamps discussion length to the same window, so the stepper can
// never offer a value the room will silently reject.
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

/**
 * One boxed value with a decrement and an increment arrow. Arrows disable at
 * the bounds instead of wrapping, so a host never rolls a maximum back to a
 * minimum by pressing once too often.
 */
function Stepper({ label, hint, value, onDecrease, onIncrease, canDecrease, canIncrease, disabled }: { label: string; hint: string; value: string; onDecrease: () => void; onIncrease: () => void; canDecrease: boolean; canIncrease: boolean; disabled: boolean }) {
  return (
    <div className="ta-stepper">
      <p className="ta-condensed text-xs tracking-[0.16em] text-black/65">{label}</p>
      <div className="mt-2 flex items-stretch gap-2">
        <button className="ta-stepper-arrow" type="button" aria-label={`Decrease ${label.toLowerCase()}`} disabled={disabled || !canDecrease} onClick={onDecrease}>
          &#9664;
        </button>
        <output className="ta-stepper-value ta-display">{value}</output>
        <button className="ta-stepper-arrow" type="button" aria-label={`Increase ${label.toLowerCase()}`} disabled={disabled || !canIncrease} onClick={onIncrease}>
          &#9654;
        </button>
      </div>
      <p className="ta-condensed mt-2 text-sm leading-tight text-black/70">{hint}</p>
    </div>
  );
}

/**
 * One toggleable card in the operations deck. The Hidden Agenda cover, its
 * envelopes and the pack operations all render through this, so a card reads
 * the same wherever it sits in the deck.
 */
type ToggleHandler = (id: string, enabled: boolean) => void;

const OperationCard = memo(function OperationCard({ operation, label, enabled, disabled, onToggle }: { operation: OperationDefinition; label: string; enabled: boolean; disabled: boolean; onToggle: ToggleHandler }) {
  return (
    <button
      className="ta-operation-card"
      data-enabled={enabled}
      disabled={disabled}
      onClick={() => onToggle(operation.id, !enabled)}
      type="button"
      aria-pressed={enabled}
      aria-label={`${enabled ? "Disable" : "Enable"} ${operation.name}`}
    >
      <span className="ta-operation-card-art" aria-hidden="true">
        <ArtStamp artName={operation.artName} alt="" className="h-16 w-20 object-contain" />
      </span>
      <span className="ta-operation-card-heading">
        <span className="ta-display text-base leading-none">{operation.name}</span>
        <span className="ta-condensed text-[0.58rem] tracking-[0.12em] text-black/55">{label}</span>
      </span>
      <span className="ta-operation-card-brief ta-scrollbar">{operationBrief(operation)}</span>
      <span className="ta-operation-card-footer justify-center">
        <span className="ta-condensed shrink-0 text-[0.62rem] tracking-[0.1em]">{enabled ? "ENABLED" : "DISABLED"}</span>
      </span>
    </button>
  );
});

const RoleCard = memo(function RoleCard({ role, enabled, disabled, onToggle }: { role: (typeof roleCatalog)[number]; enabled: boolean; disabled: boolean; onToggle: ToggleHandler }) {
  return (
    <button
      className="ta-operation-card"
      data-enabled={enabled}
      disabled={disabled}
      type="button"
      aria-pressed={enabled}
      aria-label={`${enabled ? "Remove" : "Add"} ${role.name} ${enabled ? "from" : "to"} the role pool`}
      onClick={() => onToggle(role.id, !enabled)}
    >
      <span className="ta-operation-card-art" aria-hidden="true">
        <ArtStamp artName={role.artName} alt="" className="h-16 w-20 object-contain" />
      </span>
      <span className="ta-operation-card-heading">
        <span className="ta-display text-base leading-none">{role.name}</span>
        <span className={`ta-condensed text-[0.58rem] tracking-[0.12em] ${role.faction === "VIRUS" ? "text-ta-red" : "text-[#1d5b79]"}`}>{role.faction}</span>
      </span>
      <span className="ta-operation-card-brief">{role.effect}</span>
      <span className="ta-operation-card-footer justify-center">
        <span className="ta-condensed shrink-0 text-[0.62rem] tracking-[0.1em]">{enabled ? "ENABLED" : "DISABLED"}</span>
      </span>
    </button>
  );
});

export function SettingsScreen({ timerEnabled, setTimerEnabled, projection, liveSession = false, isHost = false, onCommand, pending = false, error }: { timerEnabled: boolean; setTimerEnabled: (value: boolean) => void; projection?: RoomProjection; liveSession?: boolean; isHost?: boolean; onCommand?: CommandSender; pending?: boolean; error?: string }) {
  const enabledIDs = new Set(projection?.public.settings.enabled_operations ?? operationCatalog.filter((operation) => operation.status === "enabled").map((operation) => operation.id));
  const canEdit = Boolean(liveSession && isHost && projection?.public.phase === "LOBBY");
  const configuredOperations = operationCatalog.filter((operation) => operation.status !== "recovered-only" && liveOperationIDs.has(operation.id));

  // Operations deck contains all standard and expansion operations together.
  // Hidden Agenda is one group dealt under one cover, so its envelopes live in
  // the Hidden Agenda section below where they emanate from the master cover.
  const hiddenAgendaCover = getOperation("HiddenAgenda");
  const hiddenAgendaMembers = configuredOperations.filter((operation) => hiddenAgendaMemberIDs.has(operation.id));
  const deckOperations = configuredOperations.filter((operation) => !hiddenAgendaMemberIDs.has(operation.id) && operation.id !== "HiddenAgenda");

  // Sort operations by enabled status first so active operations are at the top of the deck
  const sortedDeckOperations = [...deckOperations].sort((a, b) => {
    const aEnabled = enabledIDs.has(a.id);
    const bEnabled = enabledIDs.has(b.id);
    if (aEnabled && !bEnabled) return -1;
    if (!aEnabled && bEnabled) return 1;
    return 0;
  });

  const sortedHiddenAgendaMembers = [...hiddenAgendaMembers].sort((a, b) => {
    const aEnabled = enabledIDs.has(a.id);
    const bEnabled = enabledIDs.has(b.id);
    if (aEnabled && !bEnabled) return -1;
    if (!aEnabled && bEnabled) return 1;
    return 0;
  });

  // Hidden Agenda counts as one operation in the pool, so the deck counter and
  // the cover's own state follow the group rather than its members.
  const hiddenAgendaEnabled = hiddenAgendaMembers.some((operation) => enabledIDs.has(operation.id));
  const deckSize = deckOperations.length + 1;
  const activeCount = deckOperations.filter((operation) => enabledIDs.has(operation.id)).length + (hiddenAgendaEnabled ? 1 : 0);

  const toggleOperation = useCallback<ToggleHandler>((operationID, enabled) => {
    if (!canEdit) return;
    onCommand?.("lobby.operation_enabled", { operationKind: operationID, operationEnabled: enabled });
  }, [canEdit, onCommand]);

  // The room needs at least one operation left to deal, so the last enabled
  // card locks rather than emptying the deck.
  function lockedOff(operationID: string) {
    return !canEdit || (enabledIDs.has(operationID) && enabledIDs.size <= 1);
  }

  // Outside a live room there is no server to hold these, so the screen keeps
  // its own copy and stays interactive in the workbench.
  const [localSeconds, setLocalSeconds] = useState(300);
  const [localVirusCount, setLocalVirusCount] = useState(0);
  const currentSeconds = projection?.public.settings.discussion_seconds ?? localSeconds;
  const currentVirusCount = projection?.public.settings.virus_count ?? localVirusCount;
  const isTimerActive = projection ? projection.public.settings.discussion_timer_enabled : timerEnabled;
  const [localRoles, setLocalRoles] = useState<Set<string>>(() => new Set());
  const enabledRoleIDs = new Set(projection?.public.settings.enabled_roles ?? [...localRoles]);
  const controlsLocked = liveSession && (!canEdit || pending);

  /**
   * One step below the minimum turns the timer off rather than clamping, so the
   * left arrow keeps meaning "less time" all the way down and the OFF state
   * needs no separate control.
   */
  function setDuration(seconds: number) {
    if (seconds < MIN_DISCUSSION_SECONDS) {
      if (canEdit) onCommand?.("lobby.discussion_timer", { discussionTimerEnabled: false, discussionSeconds: currentSeconds });
      else if (!liveSession) setTimerEnabled(false);
      return;
    }
    const clamped = Math.min(MAX_DISCUSSION_SECONDS, seconds);
    if (canEdit) {
      onCommand?.("lobby.discussion_timer", { discussionTimerEnabled: true, discussionSeconds: clamped });
    } else if (!liveSession) {
      setTimerEnabled(true);
      setLocalSeconds(clamped);
    }
  }

  function setVirusCount(count: number) {
    const clamped = Math.min(MAX_VIRUS_COUNT, Math.max(MIN_VIRUS_COUNT, count));
    if (canEdit) {
      onCommand?.("lobby.virus_count", { virusCount: clamped });
    } else if (!liveSession) {
      setLocalVirusCount(clamped);
    }
  }

  const toggleRole = useCallback<ToggleHandler>((roleID, enabled) => {
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
  }, [canEdit, liveSession, onCommand]);

  return (
    <div className="ta-rise ta-screen ta-screen--settings space-y-4">
      <PaperTitle>Room settings</PaperTitle>

      {/* Match setup: the two numbers a host actually tunes, side by side. */}
      <div className="ta-paper p-4">
        <div className="mb-4">
          <p className="ta-condensed text-xs tracking-[0.16em]">MATCH SETUP</p>
            <p className="ta-condensed text-sm">{pending ? "Saving the room setting…" : controlsLocked ? "Only the host can change these before the match starts." : "Set the discussion length and how many agents work for VIRUS."}</p>
        </div>
        <div className="grid gap-3 sm:grid-cols-2">
          <Stepper
            label="DISCUSSION TIMER"
            value={isTimerActive ? formatDuration(currentSeconds) : "OFF"}
            hint={isTimerActive ? `Step below ${formatDuration(MIN_DISCUSSION_SECONDS)} to run discussion untimed. Longest is ${formatDuration(MAX_DISCUSSION_SECONDS)}.` : "Discussion runs untimed until the host moves the room on. Step up to put it back on a clock."}
            disabled={controlsLocked}
            canDecrease={isTimerActive}
            canIncrease={!isTimerActive || currentSeconds < MAX_DISCUSSION_SECONDS}
            onDecrease={() => setDuration(currentSeconds - DISCUSSION_STEP_SECONDS)}
            onIncrease={() => setDuration(isTimerActive ? currentSeconds + DISCUSSION_STEP_SECONDS : Math.max(MIN_DISCUSSION_SECONDS, currentSeconds))}
          />
          <Stepper
            label="VIRUS TEAM SIZE"
            value={currentVirusCount === 0 ? "AUTO" : String(currentVirusCount)}
            hint={currentVirusCount === 0 ? "Scales with the table: 2 VIRUS agents for 5 to 6 players, 3 for 7 or more." : `${currentVirusCount} agent${currentVirusCount === 1 ? "" : "s"} start the match working for VIRUS.`}
            disabled={controlsLocked}
            canDecrease={currentVirusCount > MIN_VIRUS_COUNT}
            canIncrease={currentVirusCount < MAX_VIRUS_COUNT}
            onDecrease={() => setVirusCount(currentVirusCount - 1)}
            onIncrease={() => setVirusCount(currentVirusCount + 1)}
          />
        </div>
      </div>

      {/* Special roles, dealt on top of an agency and toggled one by one. */}
      <div className="ta-paper p-4">
        <div className="mb-4 flex items-end justify-between gap-3">
          <div>
            <p className="ta-condensed text-xs tracking-[0.16em]">SPECIAL ROLES</p>
            <p className="ta-condensed mt-1 max-w-prose text-sm leading-tight">
              A secret role dealt on top of an agency at the start of the match. It never changes who an agent wins with, only what the table can learn about them or
              do to them. Leave the pool empty to play without roles.
            </p>
          </div>
          <span className="ta-condensed shrink-0 text-xs tracking-[0.12em]">
            {roleCatalog.filter((role) => enabledRoleIDs.has(role.id)).length} / {roleCatalog.length} IN THE POOL
          </span>
        </div>
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {roleCatalog.map((role) => {
            const enabled = enabledRoleIDs.has(role.id);
            return <RoleCard key={role.id} role={role} enabled={enabled} disabled={controlsLocked} onToggle={toggleRole} />;
          })}
        </div>
      </div>

      {/* Every operation the room can deal in the main operations deck,
          plus Hidden Agenda centered below with all envelopes emanating from it. */}
      <div className="ta-paper p-4">
        <div className="mb-4 flex items-end justify-between gap-3">
          <div>
            <p className="ta-condensed text-xs tracking-[0.16em]">OPERATIONS DECK</p>
              <p className="ta-condensed text-sm">Enabled operations are dealt from one shuffled global deck before the deck cycles. Dimmed cards are switched off.</p>
          </div>
          <span className="ta-condensed text-xs tracking-[0.12em]">
            {activeCount} / {deckSize} ACTIVE
          </span>
        </div>
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {sortedDeckOperations.map((operation) => (
            <OperationCard
              key={operation.id}
              operation={operation}
              label={packOperationIDs.has(operation.id) ? "PACK 01" : operation.category.toUpperCase()}
              enabled={enabledIDs.has(operation.id)}
              disabled={lockedOff(operation.id)}
              onToggle={toggleOperation}
            />
          ))}
        </div>

        {/* Hidden Agenda is centered in the box with all envelopes emanating from it. */}
        <div className="mt-6 border-2 border-black/25 bg-black/[0.02] p-4">
          <div className="mb-4 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
          </div>

          <div className="flex flex-col items-center">
            {/* Master Cover Card in the Middle */}
            <div className="w-full max-w-xs">
              <OperationCard
                operation={hiddenAgendaCover}
                label="THE MASTER COVER"
                enabled={hiddenAgendaEnabled}
                disabled={!canEdit}
                onToggle={toggleOperation}
              />
            </div>

            {/* Visual connector lines emanating from cover. The tick row mirrors the
                envelope grid's columns so the legs land under the cards at every width. */}
            <div className="my-3 flex w-full flex-col items-center" aria-hidden="true">
              <div className="h-4 w-0.5 bg-black/40" />
              {/* One column: a bare stem reads better than a bracket over a single stack. */}
              <div className="hidden w-full flex-col items-center sm:flex">
                <div className="h-2 w-full border-t-2 border-black/40" />
                <div className="grid h-2 w-full grid-cols-2 gap-3 overflow-hidden lg:grid-cols-3 xl:grid-cols-5">
                  {sortedHiddenAgendaMembers.map((operation) => (
                    <div className="flex justify-center" key={operation.id}>
                      <div className="h-2 w-0.5 bg-black/40" />
                    </div>
                  ))}
                </div>
              </div>
              <span className="ta-condensed mt-1 max-w-full text-center text-[0.6rem] font-bold tracking-[0.12em] text-black/60 uppercase sm:text-[0.65rem] sm:tracking-[0.2em]">
              </span>
            </div>

            {/* Envelopes radiating/emanating below */}
            <div className="grid w-full gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
              {sortedHiddenAgendaMembers.map((operation) => (
                <OperationCard
                  key={operation.id}
                  operation={operation}
                  label="ENVELOPE"
                  enabled={enabledIDs.has(operation.id)}
                  disabled={lockedOff(operation.id)}
                  onToggle={toggleOperation}
                />
              ))}
            </div>
          </div>
        </div>
      </div>

      {error ? <p className="ta-condensed text-sm text-white">{error}</p> : null}
    </div>
  );
}
