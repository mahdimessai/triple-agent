"use client";

import { useState } from "react";
import type { ClientCommand, RoomProjection } from "../protocol";
import type { PendingCommand } from "../use-room";
import { getOperation, operationIdForServerKind, operationResultText, roomBriefing } from "../operations";
import { ArtStamp, InkButton, PaperTitle } from "../ui";

export type OperationScreenProps = {
  projection: RoomProjection;
  pending: PendingCommand | null;
  onSend(command: ClientCommand): void;
};

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

function OperationResultView({ projection }: { projection: RoomProjection }) {
  const result = projection.private.operation_result;
  if (!result) {
    return projection.private.operation_instruction ? (
      <div className="ta-operation-state"><p className="ta-sans text-lg">{projection.private.operation_instruction}</p></div>
    ) : null;
  }
  return (
    <div className="ta-operation-state ta-operation-state-choice">
      <div>
        <p className="ta-condensed text-xs tracking-[0.16em] text-black/60">FOR YOUR EYES ONLY</p>
        <p className="ta-sans mt-1 text-lg"><HighlightFaction text={operationResultText(result, projection)} /></p>
      </div>
    </div>
  );
}

export function OperationScreen({ projection, pending, onSend }: OperationScreenProps) {
  const [selectedTargets, setSelectedTargets] = useState<string[]>([]);
  const [selectedChoice, setSelectedChoice] = useState("");
  const room = projection.public;
  const personal = projection.private;
  const operationKey = `${room.operation?.kind ?? "none"}:${room.operation?.step ?? 0}:${personal.operation_kind ?? ""}`;

  const [prevOperationKey, setPrevOperationKey] = useState(operationKey);
  if (prevOperationKey !== operationKey) {
    setPrevOperationKey(operationKey);
    setSelectedTargets([]);
    setSelectedChoice("");
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

      {secretOperation ? (
        <div className="ta-operation-state ta-operation-state-choice">
          <div>
            <p className="ta-condensed text-xs tracking-[0.16em] text-black/60">YOUR ORDERS · FOR YOUR EYES ONLY</p>
            <p className="ta-display mt-1 text-3xl">{personal.operation_name ?? secretOperation.name}</p>
            <p className="ta-sans mt-2 text-base leading-snug">{personal.operation_instruction ?? secretOperation.privatePrompt}</p>
          </div>
          <ArtStamp artName={secretOperation.artName} alt="" className="h-16 w-auto shrink-0 object-contain" />
        </div>
      ) : null}

      {room.phase === "OPERATION_RESULT" ? <OperationResultView projection={projection} /> : !isInputOwner ? (
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
        {room.phase === "OPERATION_INPUT" ? (personal.can_submit ? "Confirm operation" : `Waiting for ${activePlayerName}`) : personal.can_submit ? "Done" : `Waiting for ${activePlayerName}`}
      </InkButton>
    </div>
  );
}
