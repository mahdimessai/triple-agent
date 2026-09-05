"use client";

import { useState } from "react";
import type { ClientCommand, RoomProjection } from "../protocol";
import type { PendingCommand } from "../use-room";
import {
  getOperation,
  hiddenAgendaMemberIds,
  liveOperationIds,
  operations,
  packOperationIds,
  type OperationDefinition,
} from "../operations";
import { roles, type RoleDefinition } from "../roles";
import { InkButton, PaperTitle } from "../ui";
import { DossierDialog } from "./settings/dossier-dialog";
import { OperationOptionCard, RoleOptionCard } from "./settings/option-cards";
import { SettingStepper } from "./settings/setting-stepper";
import type { InspectedItem, ToggleHandler } from "./settings/types";

const MIN_DISCUSSION_SECONDS = 60;
const MAX_DISCUSSION_SECONDS = 900;
const DISCUSSION_STEP_SECONDS = 30;
const MIN_VIRUS_COUNT = 0;
const MAX_VIRUS_COUNT = 4;

const CONFIGURED_OPERATIONS = operations.filter((operation) => operation.status !== "recovered-only" && liveOperationIds.has(operation.id));
const HIDDEN_AGENDA_COVER = getOperation("HiddenAgenda");
const HIDDEN_AGENDA_MEMBERS = CONFIGURED_OPERATIONS.filter((operation) => hiddenAgendaMemberIds.has(operation.id));
const DECK_OPERATIONS = CONFIGURED_OPERATIONS.filter((operation) => !hiddenAgendaMemberIds.has(operation.id) && operation.id !== "HiddenAgenda");

export type SettingsScreenProps = {
  projection: RoomProjection;
  pending: PendingCommand | null;
  onSend(command: ClientCommand): void;
  onClose(): void;
  error?: string | null;
};

function formatDuration(seconds: number): string {
  const minutes = Math.floor(seconds / 60);
  return `${minutes}:${String(seconds % 60).padStart(2, "0")}`;
}

function MatchSetupSection({
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
  onSetDuration(seconds: number): void;
  onSetVirusCount(count: number): void;
}) {
  return (
    <div className="ta-paper p-4">
      <div className="mb-3 flex items-center justify-between border-b-2 border-black/15 pb-2.5">
        <div>
          <p className="ta-condensed text-xs tracking-[0.18em] uppercase text-ta-ink">MATCH SETUP</p>
          <p className="ta-sans mt-0.5 text-xs leading-snug text-black/70">
            {isHost ? "Configure the round discussion timer and initial VIRUS agents." : "Set the discussion length and how many agents start as VIRUS."}
          </p>
        </div>
        {!isHost ? <span className="ta-condensed border-2 border-black/25 bg-black/10 px-2 py-0.5 text-[0.65rem] tracking-wider uppercase">HOST CONTROLLED</span> : null}
      </div>
      <div className="grid gap-3 sm:grid-cols-2">
        <SettingStepper
          label="DISCUSSION TIMER"
          value={isTimerActive ? formatDuration(currentSeconds) : "OFF"}
          hint={isTimerActive ? `Step below ${formatDuration(MIN_DISCUSSION_SECONDS)} for untimed. Max ${formatDuration(MAX_DISCUSSION_SECONDS)}.` : "Discussion runs untimed until the host advances."}
          disabled={controlsLocked}
          canDecrease={isTimerActive}
          canIncrease={!isTimerActive || currentSeconds < MAX_DISCUSSION_SECONDS}
          onDecrease={() => onSetDuration(currentSeconds - DISCUSSION_STEP_SECONDS)}
          onIncrease={() => onSetDuration(isTimerActive ? currentSeconds + DISCUSSION_STEP_SECONDS : Math.max(MIN_DISCUSSION_SECONDS, currentSeconds))}
        />
        <SettingStepper
          label="VIRUS TEAM SIZE"
          value={currentVirusCount === 0 ? "AUTO" : String(currentVirusCount)}
          hint={currentVirusCount === 0 ? "Auto-scales: 2 for 5-6 players, 3 for 7+." : `${currentVirusCount} agent${currentVirusCount === 1 ? "" : "s"} start as VIRUS.`}
          disabled={controlsLocked}
          canDecrease={currentVirusCount > MIN_VIRUS_COUNT}
          canIncrease={currentVirusCount < MAX_VIRUS_COUNT}
          onDecrease={() => onSetVirusCount(currentVirusCount - 1)}
          onIncrease={() => onSetVirusCount(currentVirusCount + 1)}
        />
      </div>
    </div>
  );
}

