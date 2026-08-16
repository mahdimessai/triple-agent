"use client";

import { useEffect, useState } from "react";
import { ArtStamp } from "@/components/ui/art-stamp";
import { InkButton } from "@/components/ui/ink-button";
import { PaperTitle } from "@/components/ui/paper-title";

/**
 * The beat between two operations. The device goes back on the table, the room
 * talks about whatever just happened, and the server deals the next operation
 * when the deadline passes, the host can cut it short.
 */
export function InterludeScreen({ deadline, seconds = 7, isHost = false, onSkip }: { deadline?: string; seconds?: number; isHost?: boolean; onSkip: () => void }) {
  const [remaining, setRemaining] = useState(seconds);

  useEffect(() => {
    if (!deadline) return;
    const target = new Date(deadline).getTime();
    if (!Number.isFinite(target)) return;

    let timeout: number | undefined;
    const stop = () => {
      if (timeout !== undefined) {
        window.clearTimeout(timeout);
        timeout = undefined;
      }
    };
    const tick = () => {
      if (document.visibilityState !== "visible") {
        stop();
        return;
      }
      const remainingMs = target - Date.now();
      const next = Math.max(0, Math.ceil(remainingMs / 1000));
      setRemaining(next);
      stop();
      if (next > 0) {
        const delay = Math.max(50, remainingMs % 1000 || 1000);
        timeout = window.setTimeout(tick, delay);
      }
    };
    const handleVisibilityChange = () => {
      stop();
      if (document.visibilityState === "visible") tick();
    };

    tick();
    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => {
      stop();
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, [deadline]);

  return (
    <div className="ta-rise ta-screen items-center text-center">
      <PaperTitle>Waiting for next operation…</PaperTitle>
      <div className="ta-clock" role="timer" aria-live="off" aria-label={`${remaining} seconds until the next operation`}>
        <ArtStamp artName="clock" alt="" className="ta-clock-face" />
        <span className="ta-clock-hand" aria-hidden="true">
          <ArtStamp artName="clockHand" alt="" className="h-full w-full object-contain" />
        </span>
        <span className="ta-clock-pin" aria-hidden="true">
          <ArtStamp artName="clockHandMiddle" alt="" className="h-full w-full object-contain" />
        </span>
      </div>
      <p className="ta-paper ta-condensed w-full px-4 py-3 text-base leading-tight">Return the device to the table. You may tell the truth or lie about your new information.</p>
      <p className="ta-display text-4xl text-ta-paper">{remaining}</p>
      {isHost ? <InkButton variant="orange" className="w-full" onClick={onSkip}>Skip to next operation</InkButton> : null}
    </div>
  );
}
