"use client";

import { useEffect, useState, useSyncExternalStore } from "react";

type InstallChoice = {
  outcome: "accepted" | "dismissed";
  platform: string;
};

interface BeforeInstallPromptEvent extends Event {
  prompt(): Promise<InstallChoice>;
  userChoice: Promise<InstallChoice>;
}

const DISMISS_KEY = "triple-agent-install-dismissed-v1";

const emptySubscribe = () => () => {};

function isIOSDevice() {
  if (typeof window === "undefined") return false;
  const userAgent = navigator.userAgent;
  const isAppleMobile = /iPhone|iPad|iPod/i.test(userAgent);
  const isIPadDesktopMode = navigator.platform === "MacIntel" && navigator.maxTouchPoints > 1;
  return isAppleMobile || isIPadDesktopMode;
}

function isStandalone() {
  if (typeof window === "undefined") return false;
  const appleStandalone = Boolean((navigator as Navigator & { standalone?: boolean }).standalone);
  return window.matchMedia("(display-mode: standalone)").matches || appleStandalone;
}

function wasDismissed() {
  if (typeof window === "undefined") return false;
  try {
    return window.localStorage.getItem(DISMISS_KEY) === "1";
  } catch {
    return false;
  }
}

function rememberDismissal() {
  try {
    window.localStorage.setItem(DISMISS_KEY, "1");
  } catch {
    // Private browsing can deny storage; the in-memory state still dismisses this view.
  }
}

export function InstallApp() {
  const [installPrompt, setInstallPrompt] = useState<BeforeInstallPromptEvent | null>(null);
  const isIOS = useSyncExternalStore(emptySubscribe, isIOSDevice, () => false);
  const isBrowserStandalone = useSyncExternalStore(emptySubscribe, isStandalone, () => false);
  const isBrowserDismissed = useSyncExternalStore(emptySubscribe, wasDismissed, () => false);
  const [installedState, setInstalledState] = useState(false);
  const [hasEngaged, setHasEngaged] = useState(false);
  const [dismissedState, setDismissedState] = useState(false);
  const [showIOSHelp, setShowIOSHelp] = useState(false);
  const [installing, setInstalling] = useState(false);

  const isInstalled = installedState || isBrowserStandalone;
  const dismissed = dismissedState || isBrowserDismissed;

  useEffect(() => {
    if (isStandalone()) {
      return;
    }

    const engage = () => setHasEngaged(true);
    const handleInstallPrompt = (event: Event) => {
      event.preventDefault();
      setInstallPrompt(event as BeforeInstallPromptEvent);
    };
    const handleInstalled = () => {
      setInstalledState(true);
      setInstallPrompt(null);
    };

    window.addEventListener("pointerdown", engage, { once: true, passive: true });
    window.addEventListener("keydown", engage, { once: true });
    window.addEventListener("beforeinstallprompt", handleInstallPrompt);
    window.addEventListener("appinstalled", handleInstalled);

    return () => {
      window.removeEventListener("pointerdown", engage);
      window.removeEventListener("keydown", engage);
      window.removeEventListener("beforeinstallprompt", handleInstallPrompt);
      window.removeEventListener("appinstalled", handleInstalled);
    };
  }, []);

  function dismiss() {
    rememberDismissal();
    setDismissedState(true);
    setShowIOSHelp(false);
  }

  async function install() {
    if (!installPrompt || installing) return;

    setInstalling(true);
    try {
      const choice = await installPrompt.prompt();
      if (choice.outcome === "accepted") setInstalledState(true);
    } catch {
      // The browser owns the prompt UI; a dismissed or unavailable prompt is not an app error.
    } finally {
      setInstallPrompt(null);
      setInstalling(false);
    }
  }

  if (isInstalled || dismissed || !hasEngaged) return null;

  const canShowChromiumInstall = Boolean(installPrompt);
  if (!isIOS && !canShowChromiumInstall) return null;

  if (isIOS && showIOSHelp) {
    return (
      <div className="fixed inset-x-3 bottom-3 z-50 mx-auto max-w-md" role="dialog" aria-labelledby="ios-install-title" aria-modal="false">
        <div className="ta-paper border-2 border-black p-4 text-left">
          <div className="flex items-start justify-between gap-4">
            <div>
              <p className="ta-condensed text-xs tracking-[0.18em] text-black/60">IPHONE / IPAD</p>
              <h2 id="ios-install-title" className="ta-display text-2xl">Install Triple Agent</h2>
            </div>
            <button className="ta-condensed min-h-11 min-w-11 border-2 border-black px-2 text-xl" type="button" onClick={dismiss} aria-label="Dismiss install help">×</button>
          </div>
          <ol className="ta-condensed mt-3 list-decimal space-y-1 pl-5 text-sm leading-relaxed">
            <li>Tap your browser&apos;s Share button.</li>
            <li>Choose <span className="font-bold">Add to Home Screen</span>.</li>
            <li>Tap <span className="font-bold">Add</span> to save the game.</li>
          </ol>
          <button className="ta-ink-button mt-4 w-full px-4 text-sm" type="button" onClick={() => setShowIOSHelp(false)}>Close</button>
        </div>
      </div>
    );
  }

  return (
    <div className="fixed inset-x-3 bottom-3 z-50 mx-auto max-w-md" role="status" aria-live="polite">
      <div className="ta-paper flex items-center gap-3 border-2 border-black p-3">
        <div className="min-w-0 flex-1">
          <p className="ta-display text-lg leading-none">Keep Triple Agent close</p>
          <p className="ta-condensed mt-1 text-xs text-black/70">Launch straight from your Home Screen.</p>
        </div>
        {isIOS ? (
          <button className="ta-ink-button shrink-0 px-3 text-xs" type="button" onClick={() => setShowIOSHelp(true)}>How to install</button>
        ) : (
          <button className="ta-ink-button shrink-0 px-3 text-xs" type="button" onClick={() => void install()} disabled={installing}>
            {installing ? "Opening…" : "Install"}
          </button>
        )}
        <button className="ta-condensed min-h-11 min-w-11 border-2 border-black px-2 text-xl" type="button" onClick={dismiss} aria-label="Dismiss install prompt">×</button>
      </div>
    </div>
  );
}
