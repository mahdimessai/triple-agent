import { useState } from "react";
import { ArtStamp } from "@/components/ui/art-stamp";
import { InkButton } from "@/components/ui/ink-button";
import { PaperTitle } from "@/components/ui/paper-title";
import { getOperation, type OperationDefinition, type OperationId } from "@/components/triple-agent/operation-catalog";
import type { RoomProjection } from "@/components/triple-agent/server-client";
import { operationIdForServerKind } from "@/features/triple-agent/model/screen";
import { operationResultText, roomBriefing } from "@/features/triple-agent/operations/presentation";

const operationTargets = ["PLAYER A", "PLAYER B", "PLAYER D", "PLAYER E"];

function OperationInput({ operation }: { operation: OperationDefinition }) {
  const [selectedTargets, setSelectedTargets] = useState<string[]>([]);
  const [defectChoice, setDefectChoice] = useState<"defect" | "stay">("stay");

  if (operation.status === "disabled-by-config") return <div className="ta-operation-state ta-operation-state-muted"><div><p className="ta-condensed text-xs tracking-[0.16em]">SERVER CONFIGURATION</p><p className="ta-condensed mt-1 text-lg">This operation is disabled in the current room.</p></div><span className="ta-condensed text-xs tracking-[0.12em]">CONFIG OFF</span></div>;
  if (operation.status === "recovered-only") return <div className="ta-operation-state"><div><p className="ta-condensed text-xs tracking-[0.16em]">RECOVERED VARIANT</p><p className="ta-condensed mt-1 text-lg">The server supplies this variant&apos;s legal input and result.</p></div><span className="ta-condensed text-xs tracking-[0.12em]">SERVER-DEFINED</span></div>;
  if (operation.category === "choice") {
    if (operation.input === "Defect or stay") return <div className="ta-operation-state ta-operation-state-choice"><div><p className="ta-condensed text-xs tracking-[0.16em]">PLAYER C INPUT</p><p className="ta-condensed mt-1 text-lg">Choose whether to defect from your current agency.</p></div><div className="grid grid-cols-2 gap-2">{["stay", "defect"].map((choice) => <button className="ta-target-button" data-selected={defectChoice === choice} key={choice} onClick={() => setDefectChoice(choice as "defect" | "stay")} type="button" aria-pressed={defectChoice === choice}>{choice}</button>)}</div></div>;
    const targetLimit = operation.input.includes("two") ? 2 : 1;
    return <div className="ta-operation-state ta-operation-state-choice"><div className="flex items-start justify-between gap-3"><div><p className="ta-condensed text-xs tracking-[0.16em]">PLAYER C INPUT</p><p className="ta-condensed mt-1 text-lg">Select {targetLimit === 2 ? "two players" : "one player"} for the server.</p></div><span className="ta-condensed shrink-0 text-xs tracking-[0.12em]">{selectedTargets.length} / {targetLimit}</span></div><div className="grid grid-cols-2 gap-2">{operationTargets.map((target) => { const selected = selectedTargets.includes(target); return <button className="ta-target-button" data-selected={selected} key={target} onClick={() => setSelectedTargets((current) => selected ? current.filter((item) => item !== target) : current.length < targetLimit ? [...current, target] : current)} type="button" aria-pressed={selected}>{target}</button>; })}</div></div>;
  }
  const stateLabel = operation.category === "information" ? "PRIVATE RESULT READY" : operation.category === "hidden" ? "PRIVATE OBJECTIVE DELIVERED" : "PUBLIC MODIFIER AWAITING TARGET";
  return <div className="ta-operation-state"><div><p className="ta-condensed text-xs tracking-[0.16em]">SERVER STATE</p><p className="ta-condensed mt-1 text-lg">{stateLabel}</p></div><span className="ta-condensed text-xs tracking-[0.12em]">NO PLAYER INPUT</span></div>;
}

function HighlightFaction({ text }: { text: string }) {
  const parts = text.split(/(VIRUS|SERVICE)/g);
  return (
    <>
      {parts.map((part, index) => {
        if (part === "VIRUS") return <span key={index} className="text-ta-red font-bold">VIRUS</span>;
        if (part === "SERVICE") return <span key={index} className="text-[#1d5b79] font-bold">SERVICE</span>;
        return part;
      })}
    </>
  );
}

