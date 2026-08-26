"use client";

import { useEffect, useState } from "react";
import type { ClientCommand, RoomProjection } from "../protocol";
import type { PendingCommand } from "../use-room";
import { ArtStamp, InkButton, PaperTitle } from "../ui";

export type InterludeScreenProps = {
  projection: RoomProjection;
  pending: PendingCommand | null;
  onSend(command: ClientCommand): void;
};

function useRemainingSeconds(deadline: string | undefined, fallback: number): number {
  const [remaining, setRemaining] = useState<number>(() => {
    if (!deadline) return fallback;
    const target = Date.parse(deadline);
    if (!Number.isFinite(target)) return fallback;
    return Math.max(0, Math.ceil((target - Date.now()) / 1000));
  });

  useEffect(() => {
    if (!deadline) return;
    const target = Date.parse(deadline);
    if (!Number.isFinite(target)) return;

    let timer: number | null = null;
    const stop = () => {
      if (timer !== null) {
        window.clearTimeout(timer);
        timer = null;
      }
    };
    const tick = () => {
      if (document.visibilityState !== "visible") {
        stop();
        return;
      }
      const remainingMs = target - Date.now();
      const seconds = Math.max(0, Math.ceil(remainingMs / 1000));
      setRemaining(seconds);
      stop();
      if (seconds > 0) timer = window.setTimeout(tick, Math.max(50, remainingMs % 1000 || 1000));
    };
    const onVisibility = () => {
      stop();
      if (document.visibilityState === "visible") tick();
    };

    const initialMs = target - Date.now();
    const initialSeconds = Math.max(0, Math.ceil(initialMs / 1000));
    if (initialSeconds > 0) {
      timer = window.setTimeout(tick, Math.max(50, initialMs % 1000 || 1000));
    }
    document.addEventListener("visibilitychange", onVisibility);
    return () => {
      stop();
      document.removeEventListener("visibilitychange", onVisibility);
    };
  }, [deadline, fallback]);

  return deadline ? remaining : fallback;
}

export function InterludeScreen({ projection, pending, onSend }: InterludeScreenProps) {
  const seconds = projection.public.settings.interlude_seconds ?? 7;
  const remaining = useRemainingSeconds(projection.public.discussion_deadline, seconds);
  const busy = pending?.kind === "interlude.advance";

  return (
    <div className="ta-rise ta-screen items-center text-center">
      <PaperTitle>Waiting for next operation…</PaperTitle>
      <div className="ta-clock" role="timer" aria-live="off" aria-label={`${remaining} seconds until the next operation`}>
        <ArtStamp artName="clock" alt="" className="ta-clock-face" />
        <span className="ta-clock-hand" aria-hidden="true"><ArtStamp artName="clockHand" alt="" className="h-full w-full object-contain" /></span>
        <span className="ta-clock-pin" aria-hidden="true"><ArtStamp artName="clockHandMiddle" alt="" className="h-full w-full object-contain" /></span>
      </div>
      <p className="ta-paper ta-sans w-full px-4 py-3 text-base leading-snug">Return the device to the table. You may tell the truth or lie about your new information.</p>
      <p className="ta-display text-4xl text-ta-paper">{remaining}</p>
      {projection.private.can_submit ? (
        <InkButton variant="orange" className="w-full" onClick={() => onSend({ kind: "interlude.advance" })} disabled={Boolean(pending && !busy)} busy={busy} busyLabel="Skipping…">
          Skip to next operation
        </InkButton>
      ) : null}
    </div>
  );
}
