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
    const tick = () => setRemaining(Math.max(0, Math.ceil((target - Date.now()) / 1000)));
    tick();
    const interval = window.setInterval(tick, 250);
    return () => window.clearInterval(interval);
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
