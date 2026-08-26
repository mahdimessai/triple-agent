"use client";

import { useEffect, useState } from "react";
import type { ClientCommand, RoomProjection } from "../protocol";
import type { PendingCommand } from "../use-room";
import { InkButton, PaperTitle } from "../ui";

export type DiscussionScreenProps = {
  projection: RoomProjection;
  pending: PendingCommand | null;
  onSend(command: ClientCommand): void;
};

function useRemainingSeconds(deadline: string | undefined, enabled: boolean, fallback: number): number {
  const [remaining, setRemaining] = useState<number>(() => {
    if (!enabled || !deadline) return fallback;
    const target = Date.parse(deadline);
    if (!Number.isFinite(target)) return fallback;
    return Math.max(0, Math.ceil((target - Date.now()) / 1000));
  });

  useEffect(() => {
    if (!enabled || !deadline) return;
    const target = Date.parse(deadline);
    if (!Number.isFinite(target)) return;
    let timer: number | null = null;
    const tick = () => {
      const remainingMs = target - Date.now();
      const next = Math.max(0, Math.ceil(remainingMs / 1000));
      setRemaining(next);
      if (timer !== null) window.clearTimeout(timer);
      timer = next > 0 ? window.setTimeout(tick, Math.max(50, remainingMs % 1000 || 1000)) : null;
    };
    const onVisibility = () => {
      if (timer !== null) window.clearTimeout(timer);
      timer = null;
      if (document.visibilityState === "visible") tick();
    };
    const initialMs = target - Date.now();
    const initialSeconds = Math.max(0, Math.ceil(initialMs / 1000));
    if (initialSeconds > 0) {
      timer = window.setTimeout(tick, Math.max(50, initialMs % 1000 || 1000));
    }
    document.addEventListener("visibilitychange", onVisibility);
    return () => {
      if (timer !== null) window.clearTimeout(timer);
      document.removeEventListener("visibilitychange", onVisibility);
    };
  }, [deadline, enabled, fallback]);

  return enabled && deadline ? remaining : fallback;
}

function Stopwatch({ seconds, duration, enabled }: { seconds: number; duration: number; enabled: boolean }) {
  const urgent = enabled && seconds <= 30 && seconds > 0;
  const minutes = Math.floor(seconds / 60);
  const remainder = String(seconds % 60).padStart(2, "0");
  const progress = duration > 0 ? Math.max(0, Math.min(1, seconds / duration)) : 1;
  const radius = 54;
  const circumference = 2 * Math.PI * radius;
  const handAngle = ((duration - seconds) % 60) * 6;

  return (
    <div className="flex flex-col items-center">
      <div className="relative my-2 flex items-center justify-center">
        <svg width="160" height="160" viewBox="0 0 160 160" className={`overflow-visible ${urgent ? "scale-105" : ""}`} aria-hidden="true">
          <rect x="74" y="6" width="12" height="10" rx="2" fill="#1d1b18" />
          <rect x="68" y="2" width="24" height="6" rx="2" fill="#1d1b18" />
          <circle cx="80" cy="88" r="66" fill="#f4ede0" stroke="#1d1b18" strokeWidth="6" />
          <circle cx="80" cy="88" r={radius} fill="none" stroke="#d5c8b3" strokeWidth="8" />
          <circle cx="80" cy="88" r={radius} fill="none" stroke={urgent ? "#c8372d" : "#2a6282"} strokeWidth="8" strokeLinecap="round" strokeDasharray={circumference} strokeDashoffset={circumference * (1 - progress)} transform="rotate(-90 80 88)" />
          <line x1="80" y1="88" x2="80" y2="44" stroke={urgent ? "#c8372d" : "#1d1b18"} strokeWidth="3" strokeLinecap="round" transform={`rotate(${handAngle} 80 88)`} />
          <circle cx="80" cy="88" r="5" fill="#1d1b18" />
        </svg>
      </div>
      <p role="timer" aria-label={`${minutes} minutes ${remainder} seconds remaining`} className={`ta-display text-4xl font-black tracking-wider sm:text-5xl ${urgent ? "text-ta-red" : "text-black"}`}>{minutes}:{remainder}</p>
      <p className="ta-condensed mt-1 text-[0.7rem] tracking-[0.18em] text-black/60">{enabled ? urgent ? "FINAL SECONDS BEFORE ACCUSATION" : "ACCUSATIONS OPEN AUTOMATICALLY AT ZERO" : "DISCUSSION TIMER PAUSED"}</p>
    </div>
  );
}

export function DiscussionScreen({ projection, pending, onSend }: DiscussionScreenProps) {
  const settings = projection.public.settings;
  const duration = settings.discussion_seconds;
  const remaining = useRemainingSeconds(projection.public.discussion_deadline, settings.discussion_timer_enabled, duration);
  const totalPlayers = projection.public.players.length;
  const readyCount = projection.public.discussion_ready_count ?? 0;
  const busy = pending?.kind === "discussion.advance";

  return (
    <div className="ta-rise ta-screen">
      <PaperTitle>Discussion phase</PaperTitle>
      <div className="ta-paper p-5 text-center">
        <Stopwatch seconds={remaining} duration={duration} enabled={settings.discussion_timer_enabled} />
        <p className="ta-sans mt-4 text-base leading-snug text-black/80">Discuss suspicions and share clues. Accusations start when the timer expires or when all players vote to accuse early.</p>
      </div>
      {readyCount > 0 ? (
        <div className="ta-paper flex items-center justify-between gap-3 px-4 py-3">
          <span className="ta-condensed text-xs tracking-[0.16em]">EARLY ACCUSATION VOTE</span>
          <span className="ta-sans text-sm">{readyCount} / {totalPlayers} PLAYERS READY</span>
        </div>
      ) : null}
      <div className="flex flex-col gap-2">
        <InkButton
          variant="orange"
          className="w-full"
          onClick={() => onSend({ kind: "discussion.advance" })}
          disabled={!projection.private.can_submit || Boolean(pending && !busy)}
          busy={busy}
          busyLabel="Saving vote…"
        >
          {projection.private.can_submit ? "Vote to open accusations" : `Waiting for table (${readyCount} / ${totalPlayers} ready)`}
        </InkButton>
        {!projection.private.can_submit ? <p className="ta-condensed text-center text-xs text-white/80">Your vote is recorded. Accusations begin when the table is ready.</p> : null}
      </div>
    </div>
  );
}
