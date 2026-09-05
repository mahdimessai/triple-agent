"use client";

import type { ReactNode } from "react";
import type { ClientCommand, RoomProjection } from "../protocol";
import type { PendingCommand } from "../use-room";
import { ArtStamp, InkButton, PaperTitle } from "../ui";
import { useRemainingSeconds } from "./use-remaining-seconds";

export type InterludeScreenProps = {
  projection: RoomProjection;
  pending: PendingCommand | null;
  onSend(command: ClientCommand): void;
};

function InterludeClock({ deadline, seconds, children }: { deadline?: string; seconds: number; children: ReactNode }) {
  const remaining = useRemainingSeconds(deadline, seconds);
  return (
    <>
      <div className="ta-clock" role="timer" aria-live="off" aria-label={`${remaining} seconds until the next operation`}>
        <ArtStamp artName="clock" alt="" className="ta-clock-face" />
        <span className="ta-clock-hand" aria-hidden="true"><ArtStamp artName="clockHand" alt="" className="h-full w-full object-contain" /></span>
        <span className="ta-clock-pin" aria-hidden="true"><ArtStamp artName="clockHandMiddle" alt="" className="h-full w-full object-contain" /></span>
      </div>
      {children}
      <p className="ta-display text-4xl text-ta-paper">{remaining}</p>
    </>
  );
}

export function InterludeScreen({ projection, pending, onSend }: InterludeScreenProps) {
  const seconds = projection.public.settings.interlude_seconds ?? 7;
  const deadline = projection.public.discussion_deadline;
  const busy = pending?.kind === "interlude.advance";

  return (
    <div className="ta-rise ta-screen items-center text-center">
      <PaperTitle>Waiting for next operation…</PaperTitle>
      <InterludeClock key={`${deadline}:${seconds}`} deadline={deadline} seconds={seconds}>
        <p className="ta-paper ta-sans w-full px-4 py-3 text-base leading-snug">Return the device to the table. You may tell the truth or lie about your new information.</p>
      </InterludeClock>
      {projection.private.can_submit ? (
        <InkButton variant="orange" className="w-full" onClick={() => onSend({ kind: "interlude.advance" })} disabled={Boolean(pending && !busy)} busy={busy} busyLabel="Skipping…">
          Skip to next operation
        </InkButton>
      ) : null}
    </div>
  );
}