function LiveOperationInput({ projection, selectedTargets, setSelectedTargets, selectedChoice, setSelectedChoice }: { projection: RoomProjection; selectedTargets: string[]; setSelectedTargets: (value: string[]) => void; selectedChoice: string; setSelectedChoice: (value: string) => void }) {
  const room = projection.public;
  const personal = projection.private;
  const isInputOwner = (room.operation?.input_owner_id ?? room.active_player_id) === personal.player_id;
  const operationResult = personal.operation_result;
  const targetCount = room.operation?.target_count ?? (personal.legal_target_ids?.length ? 1 : 0);

  // Only the recipient has anything to read here; everyone else already has the
  // public briefing above and does not need a running commentary.
  if (room.phase === "OPERATION_RESULT") {
    return operationResult ? (
      <div className="ta-operation-state ta-operation-state-choice">
        <div>
          <p className="ta-condensed text-xs tracking-[0.16em] text-black/60">FOR YOUR EYES ONLY</p>
          <p className="ta-condensed mt-1 text-lg">
            <HighlightFaction text={operationResultText(operationResult, projection)} />
          </p>
        </div>
      </div>
    ) : null;
  }

  if (!isInputOwner) {
    if (personal.operation_instruction) {
      return (
        <div className="ta-operation-state">
          <div>
            <p className="ta-condensed text-xs tracking-[0.16em]">OPERATION IN PROGRESS</p>
            <p className="ta-condensed mt-1 text-lg">{personal.operation_instruction}</p>
          </div>
        </div>
      );
    }
    return null;
  }

  if ((personal.choices && personal.choices.length > 0) || room.operation?.input_kind === "CHOICE") {
    const choices = personal.choices ?? ["STAY", "DEFECT"];
    return (
      <div className="ta-operation-state ta-operation-state-choice">
        <div>
          <p className="ta-condensed text-xs tracking-[0.16em]">YOUR MOVE</p>
          <p className="ta-condensed mt-1 text-lg">{personal.operation_instruction ?? "Choose one option."}</p>
        </div>
        <div className="grid grid-cols-2 gap-2">
          {choices.map((choice) => (
            <button
              className="ta-target-button"
              data-selected={selectedChoice === choice}
              key={choice}
              onClick={() => setSelectedChoice(choice)}
              type="button"
              aria-pressed={selectedChoice === choice}
            >
              {choice.replace(/_/g, " ")}
            </button>
          ))}
        </div>
      </div>
    );
  }

  if (targetCount > 0 && personal.legal_target_ids && personal.legal_target_ids.length > 0) {
    return (
      <div className="ta-operation-state ta-operation-state-choice">
        <div className="flex items-start justify-between gap-3">
          <div>
            <p className="ta-condensed text-xs tracking-[0.16em]">YOUR MOVE</p>
            <p className="ta-condensed mt-1 text-lg">
              {personal.operation_instruction ?? `Choose ${targetCount === 2 ? "two other players" : "one other player"} for ${room.operation?.name ?? "this operation"}.`}
            </p>
          </div>
          <span className="ta-condensed shrink-0 text-xs tracking-[0.12em]">
            {selectedTargets.length} / {targetCount}
          </span>
        </div>
        <div className="grid grid-cols-2 gap-2">
          {room.players
            .filter((player) => (personal.legal_target_ids ?? []).includes(player.id))
            .map((player) => (
              <button
                className="ta-target-button"
                data-selected={selectedTargets.includes(player.id)}
                key={player.id}
                onClick={() =>
                  setSelectedTargets(
                    selectedTargets.includes(player.id)
                      ? selectedTargets.filter((id) => id !== player.id)
                      : selectedTargets.length < targetCount
                        ? [...selectedTargets, player.id]
                        : selectedTargets
                  )
                }
                type="button"
                aria-pressed={selectedTargets.includes(player.id)}
              >
                {player.name}
              </button>
            ))}
        </div>
      </div>
    );
  }

  return (
    <div className="ta-operation-state">
      <div>
        <p className="ta-condensed text-xs tracking-[0.16em]">YOUR MOVE</p>
        <p className="ta-condensed mt-1 text-lg">{personal.operation_instruction ?? "Review your briefing and confirm."}</p>
      </div>
    </div>
  );
}

