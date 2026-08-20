"use client";

import { useState } from "react";
import type { ClientCommand, RoomProjection } from "../protocol";
import type { PendingCommand } from "../use-room";
import { ArtStamp, InkButton, PaperTitle } from "../ui";

export type AccusationScreenProps = {
  projection: RoomProjection;
  pending: PendingCommand | null;
  onSend(command: ClientCommand): void;
};

export function AccusationScreen({ projection, pending, onSend }: AccusationScreenProps) {
  const [targetId, setTargetId] = useState<string | null>(null);
  const alreadyVoted = projection.private.vote_submitted || !projection.private.can_submit;
  const busy = pending?.kind === "vote.submit";
  const legalTargets = projection.private.legal_target_ids;
  const targets = projection.public.players.filter((player) => {
    if (player.id === projection.private.player_id) return false;
    return !legalTargets || legalTargets.includes(player.id);
  });
  const waitingOn = projection.public.players.filter((player) => !player.vote_submitted).length;


  return (
    <div className="ta-rise ta-screen">
      <PaperTitle>Accusation time</PaperTitle>
      <div className="ta-paper p-5 text-center">
        <ArtStamp artName="accusation" alt="Accusation illustration" className="mx-auto h-32 w-auto object-contain" />
        <p className="ta-sans mt-2 text-base leading-snug">Choose the player you believe should be imprisoned. Your target stays hidden until the reveal.</p>
      </div>
      <div className="grid gap-2">
        {targets.map((target) => (
          <button
            className="ta-paper ta-choice-row flex items-center justify-between px-4 py-3 text-left transition-transform"
            aria-pressed={targetId === target.id}
            data-selected={targetId === target.id}
            disabled={alreadyVoted || busy}
            key={target.id}
            onClick={() => setTargetId(target.id)}
            type="button"
          >
            <span className="ta-sans text-lg">{target.name}</span>
            <span className={`h-4 w-4 rounded-full border-2 border-black ${targetId === target.id ? "bg-ta-red" : "bg-transparent"}`} />
          </button>
        ))}
      </div>
      {alreadyVoted ? (
        <div className="ta-paper p-4 text-center">
          <p className="ta-condensed text-xs tracking-[0.16em] text-black/60">YOUR ACCUSATION IS IN</p>
          <p className="ta-display mt-1 text-2xl">SUBMITTED</p>
          <p className="ta-sans mt-2 text-sm text-black/65">{waitingOn > 0 ? `Waiting for ${waitingOn} more ${waitingOn === 1 ? "player" : "players"} to vote.` : "Waiting for the room to finish voting."}</p>
        </div>
      ) : (
        <div className="flex items-center justify-between gap-3">
          <span className="ta-condensed text-xs tracking-[0.16em]">TARGET HIDDEN FROM ROOM</span>
          <InkButton
            variant="orange"
            disabled={!targetId || Boolean(pending && !busy)}
            busy={busy}
            busyLabel="Submitting vote…"
            onClick={() => targetId && onSend({ kind: "vote.submit", target_id: targetId })}
          >
            Submit vote
          </InkButton>
        </div>
      )}
    </div>
  );
}
