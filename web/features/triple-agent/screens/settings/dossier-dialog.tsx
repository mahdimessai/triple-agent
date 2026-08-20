"use client";

import { useEffect, useRef, useSyncExternalStore } from "react";
import { createPortal } from "react-dom";
import { operationBrief } from "../../operations";
import { ArtStamp } from "../../ui";
import type { InspectedItem } from "./types";

const emptySubscribe = () => () => {};

function useIsMounted(): boolean {
  return useSyncExternalStore(emptySubscribe, () => true, () => false);
}

export function DossierDialog({
  item,
  onClose,
  canToggle = false,
  onToggle,
}: {
  item: InspectedItem;
  onClose(): void;
  canToggle?: boolean;
  onToggle?(id: string, enabled: boolean): void;
}) {
  const mounted = useIsMounted();
  const closeButtonRef = useRef<HTMLButtonElement | null>(null);

  useEffect(() => {
    if (!item) return;
    const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    closeButtonRef.current?.focus();
    function handleKeyDown(event: KeyboardEvent): void {
      if (event.key === "Escape") onClose();
    }
    window.addEventListener("keydown", handleKeyDown);
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
      previousFocus?.focus();
    };
  }, [item, onClose]);

  if (!item || !mounted) return null;

  const isOperation = item.type === "operation";
  const title = isOperation ? item.operation.name : item.role.name;
  const artName = isOperation ? item.operation.artName : item.role.artName;
  const tag = isOperation ? item.label : item.role.faction;
  const tagColor = !isOperation && item.role.faction === "VIRUS" ? "text-ta-red" : "text-[#1d5b79]";
  const description = isOperation ? operationBrief(item.operation) : item.role.effect;
  const id = isOperation ? item.operation.id : item.role.id;

  return createPortal(
    <div className="ta-modal-portal-backdrop" role="presentation" onClick={onClose}>
      <div
        className="ta-paper relative w-full max-w-md border-4 border-black p-5 text-left shadow-[8px_8px_0_var(--ta-shadow)]"
        role="dialog"
        aria-modal="true"
        aria-labelledby="dossier-dialog-title"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="flex items-start justify-between gap-3 border-b-2 border-black/25 pb-3.5">
          <div className="flex items-center gap-3">
            <div className="flex h-16 w-20 shrink-0 items-center justify-center border-2 border-black bg-white p-1 shadow-[2px_2px_0_var(--ta-shadow)]">
              <ArtStamp artName={artName} alt="" className="h-full w-full object-contain" />
            </div>
            <div>
              <p className={`ta-condensed text-xs tracking-[0.16em] uppercase ${tagColor}`}>{tag}</p>
              <h3 id="dossier-dialog-title" className="ta-display text-2xl leading-none text-ta-ink">{title}</h3>
            </div>
          </div>
          <button ref={closeButtonRef} type="button" className="ta-secondary-button !min-h-0 border-2 border-black px-2.5 py-1 text-xs uppercase tracking-wider" onClick={onClose} aria-label="Close dossier">✕</button>
        </div>

        <div className="my-4 space-y-3">
          <div>
            <p className="ta-condensed mb-0.5 text-[0.65rem] tracking-[0.16em] text-black/60 uppercase">{isOperation ? "TACTICAL BRIEFING" : "ROLE PROFILE"}</p>
            <p className="ta-sans text-sm leading-relaxed text-ta-ink">{description}</p>
          </div>
          {isOperation ? (
            <div>
              <p className="ta-condensed mb-0.5 text-[0.65rem] tracking-[0.16em] text-black/60 uppercase">INPUT MECHANISM</p>
              <p className="ta-sans text-xs leading-relaxed text-black/80">{item.operation.input}</p>
            </div>
          ) : (
            <div>
              <p className="ta-condensed mb-0.5 text-[0.65rem] tracking-[0.16em] text-black/60 uppercase">BACKGROUND</p>
              <p className="ta-sans text-xs leading-relaxed text-black/80">{item.role.description}</p>
            </div>
          )}
        </div>

        <div className="flex items-center justify-between border-t-2 border-black/25 pt-3.5">
          <span className="ta-sans text-xs text-black/60">Status: <strong className="text-ta-ink">{item.enabled ? "ACTIVE" : "EXCLUDED"}</strong></span>
          {canToggle && !item.disabled ? (
            <button type="button" className={`ta-secondary-button !min-h-0 border-2 border-black px-4 py-1 text-xs uppercase tracking-wider ${item.enabled ? "bg-ta-red text-ta-paper" : "bg-ta-teal text-ta-ink"}`} onClick={() => onToggle?.(id, !item.enabled)}>
              {item.enabled ? "Disable" : "Enable"}
            </button>
          ) : (
            <button type="button" className="ta-secondary-button !min-h-0 border-2 border-black px-4 py-1 text-xs uppercase tracking-wider" onClick={onClose}>Close</button>
          )}
        </div>
      </div>
    </div>,
    document.body,
  );
}
