"use client";

import { useEffect, useEffectEvent, useRef, useState, useSyncExternalStore } from "react";
import { createPortal } from "react-dom";
import type { ClientCommand, RoomProjection } from "../protocol";
import type { PendingCommand } from "../use-room";
import { getOperation, operationIdForServerKind, operationResultText, roomBriefing } from "../operations";
import { ArtStamp, InkButton, PaperTitle } from "../ui";

export type OperationScreenProps = {
  projection: RoomProjection;
  pending: PendingCommand | null;
  onSend(command: ClientCommand): void;
};

const emptySubscribe = () => () => {};

function useIsMounted(): boolean {
  return useSyncExternalStore(emptySubscribe, () => true, () => false);
}

function HighlightFaction({ text }: { text: string }) {
  const parts = text.split(/(VIRUS|SERVICE)/g);
  return (
    <>{parts.map((part, index) => {
      if (part === "VIRUS") return <span key={index} className="font-bold text-ta-red">VIRUS</span>;
      if (part === "SERVICE") return <span key={index} className="font-bold text-[#1d5b79]">SERVICE</span>;
      return part;
    })}</>
  );
}

export function ClassifiedIntelDialog({
  isOpen,
  onClose,
  projection,
}: {
  isOpen: boolean;
  onClose(): void;
  projection: RoomProjection;
}) {
  const mounted = useIsMounted();
  const closeButtonRef = useRef<HTMLButtonElement | null>(null);
  const closeFromKeyboard = useEffectEvent(onClose);

  useEffect(() => {
    if (!isOpen) return;
    const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    closeButtonRef.current?.focus();
    function handleKeyDown(event: KeyboardEvent): void {
      if (event.key === "Escape") closeFromKeyboard();
    }
    window.addEventListener("keydown", handleKeyDown);
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
      previousFocus?.focus();
    };
  }, [isOpen]);

  if (!isOpen) return null;

  const result = projection.private.operation_result;
  if (!result) return null;

  const resultText = operationResultText(result, projection);
  const secretOperation = projection.private.operation_kind
    ? getOperation(operationIdForServerKind(projection.private.operation_kind))
    : null;

  const content = (
    <div className="ta-modal-portal-backdrop" role="presentation" onClick={onClose}>
      <div
        className="ta-paper relative w-full max-w-md border-4 border-black p-5 text-left shadow-[8px_8px_0_var(--ta-shadow)]"
        role="dialog"
        aria-modal="true"
        aria-labelledby="classified-intel-title"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="flex items-start justify-between gap-3 border-b-2 border-black/25 pb-3">
          <div className="flex items-center gap-2.5">
            <span className="text-2xl" aria-hidden="true">📂</span>
            <div>
              <p className="ta-condensed text-xs font-bold tracking-[0.2em] text-ta-red uppercase">
                FOR YOUR EYES ONLY
              </p>
              <h3 id="classified-intel-title" className="ta-display text-2xl leading-none text-ta-ink">
                CLASSIFIED INTEL
              </h3>
            </div>
          </div>
          <button
            ref={closeButtonRef}
            type="button"
            className="ta-secondary-button !min-h-0 border-2 border-black px-2.5 py-1 text-xs uppercase tracking-wider"
            onClick={onClose}
            aria-label="Close classified intel"
          >
            ✕
          </button>
        </div>

        <div className="my-4 space-y-3">
          {secretOperation ? (
            <div className="flex items-center gap-3 border-b border-black/15 pb-2">
              <ArtStamp artName={secretOperation.artName} alt="" className="h-12 w-auto shrink-0 object-contain" />
              <div>
                <p className="ta-condensed text-[0.65rem] tracking-[0.16em] text-black/60 uppercase">OPERATION ORDERS</p>
                <p className="ta-display text-lg leading-tight">{projection.private.operation_name ?? secretOperation.name}</p>
              </div>
            </div>
          ) : null}

          <div>
            <p className="ta-condensed mb-1 text-[0.65rem] tracking-[0.16em] text-black/60 uppercase">
              DECRYPTED INTELLIGENCE
            </p>
            <div className="border-l-4 border-ta-teal bg-black/5 p-3">
              <p className="ta-sans text-lg leading-snug font-medium text-ta-ink">
                <HighlightFaction text={resultText} />
              </p>
            </div>
          </div>

          {result.message && result.message !== resultText ? (
            <div>
              <p className="ta-condensed mb-0.5 text-[0.65rem] tracking-[0.16em] text-black/60 uppercase">TRANSMISSION NOTE</p>
              <p className="ta-sans text-xs text-black/75">{result.message}</p>
            </div>
          ) : null}

          {projection.private.operation_instruction ? (
            <div>
              <p className="ta-condensed mb-0.5 text-[0.65rem] tracking-[0.16em] text-black/60 uppercase">DIRECTIVE</p>
              <p className="ta-sans text-xs text-black/80">{projection.private.operation_instruction}</p>
            </div>
          ) : null}
        </div>

        <div className="border-t-2 border-black/25 pt-3.5 flex justify-end">
          <button
            type="button"
            className="ta-secondary-button !min-h-0 border-2 border-black bg-ta-paper px-4 py-2 text-xs font-bold uppercase tracking-wider shadow-[2px_2px_0_var(--ta-shadow)] hover:bg-[#fff8e8]"
            onClick={onClose}
          >
            🔒 HIDE INTEL
          </button>
        </div>
      </div>
    </div>
  );

  if (typeof document !== "undefined" && mounted) {
    return createPortal(content, document.body);
  }
  return content;
}