export function OperationScreen({ operationId, projection, onNext }: { operationId: OperationId; projection?: RoomProjection; onNext: (targetIds?: string[], choice?: string) => void }) {
  const [selectedTargets, setSelectedTargets] = useState<string[]>([]);
  const [selectedChoice, setSelectedChoice] = useState("");
  // The public projection already carries the Hidden Agenda cover for hidden
  // operations, so this is the identity the room hears and the recipient reads
  // out loud. The real orders arrive separately in the private projection.
  const serverOperationId = operationIdForServerKind(projection?.public.operation?.kind);
  const operation = getOperation(projection?.public.operation ? serverOperationId : operationId);
  const secretOrders = projection?.private.operation_name && projection.private.operation_kind !== projection.public.operation?.kind
    ? getOperation(operationIdForServerKind(projection.private.operation_kind))
    : undefined;
  const activePlayerName = projection?.public.players.find((player) => player.id === projection.public.active_player_id)?.name;
  const isActivePlayer = projection ? projection.public.active_player_id === projection.private.player_id : false;
  const publicInstruction = projection?.public.operation?.public_instruction || operation.publicUpdate;
  const recipient = activePlayerName ?? "The active player";
  const waitingLabel = `Waiting for ${recipient}`;

  const targetCount = projection?.public.operation?.target_count ?? (projection?.private.legal_target_ids?.length ? 1 : 0);
  const requiresChoice = Boolean(projection?.private.choices && projection.private.choices.length > 0) || projection?.public.operation?.input_kind === "CHOICE";
  const requiresTargets = !requiresChoice && targetCount > 0 && Boolean(projection?.private.legal_target_ids?.length);
  const canSubmitForm = !requiresChoice && !requiresTargets ? true : requiresChoice ? Boolean(selectedChoice) : selectedTargets.length === targetCount;

  return (
    <div className="ta-rise ta-screen ta-screen--operation">
      <PaperTitle>{operation.name}</PaperTitle>
      <div className="ta-paper ta-operation-brief overflow-hidden p-5 text-center">
        <div className="ta-operation-brief-art">
          <ArtStamp artName={operation.artName} alt={`${operation.name} illustration`} className="mx-auto h-40 w-auto object-contain" />
        </div>
        <div className="ta-operation-brief-copy">
          <p className="ta-condensed text-xs tracking-[0.2em] text-black/60">{isActivePlayer ? "YOUR OPERATION" : `${recipient.toUpperCase()}'S OPERATION`}</p>
          <h3 className="ta-display mt-2 text-4xl">{operation.name}</h3>
          <p className="ta-condensed mt-4 text-xs tracking-[0.18em] text-black/60">READ OUT LOUD BEFORE CONTINUING IN SECRET</p>
          <p className="ta-condensed mx-auto mt-2 max-w-sm text-base leading-tight">{roomBriefing(operation, recipient, isActivePlayer, publicInstruction)}</p>
        </div>
      </div>
      {secretOrders ? (
        <div className="ta-operation-state ta-operation-state-choice">
          <div>
            <p className="ta-condensed text-xs tracking-[0.16em] text-black/60">YOUR ORDERS &mdash; FOR YOUR EYES ONLY</p>
            <p className="ta-display mt-1 text-3xl">{secretOrders.name}</p>
            <p className="ta-condensed mt-2 text-base leading-tight">{projection?.private.operation_instruction}</p>
          </div>
          <ArtStamp artName={secretOrders.artName} alt="" className="h-16 w-auto shrink-0 object-contain" />
        </div>
      ) : null}
      {projection ? (
        <LiveOperationInput
          projection={projection}
          selectedTargets={selectedTargets}
          setSelectedTargets={setSelectedTargets}
          selectedChoice={selectedChoice}
          setSelectedChoice={setSelectedChoice}
        />
      ) : (
        <OperationInput key={operation.id} operation={operation} />
      )}
      <InkButton
        variant="orange"
        className="ta-operation-submit w-full"
        onClick={() => onNext(selectedTargets, selectedChoice)}
        disabled={Boolean(projection && (!projection.private.can_submit || (projection.public.phase === "OPERATION_INPUT" && !canSubmitForm)))}
      >
        {projection?.public.phase === "OPERATION_INPUT" ? (projection.private.can_submit ? "Confirm operation" : waitingLabel) : projection?.private.can_submit ? "Done" : waitingLabel}
      </InkButton>
    </div>
  );
}