function SpecialRolesSection({
  enabledRoleIds,
  controlsLocked,
  onToggleRole,
  onInspectRole,
}: {
  enabledRoleIds: Set<string>;
  controlsLocked: boolean;
  onToggleRole: ToggleHandler;
  onInspectRole(role: RoleDefinition, enabled: boolean, disabled: boolean): void;
}) {
  const activeRoleCount = roles.filter((role) => enabledRoleIds.has(role.id)).length;
  return (
    <div className="ta-paper p-4">
      <div className="mb-3 flex items-end justify-between gap-3 border-b-2 border-black/15 pb-2.5">
        <div>
          <p className="ta-condensed text-xs tracking-[0.18em] uppercase text-ta-ink">SPECIAL ROLES</p>
          <p className="ta-sans mt-0.5 text-xs leading-snug text-black/70">Secret roles dealt on top of agency. Click card to toggle, <strong>i</strong> for dossier.</p>
        </div>
        <span className="ta-condensed shrink-0 whitespace-nowrap border border-black/20 bg-black/10 px-2 py-0.5 text-xs tracking-[0.12em]">{activeRoleCount} / {roles.length} IN POOL</span>
      </div>
      <div className="grid gap-2.5 sm:grid-cols-2">
        {roles.map((role) => (
          <RoleOptionCard key={role.id} role={role} enabled={enabledRoleIds.has(role.id)} disabled={controlsLocked} onToggle={onToggleRole} onInspect={onInspectRole} />
        ))}
      </div>
    </div>
  );
}