function TargetPicker({ projection, targetCount, selected, onChange }: {
  projection: RoomProjection;
  targetCount: number;
  selected: string[];
  onChange(value: string[]): void;
}) {
  const legal = new Set(projection.private.legal_target_ids ?? []);
  const targets = projection.public.players.filter((player) => legal.has(player.id));
  return (
    <div className="ta-operation-state ta-operation-state-choice">
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="ta-condensed text-xs tracking-[0.16em]">YOUR MOVE</p>
          <p className="ta-sans mt-1 text-lg">{projection.private.operation_instruction ?? `Choose ${targetCount === 2 ? "two other players" : "one other player"}.`}</p>
        </div>
        <span className="ta-condensed shrink-0 text-xs tracking-[0.12em]">{selected.length} / {targetCount}</span>
      </div>
      <div className="grid grid-cols-2 gap-2">
        {targets.map((player) => {
          const chosen = selected.includes(player.id);
          return (
            <button
              className="ta-target-button"
              data-selected={chosen}
              key={player.id}
              onClick={() => onChange(chosen ? selected.filter((id) => id !== player.id) : selected.length < targetCount ? [...selected, player.id] : selected)}
              type="button"
              aria-pressed={chosen}
            >
              {player.name}
            </button>
          );
        })}
      </div>
    </div>
  );
}

function ChoicePicker({ projection, selected, onChange }: { projection: RoomProjection; selected: string; onChange(value: string): void }) {
  const choices = projection.private.choices ?? ["STAY", "DEFECT"];
  return (
    <div className="ta-operation-state ta-operation-state-choice">
      <div>
        <p className="ta-condensed text-xs tracking-[0.16em]">YOUR MOVE</p>
        <p className="ta-sans mt-1 text-lg">{projection.private.operation_instruction ?? "Choose one option."}</p>
      </div>
      <div className="grid grid-cols-2 gap-2">
        {choices.map((choice) => (
          <button className="ta-target-button" data-selected={selected === choice} key={choice} onClick={() => onChange(choice)} type="button" aria-pressed={selected === choice}>
            {choice.replaceAll("_", " ")}
          </button>
        ))}
      </div>
    </div>
  );
}

