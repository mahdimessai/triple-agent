import type { MouseEvent } from "react";
import type { OperationDefinition } from "../../operations";
import type { RoleDefinition } from "../../roles";
import { ArtStamp } from "../../ui";
import type { ToggleHandler } from "./types";

export function OperationOptionCard({
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
  onInspect(operation: OperationDefinition, label: string, enabled: boolean, disabled: boolean): void;
}) {
  function inspect(event: MouseEvent): void {
    event.stopPropagation();
    onInspect(operation, label, enabled, disabled);
  }

  return (
    <div className="ta-compact-card group" data-enabled={enabled} data-disabled={disabled ? "true" : undefined}>
      <button
        type="button"
        className="flex min-w-0 flex-1 items-center gap-2.5 py-0.5 text-left"
        disabled={disabled}
        onClick={() => onToggle(operation.id, !enabled)}
        aria-pressed={enabled}
        aria-label={`${enabled ? "Disable" : "Enable"} ${operation.name}`}
      >
        <span className="ta-compact-card-art shrink-0" aria-hidden="true">
          <ArtStamp artName={operation.artName} alt="" className="h-6 w-7 object-contain" />
        </span>
        <span className="min-w-0 flex-1 pr-1">
          <span className="ta-narrow block text-sm leading-tight text-ta-ink uppercase sm:text-base">{operation.name}</span>
          <span className="ta-narrow block text-[0.62rem] tracking-[0.14em] text-black/55 uppercase">{label}</span>
        </span>
      </button>
      <div className="flex shrink-0 items-center gap-1.5 pl-1">
        <button type="button" className="ta-compact-info-btn" aria-label={`Inspect ${operation.name} briefing`} title="View tactical briefing" onClick={inspect}>i</button>
        <button
          type="button"
          disabled={disabled}
          onClick={() => onToggle(operation.id, !enabled)}
          className={`ta-compact-status-badge ${enabled ? "ta-badge-enabled" : "ta-badge-disabled"}`}
          aria-label={`Toggle ${operation.name}`}
        >
          {enabled ? "ON" : "OFF"}
        </button>
      </div>
    </div>
  );
}

export function RoleOptionCard({
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
  onInspect(role: RoleDefinition, enabled: boolean, disabled: boolean): void;
}) {
  function inspect(event: MouseEvent): void {
    event.stopPropagation();
    onInspect(role, enabled, disabled);
  }

  return (
    <div className="ta-compact-card group" data-enabled={enabled} data-disabled={disabled ? "true" : undefined}>
      <button
        type="button"
        className="flex min-w-0 flex-1 items-center gap-2.5 py-0.5 text-left"
        disabled={disabled}
        onClick={() => onToggle(role.id, !enabled)}
        aria-pressed={enabled}
        aria-label={`${enabled ? "Remove" : "Add"} ${role.name}`}
      >
        <span className="ta-compact-card-art shrink-0" aria-hidden="true">
          <ArtStamp artName={role.artName} alt="" className="h-6 w-7 object-contain" />
        </span>
        <span className="min-w-0 flex-1 pr-1">
          <span className="ta-narrow block text-sm leading-tight text-ta-ink uppercase sm:text-base">{role.name}</span>
          <span className={`ta-narrow block text-[0.62rem] tracking-[0.14em] uppercase ${role.faction === "VIRUS" ? "text-ta-red" : "text-[#1d5b79]"}`}>{role.faction}</span>
        </span>
      </button>
      <div className="flex shrink-0 items-center gap-1.5 pl-1">
        <button type="button" className="ta-compact-info-btn" aria-label={`Inspect ${role.name} dossier`} title="View role dossier" onClick={inspect}>i</button>
        <button
          type="button"
          disabled={disabled}
          onClick={() => onToggle(role.id, !enabled)}
          className={`ta-compact-status-badge ${enabled ? "ta-badge-enabled" : "ta-badge-disabled"}`}
          aria-label={`Toggle ${role.name}`}
        >
          {enabled ? "ON" : "OFF"}
        </button>
      </div>
    </div>
  );
}