function OperationsDeckSection({
  enabledIds,
  canEdit,
  onToggleOperation,
  onInspectOperation,
}: {
  enabledIds: Set<string>;
  canEdit: boolean;
  onToggleOperation: ToggleHandler;
  onInspectOperation(operation: OperationDefinition, label: string, enabled: boolean, disabled: boolean): void;
}) {
  const hiddenAgendaEnabled = HIDDEN_AGENDA_MEMBERS.some((operation) => enabledIds.has(operation.id));
  const activeCount = DECK_OPERATIONS.filter((operation) => enabledIds.has(operation.id)).length + (hiddenAgendaEnabled ? 1 : 0);
  const deckSize = DECK_OPERATIONS.length + 1;
  const lockedOff = (operationId: string) => !canEdit || (enabledIds.has(operationId) && enabledIds.size <= 1);

  return (
    <div className="ta-paper p-4">
      <div className="mb-3 flex items-end justify-between gap-3 border-b-2 border-black/15 pb-2.5">
        <div>
          <p className="ta-condensed text-xs tracking-[0.18em] uppercase text-ta-ink">OPERATIONS DECK</p>
          <p className="ta-sans text-xs leading-snug text-black/70">Dealt from one shuffled global deck. Click card to toggle, <strong>i</strong> for tactical briefing.</p>
        </div>
        <span className="ta-condensed shrink-0 whitespace-nowrap border border-black/20 bg-black/10 px-2 py-0.5 text-xs tracking-[0.12em]">{activeCount} / {deckSize} ACTIVE</span>
      </div>
      <div className="grid gap-2.5 sm:grid-cols-2">
        {DECK_OPERATIONS.map((operation) => (
          <OperationOptionCard
            key={operation.id}
            operation={operation}
            label={packOperationIds.has(operation.id) ? "PACK 01" : operation.category.toUpperCase()}
            enabled={enabledIds.has(operation.id)}
            disabled={lockedOff(operation.id)}
            onToggle={onToggleOperation}
            onInspect={onInspectOperation}
          />
        ))}
      </div>
      <div className="mt-4 border-2 border-black/25 bg-black/[0.03] p-3.5">
        <p className="ta-condensed mb-2.5 text-[0.68rem] tracking-[0.18em] text-black/70 uppercase">HIDDEN AGENDA EXPANSION</p>
        <div className="flex flex-col items-center">
          <div className="mb-3 w-full max-w-sm">
            <OperationOptionCard operation={HIDDEN_AGENDA_COVER} label="MASTER COVER" enabled={hiddenAgendaEnabled} disabled={!canEdit} onToggle={onToggleOperation} onInspect={onInspectOperation} />
          </div>
          <div className="grid w-full gap-2.5 sm:grid-cols-2">
            {HIDDEN_AGENDA_MEMBERS.map((operation) => (
              <OperationOptionCard key={operation.id} operation={operation} label="ENVELOPE" enabled={enabledIds.has(operation.id)} disabled={lockedOff(operation.id)} onToggle={onToggleOperation} onInspect={onInspectOperation} />
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

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
  const enabledIds = new Set(projection.public.settings.enabled_operations ?? []);
  const enabledRoleIds = new Set(projection.public.settings.enabled_roles ?? []);
  const currentSeconds = projection.public.settings.discussion_seconds ?? 300;
  const currentVirusCount = projection.public.settings.virus_count ?? 0;
  const isTimerActive = projection.public.settings.discussion_timer_enabled ?? true;

  function setDuration(seconds: number): void {
    if (!canEdit) return;
    if (seconds < MIN_DISCUSSION_SECONDS) {
      onSend({ kind: "lobby.discussion_timer", discussion_timer_enabled: false, discussion_seconds: currentSeconds });
      return;
    }
    onSend({ kind: "lobby.discussion_timer", discussion_timer_enabled: true, discussion_seconds: Math.min(MAX_DISCUSSION_SECONDS, seconds) });
  }

  function setVirusCount(count: number): void {
    if (canEdit) onSend({ kind: "lobby.virus_count", virus_count: Math.min(MAX_VIRUS_COUNT, Math.max(MIN_VIRUS_COUNT, count)) });
  }

  const toggleOperation: ToggleHandler = (operationId, enabled) => {
    if (canEdit) onSend({ kind: "lobby.operation_enabled", operation_kind: operationId, operation_enabled: enabled });
  };
  const toggleRole: ToggleHandler = (roleId, enabled) => {
    if (canEdit) onSend({ kind: "lobby.role_enabled", role_id: roleId, role_enabled: enabled });
  };

  return (
    <div className="space-y-4">
      {showHeader ? <PaperTitle>Room settings</PaperTitle> : null}
      <MatchSetupSection isTimerActive={isTimerActive} currentSeconds={currentSeconds} currentVirusCount={currentVirusCount} controlsLocked={controlsLocked} isHost={isHost} onSetDuration={setDuration} onSetVirusCount={setVirusCount} />
      <SpecialRolesSection
        enabledRoleIds={enabledRoleIds}
        controlsLocked={controlsLocked}
        onToggleRole={toggleRole}
        onInspectRole={(role, enabled, disabled) => setInspectedItem({ type: "role", role, enabled, disabled })}
      />
      <OperationsDeckSection
        enabledIds={enabledIds}
        canEdit={canEdit}
        onToggleOperation={toggleOperation}
        onInspectOperation={(operation, label, enabled, disabled) => setInspectedItem({ type: "operation", operation, label, enabled, disabled })}
      />
      {error ? <p className="ta-paper ta-sans border-l-4 border-ta-red px-3 py-2 text-sm leading-snug text-ta-red" role="alert">{error}</p> : null}
      <DossierDialog
        item={inspectedItem}
        onClose={() => setInspectedItem(null)}
        canToggle={canEdit}
        onToggle={(id, enabled) => {
          if (inspectedItem?.type === "operation") toggleOperation(id, enabled);
          else if (inspectedItem?.type === "role") toggleRole(id, enabled);
        }}
      />
    </div>
  );
}

export function SettingsScreen({ projection, pending, onSend, onClose, error = null }: SettingsScreenProps) {
  return (
    <div className="ta-rise ta-screen ta-screen--wide ta-screen--settings space-y-4">
      <SettingsPanel projection={projection} pending={pending} onSend={onSend} showHeader error={error} />
      <InkButton className="w-full" onClick={onClose}>Back to game</InkButton>
    </div>
  );
}
