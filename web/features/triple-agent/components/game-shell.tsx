"use client";

import { useRef, type ReactNode } from "react";
import type { ScreenId } from "@/features/triple-agent/model/screen";
import { GameHeader } from "./game-header";

export function GameShell({ children, screen, setScreen, session, connectionState = "closed" }: { children: ReactNode; screen: ScreenId; setScreen: (screen: ScreenId) => void; session?: unknown; connectionState?: "connecting" | "open" | "closed" }) {
  const liveSession = Boolean(session);
  // Settings is a detour, not a destination: remember the screen it was opened
  // from so the same button returns the player there.
  const screenBeforeSettings = useRef<ScreenId>("lobby");
  const offline = liveSession && connectionState !== "open";

  function toggleSettings() {
    if (screen === "settings") setScreen(screenBeforeSettings.current);
    else {
      screenBeforeSettings.current = screen;
      setScreen("settings");
    }
  }

  return (
    <main className="ta-viewport">
      <section className="ta-device">
        <GameHeader screen={screen} toggleSettings={toggleSettings} liveSession={liveSession} />
        {offline ? <p className="ta-connection-banner" role="status">{connectionState === "connecting" ? "Reconnecting to the room…" : "Connection lost: your actions will not reach the room"}</p> : null}
        <div className="ta-stage"><div className="ta-stage-inner">{children}</div></div>
      </section>
    </main>
  );
}
