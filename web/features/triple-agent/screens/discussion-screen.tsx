import { useEffect, useRef, useState } from "react";
import { InkButton } from "@/components/ui/ink-button";
import { PaperTitle } from "@/components/ui/paper-title";
import type { RoomProjection } from "@/components/triple-agent/server-client";

function AnimatedStopwatch({
  enabled,
  deadline,
  durationSeconds = 300,
}: {
  enabled: boolean;
  deadline?: string;
  durationSeconds?: number;
}) {
  const [secondsRemaining, setSecondsRemaining] = useState<number>(() => {
    if (!enabled || !deadline) return durationSeconds;
    const deadlineMs = Date.parse(deadline);
    if (!Number.isFinite(deadlineMs)) return durationSeconds;
    return Math.max(0, Math.ceil((deadlineMs - Date.now()) / 1000));
  });
  const timeoutRef = useRef<number | undefined>(undefined);

  useEffect(() => {
    if (!enabled || !deadline) {
      return;
    }
    const deadlineMs = Date.parse(deadline);
    if (!Number.isFinite(deadlineMs)) return;

    const stop = () => {
      if (timeoutRef.current !== undefined) {
        window.clearTimeout(timeoutRef.current);
        timeoutRef.current = undefined;
      }
    };

    const update = () => {
      if (document.visibilityState !== "visible") {
        stop();
        return;
      }
      const remainingMs = deadlineMs - Date.now();
      const next = Math.max(0, Math.ceil(remainingMs / 1000));
      setSecondsRemaining(next);
      stop();
      if (next > 0) {
        const delay = Math.max(50, remainingMs % 1000 || 1000);
        timeoutRef.current = window.setTimeout(update, delay);
      }
    };

    const handleVisibilityChange = () => {
      stop();
      if (document.visibilityState === "visible") update();
    };

    update();
    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => {
      stop();
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, [enabled, deadline, durationSeconds]);

  const currentSeconds = secondsRemaining ?? durationSeconds;
  const isUrgent = currentSeconds <= 30 && currentSeconds > 0;
  const minutes = Math.floor(currentSeconds / 60);
  const seconds = String(currentSeconds % 60).padStart(2, "0");

  const progress = durationSeconds > 0 ? Math.max(0, Math.min(1, currentSeconds / durationSeconds)) : 1;
  const radius = 54;
  const circumference = 2 * Math.PI * radius;
  const strokeDashoffset = circumference * (1 - progress);
  // Second hand angle (rotates 360 deg every 60s)
  const handAngle = ((durationSeconds - currentSeconds) % 60) * 6;

  return (
    <div className="flex flex-col items-center">
      {/* Stopwatch SVG */}
      <div className="relative my-2 flex items-center justify-center">
        <svg
          width="160"
          height="160"
          viewBox="0 0 160 160"
          className={`overflow-visible transition-transform duration-300 ${isUrgent ? "scale-105" : ""}`}
          aria-hidden="true"
        >
          {/* Stopwatch Top Crown / Buttons */}
          <rect x="74" y="6" width="12" height="10" rx="2" fill="#1d1b18" />
          <rect x="68" y="2" width="24" height="6" rx="2" fill="#1d1b18" />
          <rect x="114" y="20" width="8" height="12" rx="2" transform="rotate(35 118 26)" fill="#1d1b18" />

          {/* Outer Ring & Dial Shadow */}
          <circle cx="80" cy="88" r="66" fill="#f4ede0" stroke="#1d1b18" strokeWidth="6" />
          <circle cx="80" cy="88" r="54" fill="none" stroke="#d5c8b3" strokeWidth="8" />

          {/* Progress Ring */}
          <circle
            cx="80"
            cy="88"
            r={radius}
            fill="none"
            stroke={isUrgent ? "#c8372d" : "#2a6282"}
            strokeWidth="8"
            strokeLinecap="round"
            strokeDasharray={circumference}
            strokeDashoffset={strokeDashoffset}
            transform="rotate(-90 80 88)"
            className="transition-all duration-300 ease-linear"
          />

          {/* Hour & Minute Tick Marks */}
          {[0, 30, 60, 90, 120, 150, 180, 210, 240, 270, 300, 330].map((deg) => (
            <line
              key={deg}
              x1="80"
              y1="40"
              x2="80"
              y2={deg % 90 === 0 ? "46" : "43"}
              stroke="#1d1b18"
              strokeWidth={deg % 90 === 0 ? "3" : "1.5"}
              transform={`rotate(${deg} 80 88)`}
            />
          ))}

          {/* Center Pivot */}
          <circle cx="80" cy="88" r="5" fill="#1d1b18" />

          {/* Sweeping / Ticking Stopwatch Hand */}
          <line
            x1="80"
            y1="88"
            x2="80"
            y2="44"
            stroke={isUrgent ? "#c8372d" : "#1d1b18"}
            strokeWidth="3"
            strokeLinecap="round"
            transform={`rotate(${handAngle} 80 88)`}
            className="transition-transform duration-200"
          />
          <circle cx="80" cy="88" r="2" fill="#fff" />
        </svg>
      </div>

      {/* Digital Countdown readout */}
      <div className="mt-1 text-center">
        <p
          role="timer"
          aria-label={`${minutes} minutes ${seconds} seconds remaining`}
          className={`ta-display text-4xl sm:text-5xl font-black tracking-wider ${
            isUrgent ? "text-ta-red animate-pulse" : "text-black"
          }`}
        >
          {minutes}:{seconds}
        </p>
        <p className="ta-condensed text-[0.7rem] tracking-[0.18em] text-black/60 mt-1">
          {enabled
            ? isUrgent
              ? "FINAL SECONDS BEFORE ACCUSATION"
              : "ACCUSATIONS OPEN AUTOMATICALLY AT ZERO"
            : "DISCUSSION TIMER PAUSED"}
        </p>
      </div>
    </div>
  );
}

export function DiscussionScreen({
  timerEnabled,
  canAdvance = true,
  projection,
  loading = false,
  onNext,
}: {
  timerEnabled: boolean;
  canAdvance?: boolean;
  projection?: RoomProjection;
  loading?: boolean;
  onNext: () => void;
}) {
  const isTimerActive = projection ? projection.public.settings.discussion_timer_enabled : timerEnabled;
  const totalPlayers = projection?.public.players.length ?? 5;
  const readyCount = projection?.public.discussion_ready_count ?? 0;
  const durationSeconds = projection?.public.settings.discussion_seconds ?? 300;

  const hasVoted = !canAdvance;
  const buttonText = hasVoted
    ? `Waiting for table (${readyCount} / ${totalPlayers} ready)`
    : "Vote to open accusations";

  return (
    <div className="ta-rise ta-screen">
      <PaperTitle>Discussion phase</PaperTitle>

      <div className="ta-paper p-5 text-center">
        <AnimatedStopwatch
          enabled={isTimerActive}
          deadline={projection?.public.discussion_deadline}
          durationSeconds={durationSeconds}
        />
        <p className="ta-condensed mt-4 text-base leading-tight text-black/80">
          Discuss suspicions and share clues. Accusations start when the timer expires or when all players vote to accuse early.
        </p>
      </div>

      {readyCount > 0 ? (
        <div className="ta-paper flex items-center justify-between gap-3 px-4 py-3">
          <span className="ta-condensed text-xs tracking-[0.16em]">EARLY ACCUSATION VOTE</span>
          <span className="ta-condensed text-sm font-bold">
            {readyCount} / {totalPlayers} PLAYERS READY
          </span>
        </div>
      ) : null}

      <div className="flex flex-col gap-2">
        <InkButton variant="orange" className="w-full" onClick={onNext} disabled={hasVoted} loading={loading} loadingLabel="Saving vote…">
          {buttonText}
        </InkButton>
        {hasVoted ? (
          <p className="ta-condensed text-center text-xs text-white/80">
            Your vote is recorded. Accusations begin when all {totalPlayers} players are ready.
          </p>
        ) : null}
      </div>
    </div>
  );
}