export function OperationScreen({ projection, pending, onSend }: OperationScreenProps) {
  const [selectedTargets, setSelectedTargets] = useState<string[]>([]);
  const [selectedChoice, setSelectedChoice] = useState("");
  const [isIntelOpen, setIsIntelOpen] = useState(false);
  const room = projection.public;
  const personal = projection.private;
  const operationKey = `${room.operation?.kind ?? "none"}:${room.operation?.step ?? 0}:${personal.operation_kind ?? ""}`;

  const [prevOperationKey, setPrevOperationKey] = useState(operationKey);
  if (prevOperationKey !== operationKey) {
    setPrevOperationKey(operationKey);
    setSelectedTargets([]);
    setSelectedChoice("");
    setIsIntelOpen(false);
  }

  const publicOperation = getOperation(operationIdForServerKind(room.operation?.kind));
  const secretOperation = personal.operation_kind && personal.operation_kind !== room.operation?.kind
    ? getOperation(operationIdForServerKind(personal.operation_kind))
    : null;
  const activePlayerName = room.operation?.active_player_name
    ?? room.players.find((player) => player.id === room.active_player_id)?.name
    ?? "The active player";
  const isActivePlayer = (room.operation?.active_player_id ?? room.active_player_id) === personal.player_id;
  const isInputOwner = (room.operation?.input_owner_id ?? room.active_player_id) === personal.player_id;
  const publicInstruction = room.operation?.public_instruction || publicOperation.publicUpdate;
  const targetCount = room.operation?.target_count ?? (room.operation?.input_kind === "TWO_TARGETS" ? 2 : room.operation?.input_kind === "ONE_TARGET" ? 1 : 0);
  const requiresChoice = room.operation?.input_kind === "CHOICE" || Boolean(personal.choices?.length);
  const requiresTargets = !requiresChoice && targetCount > 0;
  const canSubmitForm = requiresChoice ? Boolean(selectedChoice) : requiresTargets ? selectedTargets.length === targetCount : true;
  const inputBusy = pending?.kind === "operation.resolve";
  const doneBusy = pending?.kind === "operation.explain_done";

  function submit(): void {
    if (room.phase === "OPERATION_RESULT") {
      onSend({ kind: "operation.explain_done" });
      return;
    }
    onSend({
      kind: "operation.resolve",
      ...(selectedTargets.length ? { target_ids: selectedTargets } : {}),
      ...(selectedChoice ? { choice: selectedChoice } : {}),
    });
  }

  return (
    <div className="ta-rise ta-screen ta-screen--operation">
      <PaperTitle>{publicOperation.name}</PaperTitle>
      <div className="ta-paper ta-operation-brief overflow-hidden p-5 text-center">
        <div className="ta-operation-brief-art"><ArtStamp artName={publicOperation.artName} alt={`${publicOperation.name} illustration`} className="mx-auto h-40 w-auto object-contain" /></div>
        <div className="ta-operation-brief-copy">
          <p className="ta-condensed text-xs tracking-[0.2em] text-black/60">{isActivePlayer ? "YOUR OPERATION" : `${activePlayerName.toUpperCase()}'S OPERATION`}</p>
          <h3 className="ta-display mt-2 text-4xl">{publicOperation.name}</h3>
          <p className="ta-condensed mt-4 text-xs tracking-[0.18em] text-black/60">READ OUT LOUD BEFORE CONTINUING IN SECRET</p>
          <p className="ta-sans mx-auto mt-2 max-w-sm text-base leading-snug">{roomBriefing(publicOperation, activePlayerName, isActivePlayer, publicInstruction)}</p>
        </div>
      </div>

      {secretOperation && room.phase !== "OPERATION_RESULT" ? (
        <div className="ta-operation-state ta-operation-state-choice">
          <div>
            <p className="ta-condensed text-xs tracking-[0.16em] text-black/60">YOUR ORDERS · FOR YOUR EYES ONLY</p>
            <p className="ta-display mt-1 text-3xl">{personal.operation_name ?? secretOperation.name}</p>
            <p className="ta-sans mt-2 text-base leading-snug">{personal.operation_instruction ?? secretOperation.privatePrompt}</p>
          </div>
          <ArtStamp artName={secretOperation.artName} alt="" className="h-16 w-auto shrink-0 object-contain" />
        </div>
      ) : null}

      {room.phase === "OPERATION_RESULT" ? (
        personal.operation_result ? (
          <div className="ta-paper p-4 text-center shadow-[4px_4px_0_var(--ta-shadow)]">
            <div className="flex items-center justify-center gap-2">
              <span className="text-xl" aria-hidden="true">📂</span>
              <p className="ta-condensed text-xs font-bold tracking-[0.18em] text-ta-red uppercase">
                CLASSIFIED INTEL READY
              </p>
            </div>
            <p className="ta-sans mt-1 text-sm text-black/75">
              Private intelligence has been decoded for your eyes only.
            </p>
            <button
              type="button"
              className="ta-secondary-button mt-3 w-full border-2 border-black bg-ta-paper py-2.5 font-bold tracking-wider text-ta-ink uppercase shadow-[3px_3px_0_var(--ta-shadow)] hover:bg-[#fff8e8]"
              onClick={() => setIsIntelOpen(true)}
            >
              📂 VIEW CLASSIFIED INTEL
            </button>
          </div>
        ) : personal.operation_instruction ? (
          <div className="ta-operation-state">
            <div>
              <p className="ta-condensed text-xs tracking-[0.16em]">OPERATION RESULT</p>
              <p className="ta-sans mt-1 text-lg">{personal.operation_instruction}</p>
            </div>
          </div>
        ) : null
      ) : !isInputOwner ? (
        personal.operation_instruction ? (
          <div className="ta-operation-state"><div><p className="ta-condensed text-xs tracking-[0.16em]">OPERATION IN PROGRESS</p><p className="ta-sans mt-1 text-lg">{personal.operation_instruction}</p></div></div>
        ) : null
      ) : requiresChoice ? (
        <ChoicePicker projection={projection} selected={selectedChoice} onChange={setSelectedChoice} />
      ) : requiresTargets ? (
        <TargetPicker projection={projection} targetCount={targetCount} selected={selectedTargets} onChange={setSelectedTargets} />
      ) : (
        <div className="ta-operation-state"><div><p className="ta-condensed text-xs tracking-[0.16em]">YOUR MOVE</p><p className="ta-sans mt-1 text-lg">{personal.operation_instruction ?? "Review your briefing and confirm."}</p></div></div>
      )}

      <InkButton
        variant="orange"
        className="ta-operation-submit w-full"
        onClick={submit}
        disabled={!personal.can_submit || (room.phase === "OPERATION_INPUT" && !canSubmitForm) || Boolean(pending && !inputBusy && !doneBusy)}
        busy={inputBusy || doneBusy}
        busyLabel={room.phase === "OPERATION_INPUT" ? "Saving operation…" : "Saving…"}
      >
        {room.phase === "OPERATION_INPUT"
          ? (personal.can_submit ? "Confirm operation" : `Waiting for ${activePlayerName}`)
          : personal.can_submit
          ? "Done"
          : personal.operation_acknowledged
          ? "Waiting for other agent…"
          : `Waiting for ${activePlayerName}`}
      </InkButton>

      {/* Reopenable Classified Intel Modal Dialog */}
      <ClassifiedIntelDialog
        isOpen={isIntelOpen}
        onClose={() => setIsIntelOpen(false)}
        projection={projection}
      />
    </div>
  );
}
