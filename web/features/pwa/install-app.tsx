"use client";

import { useEffect, useState } from "react";

type InstallChoice = { outcome: "accepted" | "dismissed"; platform: string };
interface BeforeInstallPromptEvent extends Event {
  prompt(): Promise<InstallChoice>;
  userChoice: Promise<InstallChoice>;
}

const DISMISS_KEY = "triple-agent-install-dismissed-v1";

function browserInstallState() {
  const standalone = window.matchMedia("(display-mode: standalone)").matches || Boolean((navigator as Navigator & { standalone?: boolean }).standalone);
  const ios = /iPhone|iPad|iPod/i.test(navigator.userAgent) || (navigator.platform === "MacIntel" && navigator.maxTouchPoints > 1);
  let dismissed = false;
  try { dismissed = window.localStorage.getItem(DISMISS_KEY) === "1"; } catch {}
  return { standalone, ios, dismissed };
}

export function InstallApp() {
  const [ready, setReady] = useState(false);
  const [ios, setIos] = useState(false);
  const [installed, setInstalled] = useState(false);
  const [dismissed, setDismissed] = useState(false);
  const [engaged, setEngaged] = useState(false);
  const [prompt, setPrompt] = useState<BeforeInstallPromptEvent | null>(null);
  const [showIOSHelp, setShowIOSHelp] = useState(false);
  const [installing, setInstalling] = useState(false);

  useEffect(() => {
    const state = browserInstallState();
    const syncTimer = window.setTimeout(() => {
      setIos(state.ios);
      setInstalled(state.standalone);
      setDismissed(state.dismissed);
      setReady(true);
    }, 0);
    if (state.standalone) return () => window.clearTimeout(syncTimer);

    const engage = () => setEngaged(true);
    const onPrompt = (event: Event) => {
      event.preventDefault();
      setPrompt(event as BeforeInstallPromptEvent);
    };
    const onInstalled = () => { setInstalled(true); setPrompt(null); };
    window.addEventListener("pointerdown", engage, { once: true, passive: true });
    window.addEventListener("keydown", engage, { once: true });
    window.addEventListener("beforeinstallprompt", onPrompt);
    window.addEventListener("appinstalled", onInstalled);
    return () => {
      window.clearTimeout(syncTimer);
      window.removeEventListener("pointerdown", engage);
      window.removeEventListener("keydown", engage);
      window.removeEventListener("beforeinstallprompt", onPrompt);
      window.removeEventListener("appinstalled", onInstalled);
    };
  }, []);

  function dismiss() {
    try { window.localStorage.setItem(DISMISS_KEY, "1"); } catch {}
    setDismissed(true);
    setShowIOSHelp(false);
  }

  async function install() {
    if (!prompt || installing) return;
    setInstalling(true);
    try {
      const choice = await prompt.prompt();
      if (choice.outcome === "accepted") setInstalled(true);
    } finally {
      setPrompt(null);
      setInstalling(false);
    }
  }

  if (!ready || installed || dismissed || !engaged || (!ios && !prompt)) return null;

  if (ios && showIOSHelp) {
    return (
      <div className="fixed inset-x-3 bottom-3 z-50 mx-auto max-w-md" role="dialog" aria-labelledby="ios-install-title">
        <div className="ta-paper p-4">
          <div className="flex items-start justify-between gap-4">
            <div><p className="ta-condensed text-xs tracking-[0.18em] text-black/60">IPHONE / IPAD</p><h2 id="ios-install-title" className="ta-display text-2xl">Install Triple Agent</h2></div>
            <button className="ta-secondary-button" type="button" onClick={dismiss} aria-label="Dismiss install help">×</button>
          </div>
          <ol className="ta-condensed mt-3 list-decimal space-y-1 pl-5 text-sm"><li>Tap your browser&apos;s Share button.</li><li>Choose <strong>Add to Home Screen</strong>.</li><li>Tap <strong>Add</strong>.</li></ol>
          <button className="ta-ink-button mt-4 w-full px-4" type="button" onClick={() => setShowIOSHelp(false)}>Close</button>
        </div>
      </div>
    );
  }

  return (
    <div className="fixed inset-x-3 bottom-3 z-50 mx-auto max-w-md" role="status" aria-live="polite">
      <div className="ta-paper flex items-center gap-3 p-3">
        <div className="min-w-0 flex-1"><p className="ta-display text-lg">Keep Triple Agent close</p><p className="ta-condensed text-xs text-black/70">Launch straight from your Home Screen.</p></div>
        {ios ? <button className="ta-ink-button px-3 text-xs" type="button" onClick={() => setShowIOSHelp(true)}>How to install</button> : <button className="ta-ink-button px-3 text-xs" type="button" onClick={() => void install()} disabled={installing}>{installing ? "Opening…" : "Install"}</button>}
        <button className="ta-secondary-button" type="button" onClick={dismiss} aria-label="Dismiss install prompt">×</button>
      </div>
    </div>
  );
}
