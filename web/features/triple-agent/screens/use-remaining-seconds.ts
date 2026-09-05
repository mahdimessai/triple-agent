import { useEffect, useState } from "react";

// Call from a clock keyed by deadline so a new deadline starts with fresh state.
export function useRemainingSeconds(deadline: string | undefined, fallback: number): number {
  const target = deadline ? Date.parse(deadline) : NaN;
  const [remaining, setRemaining] = useState(() => Number.isFinite(target)
    ? Math.max(0, Math.ceil((target - Date.now()) / 1000))
    : fallback);

  useEffect(() => {
    if (!Number.isFinite(target)) return;
    let timer: number | undefined;
    const stop = () => window.clearTimeout(timer);
    const schedule = () => {
      const milliseconds = target - Date.now();
      if (milliseconds > 0 && document.visibilityState === "visible") {
        timer = window.setTimeout(tick, Math.max(50, milliseconds % 1000 || 1000));
      }
    };
    const tick = () => {
      setRemaining(Math.max(0, Math.ceil((target - Date.now()) / 1000)));
      schedule();
    };
    const onVisibility = () => {
      stop();
      if (document.visibilityState === "visible") tick();
    };
    schedule();
    document.addEventListener("visibilitychange", onVisibility);
    return () => {
      stop();
      document.removeEventListener("visibilitychange", onVisibility);
    };
  }, [target]);

  return remaining;
}
