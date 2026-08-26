import Image from "next/image";
import { type ButtonHTMLAttributes, type ReactNode, useCallback, useEffect, useRef, useState } from "react";
import { art, type ArtName } from "./assets";

export type InkButtonProps = Omit<ButtonHTMLAttributes<HTMLButtonElement>, "type"> & {
  children: ReactNode;
  variant?: "orange";
  busy?: boolean;
  busyLabel?: string;
};

export function InkButton({ children, variant, className = "", busy = false, busyLabel, disabled = false, ...props }: InkButtonProps) {
  const isDisabled = Boolean(disabled || busy);
  return (
    <button
      {...props}
      className={`ta-ink-button px-5 ${className}`}
      data-variant={variant}
      disabled={isDisabled}
      aria-busy={busy ? "true" : undefined}
      type="button"
    >
      {busy ? (
        <span className="ta-button-loading">
          <span className="ta-button-spinner" aria-hidden="true" />
          {busyLabel ?? children}
        </span>
      ) : children}
    </button>
  );
}

export function PaperTitle({ children, className = "" }: { children: ReactNode; className?: string }) {
  return (
    <h2 className={`ta-paper ta-display ta-angle-left px-4 py-3 text-center text-[clamp(1.25rem,4vw,2.3rem)] ${className}`}>
      {children}
    </h2>
  );
}

export type ArtStampProps = {
  artName: ArtName;
  alt?: string;
  className?: string;
  priority?: boolean;
  sizes?: string;
};

export function ArtStamp({ artName, alt = "", className = "", priority = false, sizes = "(max-width: 640px) 50vw, 320px" }: ArtStampProps) {
  const item = art[artName];
  return (
    <Image
      src={item.src}
      alt={alt}
      width={item.width}
      height={item.height}
      priority={priority}
      loading={priority ? "eager" : "lazy"}
      sizes={sizes}
      className={className}
    />
  );
}

export function FingerprintGraphic({ className = "h-24 w-20", scanning = false }: { className?: string; scanning?: boolean }) {
  return (
    <svg
      viewBox="0 0 100 120"
      fill="none"
      stroke="currentColor"
      strokeWidth="2.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={`${className} transition-all duration-200 ${scanning ? "scale-105 text-ta-teal" : "text-ta-ink/75"}`}
      aria-hidden="true"
    >
      <path d="M50 48 c-4 0 -7 3 -7 7 c0 8 7 12 7 19 c0 4 -2 7 -5 7" />
      <path d="M50 40 c-9 0 -15 6 -15 15 c0 11 12 16 12 24 c0 6 -4 10 -9 10 c-3 0 -6 -2 -7 -5" />
      <path d="M50 32 c-14 0 -22 9 -22 22 c0 13 14 18 14 29 c0 8 -5 13 -12 13 c-5 0 -9 -3 -11 -7" />
      <path d="M50 24 c-19 0 -29 11 -29 28 c0 16 16 21 16 34 c0 9 -6 15 -14 15 c-4 0 -7 -2 -9 -5" />
      <path d="M50 16 c-23 0 -35 14 -35 34 c0 18 17 25 17 38 c0 9 -5 16 -12 16" />
      <path d="M50 16 c23 0 35 14 35 34 c0 18 -17 25 -17 38 c0 9 5 16 12 16" />
      <path d="M50 24 c19 0 29 11 29 28 c0 16 -16 21 -16 34 c0 9 6 15 14 15 c4 0 7 -2 9 -5" />
      <path d="M50 32 c14 0 22 9 22 22 c0 13 -14 18 -14 29 c0 8 5 13 12 13 c5 0 9 -3 11 -7" />
      <path d="M50 40 c9 0 15 6 15 15 c0 11 -12 16 -12 24 c0 6 4 10 9 10 c3 0 6 -2 7 -5" />
      <path d="M50 48 c4 0 7 3 7 7 c0 8 -7 12 -7 19 c0 4 2 7 5 7" />
      <circle cx="50" cy="55" r="2.5" fill="currentColor" stroke="none" />
    </svg>
  );
}

export type BiometricScannerProps = {
  onComplete(): void;
  holdDurationMs?: number;
  disabled?: boolean;
  className?: string;
  label?: string;
  sublabel?: string;
};

