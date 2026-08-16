import { useState } from "react";
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
function OperationCard({ operation, label, enabled, disabled, onToggle }: { operation: OperationDefinition; label: string; enabled: boolean; disabled: boolean; onToggle: () => void }) {
  return (
    <button
      className="ta-operation-card"
      data-enabled={enabled}
      disabled={disabled}
      onClick={onToggle}
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
      <span className="ta-operation-card-brief">{operationBrief(operation)}</span>
      <span className="ta-operation-card-footer">
        <span className="ta-condensed min-w-0 truncate text-[0.62rem] tracking-[0.08em] text-black/55">{operation.input}</span>
        <span className="ta-condensed shrink-0 text-[0.62rem] tracking-[0.1em]">{enabled ? "ENABLED" : "DISABLED"}</span>
      </span>
    </button>
  );
}

export function SettingsScreen({ timerEnabled, setTimerEnabled, projection, liveSession = false, isHost = false, onCommand, error }: { timerEnabled: boolean; setTimerEnabled: (value: boolean) => void; projection?: RoomProjection; liveSession?: boolean; isHost?: boolean; onCommand?: CommandSender; error?: string }) {
  const enabledIDs = new Set(projection?.public.settings.enabled_operations ?? operationCatalog.filter((operation) => operation.status === "enabled").map((operation) => operation.id));
  const canEdit = Boolean(liveSession && isHost && projection?.public.phase === "LOBBY");
  const configuredOperations = operationCatalog.filter((operation) => operation.status !== "recovered-only" && liveOperationIDs.has(operation.id));

  // The deck reads top to bottom in the order a host cares about: the plain
  // operations that ship on, then the one Hidden Agenda slot with its envelopes
  // underneath it, then Expansion Pack 01, which ships off, at the very end.
  const hiddenAgendaCover = getOperation("HiddenAgenda");
  const hiddenAgendaMembers = configuredOperations.filter((operation) => hiddenAgendaMemberIDs.has(operation.id));
  const standardOperations = configuredOperations.filter((operation) => !hiddenAgendaMemberIDs.has(operation.id) && !packOperationIDs.has(operation.id));
  const packOperations = configuredOperations.filter((operation) => packOperationIDs.has(operation.id));

  // Hidden Agenda counts as one operation in the pool, so the deck counter and
  // the cover's own state follow the group rather than its members.
  const hiddenAgendaEnabled = hiddenAgendaMembers.some((operation) => enabledIDs.has(operation.id));
  const deckSize = standardOperations.length + packOperations.length + 1;
  const activeCount = standardOperations.filter((operation) => enabledIDs.has(operation.id)).length + packOperations.filter((operation) => enabledIDs.has(operation.id)).length + (hiddenAgendaEnabled ? 1 : 0);

  function toggleOperation(operationID: string) {
    if (!canEdit) return;
    onCommand?.("lobby.operation_enabled", { operationKind: operationID, operationEnabled: !enabledIDs.has(operationID) });
  }

  // The server takes the cover as one command and flips every envelope with it.
  function toggleHiddenAgenda() {
    if (!canEdit) return;
    onCommand?.("lobby.operation_enabled", { operationKind: "HiddenAgenda", operationEnabled: !hiddenAgendaEnabled });
  }

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
  const [localRoles, setLocalRoles] = useState<Set<string>>(() => new Set());
  const enabledRoleIDs = new Set(projection?.public.settings.enabled_roles ?? [...localRoles]);
  const controlsLocked = liveSession && !canEdit;

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

  function toggleRole(roleID: string) {
    const turningOn = !enabledRoleIDs.has(roleID);
    if (canEdit) {
      onCommand?.("lobby.role_enabled", { roleId: roleID, roleEnabled: turningOn });
    } else if (!liveSession) {
      setLocalRoles((current) => {
        const next = new Set(current);
        if (next.has(roleID)) next.delete(roleID);
        else next.add(roleID);
        return next;
      });
    }
  }

  return (
    <div className="ta-rise ta-screen ta-screen--settings space-y-4">
      <PaperTitle>Room settings</PaperTitle>

      {/* Match setup: the two numbers a host actually tunes, side by side. */}
      <div className="ta-paper p-4">
        <div className="mb-4">
          <p className="ta-condensed text-xs tracking-[0.16em]">MATCH SETUP</p>
          <p className="ta-condensed text-sm">{controlsLocked ? "Only the host can change these before the match starts." : "Set the discussion length and how many agents work for VIRUS."}</p>
        </div>
        <div className="grid gap-3 sm:grid-cols-2">
          <Stepper
            label="DISCUSSION TIMER"
            value={timerEnabled ? formatDuration(currentSeconds) : "OFF"}
            hint={timerEnabled ? `Step below ${formatDuration(MIN_DISCUSSION_SECONDS)} to run discussion untimed. Longest is ${formatDuration(MAX_DISCUSSION_SECONDS)}.` : "Discussion runs untimed until the host moves the room on. Step up to put it back on a clock."}
            disabled={controlsLocked}
            canDecrease={timerEnabled}
            canIncrease={!timerEnabled || currentSeconds < MAX_DISCUSSION_SECONDS}
            onDecrease={() => setDuration(currentSeconds - DISCUSSION_STEP_SECONDS)}
            onIncrease={() => setDuration(timerEnabled ? currentSeconds + DISCUSSION_STEP_SECONDS : currentSeconds)}
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
            return (
              <button
                className="ta-operation-card"
                data-enabled={enabled}
                disabled={controlsLocked}
                key={role.id}
                onClick={() => toggleRole(role.id)}
                type="button"
                aria-pressed={enabled}
                aria-label={`${enabled ? "Remove" : "Add"} ${role.name} ${enabled ? "from" : "to"} the role pool`}
              >
                <span className="ta-operation-card-art" aria-hidden="true">
                  <ArtStamp artName={role.artName} alt="" className="h-16 w-20 object-contain" />
                </span>
                <span className="ta-operation-card-heading">
                  <span className="ta-display text-base leading-none">{role.name}</span>
                  <span className={`ta-condensed text-[0.58rem] tracking-[0.12em] ${role.faction === "VIRUS" ? "text-ta-red" : "text-[#1d5b79]"}`}>{role.faction}</span>
                </span>
                <span className="ta-operation-card-brief">{role.effect}</span>
                <span className="ta-operation-card-footer">
                  <span className="ta-condensed min-w-0 truncate text-[0.62rem] tracking-[0.08em] text-black/55">Dealt to a {role.faction === "VIRUS" ? "VIRUS" : "SERVICE"} agent</span>
                  <span className="ta-condensed shrink-0 text-[0.62rem] tracking-[0.1em]">{enabled ? "ENABLED" : "DISABLED"}</span>
                </span>
              </button>
            );
          })}
        </div>
      </div>

      {/* Every operation the room can deal, in one deck. Hidden Agenda holds a
          single slot with its envelopes nested under it, and Expansion Pack 01,
          which ships off, sits at the end. */}
      <div className="ta-paper p-4">
        <div className="mb-4 flex items-end justify-between gap-3">
          <div>
            <p className="ta-condensed text-xs tracking-[0.16em]">OPERATIONS DECK</p>
            <p className="ta-condensed text-sm">Every enabled operation can be dealt during a match interlude. Dimmed cards are switched off.</p>
          </div>
          <span className="ta-condensed text-xs tracking-[0.12em]">
            {activeCount} / {deckSize} ACTIVE
          </span>
        </div>
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {standardOperations.map((operation) => (
            <OperationCard
              key={operation.id}
              operation={operation}
              label={operation.category.toUpperCase()}
              enabled={enabledIDs.has(operation.id)}
              disabled={lockedOff(operation.id)}
              onToggle={() => toggleOperation(operation.id)}
            />
          ))}
        </div>

        {/* Hidden Agenda is one operation with several possible contents, so it
            takes one slot in the deck and its envelopes live inside that slot
            instead of competing with the named operations for a draw. */}
        <div className="mt-4 border-2 border-black/25 p-3">
          <div className="mb-3 flex items-end justify-between gap-3">
            <div>
              <p className="ta-condensed text-xs tracking-[0.16em]">HIDDEN AGENDA</p>
              <p className="ta-condensed mt-1 max-w-prose text-sm leading-tight">
                One operation dealt under one cover. The room only hears that new orders arrived; the server picks which of the envelopes below the recipient
                actually opens. Switch the cover off to keep every hidden agenda out of the match, or leave individual envelopes off to narrow what it can be.
              </p>
            </div>
            <span className="ta-condensed shrink-0 text-xs tracking-[0.12em]">
              {hiddenAgendaMembers.filter((operation) => enabledIDs.has(operation.id)).length} / {hiddenAgendaMembers.length} ENVELOPES
            </span>
          </div>
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            <OperationCard
              operation={hiddenAgendaCover}
              label="THE COVER"
              enabled={hiddenAgendaEnabled}
              disabled={!canEdit}
              onToggle={toggleHiddenAgenda}
            />
            {hiddenAgendaMembers.map((operation) => (
              <OperationCard
                key={operation.id}
                operation={operation}
                label="ENVELOPE"
                enabled={enabledIDs.has(operation.id)}
                disabled={lockedOff(operation.id)}
                onToggle={() => toggleOperation(operation.id)}
              />
            ))}
          </div>
        </div>

        {/* Pack 01 ships off, so it sits last and reads as dimmed until a host
            switches something on. */}
        <div className="mt-4 border-2 border-black/25 p-3">
          <div className="mb-3 flex items-end justify-between gap-3">
            <div>
              <p className="ta-condensed text-xs tracking-[0.16em]">EXPANSION PACK 01</p>
              <p className="ta-condensed mt-1 max-w-prose text-sm leading-tight">Off by default. Switch these on to add them to the deck for this room.</p>
            </div>
            <span className="ta-condensed shrink-0 text-xs tracking-[0.12em]">
              {packOperations.filter((operation) => enabledIDs.has(operation.id)).length} / {packOperations.length} ACTIVE
            </span>
          </div>
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {packOperations.map((operation) => (
              <OperationCard
                key={operation.id}
                operation={operation}
                label="PACK 01"
                enabled={enabledIDs.has(operation.id)}
                disabled={lockedOff(operation.id)}
                onToggle={() => toggleOperation(operation.id)}
              />
            ))}
          </div>
        </div>
      </div>

      {error ? <p className="ta-condensed text-sm text-white">{error}</p> : null}
    </div>
  );
}