export function BiometricScanner({
  onComplete,
  holdDurationMs = 1500,
  disabled = false,
  className = "",
  label,
  sublabel,
}: BiometricScannerProps) {
  const [progress, setProgress] = useState(0);
  const [isHolding, setIsHolding] = useState(false);
  const [isCompleted, setIsCompleted] = useState(false);
  const holdStartRef = useRef<number | null>(null);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const clearTimer = useCallback(() => {
    if (timerRef.current) {
      clearInterval(timerRef.current);
      timerRef.current = null;
    }
  }, []);

  const handleHoldStart = useCallback(() => {
    if (disabled || isCompleted) return;
    setIsHolding(true);
    setProgress(0);
    const startTime = Date.now();
    holdStartRef.current = startTime;

    clearTimer();
    timerRef.current = setInterval(() => {
      const elapsed = Date.now() - startTime;
      const currentProgress = Math.min(1, elapsed / holdDurationMs);
      setProgress(currentProgress);

      if (currentProgress >= 1) {
        clearTimer();
        setIsCompleted(true);
        setIsHolding(false);
        if (typeof navigator !== "undefined" && navigator.vibrate) {
          try {
            navigator.vibrate([40, 30, 40]);
          } catch {
            // ignore vibration failures
          }
        }
        onComplete();
      }
    }, 20);
  }, [clearTimer, disabled, holdDurationMs, isCompleted, onComplete]);

  const handleHoldEnd = useCallback(() => {
    if (isCompleted) return;
    clearTimer();
    setIsHolding(false);
    setProgress(0);
    holdStartRef.current = null;
  }, [clearTimer, isCompleted]);

  useEffect(() => {
    return () => {
      clearTimer();
    };
  }, [clearTimer]);

  const radius = 54;
  const circumference = 2 * Math.PI * radius;
  const strokeDashoffset = circumference * (1 - progress);
  const percent = Math.round(progress * 100);

  return (
    <div className={`flex flex-col items-center select-none ${className}`}>
      <div
        role="button"
        tabIndex={disabled ? -1 : 0}
        aria-label={label ?? "Biometric scanner: press and hold for 1.5 seconds to decrypt"}
        onPointerDown={(e) => {
          if (e.button !== undefined && e.button !== 0) return;
          e.preventDefault();
          handleHoldStart();
        }}
        onPointerUp={handleHoldEnd}
        onPointerLeave={handleHoldEnd}
        onPointerCancel={handleHoldEnd}
        onTouchStart={(e) => {
          e.preventDefault();
          handleHoldStart();
        }}
        onTouchEnd={handleHoldEnd}
        onTouchCancel={handleHoldEnd}
        onMouseDown={(e) => {
          if (e.button === 0) handleHoldStart();
        }}
        onMouseUp={handleHoldEnd}
        onMouseLeave={handleHoldEnd}
        onKeyDown={(e) => {
          if ((e.key === " " || e.key === "Enter") && !e.repeat) {
            e.preventDefault();
            handleHoldStart();
          }
        }}
        onKeyUp={(e) => {
          if (e.key === " " || e.key === "Enter") {
            e.preventDefault();
            handleHoldEnd();
          }
        }}
        onContextMenu={(e) => e.preventDefault()}
        className={`ta-biometric-pad relative flex h-48 w-44 cursor-pointer flex-col items-center justify-center overflow-hidden border-4 border-black bg-ta-paper p-4 shadow-[5px_5px_0_var(--ta-shadow)] transition-transform duration-150 active:translate-x-0.5 active:translate-y-0.5 active:shadow-[3px_3px_0_var(--ta-shadow)] ${
          isHolding ? "bg-[#e8f5f1] ring-4 ring-ta-teal/40" : "hover:bg-[#fff8e8]"
        }`}
      >
        {/* Animated scanning laser beam */}
        {isHolding ? <div className="ta-scanner-beam" /> : null}

        {/* Circular Progress Ring */}
        <svg
          role="progressbar"
          aria-label="Scan progress"
          aria-valuenow={percent}
          aria-valuemin={0}
          aria-valuemax={100}
          className="absolute inset-0 h-full w-full -rotate-90 pointer-events-none p-2"
          viewBox="0 0 120 120"
        >
          <circle
            cx="60"
            cy="60"
            r={radius}
            fill="none"
            stroke="rgb(18 16 14 / 0.12)"
            strokeWidth="5"
          />
          <circle
            cx="60"
            cy="60"
            r={radius}
            fill="none"
            stroke="var(--ta-teal, #51d3c2)"
            strokeWidth="6"
            strokeLinecap="round"
            strokeDasharray={circumference}
            strokeDashoffset={strokeDashoffset}
            className="transition-all duration-75"
          />
        </svg>

        {/* Fingerprint Graphic */}
        <FingerprintGraphic scanning={isHolding} className="relative z-10 h-28 w-24" />

        {/* Overlay scan status text */}
        <div className="relative z-10 mt-1 text-center">
          <span className="ta-condensed text-[0.7rem] font-bold tracking-[0.14em] uppercase text-ta-ink">
            {isHolding ? `SCANNING ${percent}%` : "PLACE THUMB"}
          </span>
        </div>
      </div>

      <div className="mt-3 text-center">
        <p className="ta-condensed text-xs tracking-[0.16em] uppercase text-ta-ink/80 font-bold">
          {label ?? "PRESS & HOLD TO DECRYPT"}
        </p>
        <p className="ta-sans text-xs text-black/60">
          {sublabel ?? "Hold thumb on scanner for 1.5 seconds"}
        </p>
      </div>
    </div>
  );
}

export const FingerprintScanner = BiometricScanner;

