"use client";

import { useRef, type ReactNode } from "react";
import type { ScreenId } from "@/features/triple-agent/model/screen";
import { GameHeader } from "./game-header";

export function GameShell({
  children,
  screen,
  setScreen,
  session,
  connectionState = "closed",
  reconnecting = false,
  onHome,
}: {
  children: ReactNode;
  screen: ScreenId;
  setScreen: (screen: ScreenId) => void;
  session?: unknown;
  connectionState?: "connecting" | "open" | "closed";
  reconnecting?: boolean;
  onHome?: () => void;
}) {
  const liveSession = Boolean(session);
  const offline = liveSession && connectionState !== "open";

  return (
    <main className="ta-viewport">
      <section className="ta-device">
        <GameHeader
            screen={screen}
            liveSession={liveSession}
            goHome={onHome ?? (() => setScreen("title"))}/>
        {offline ? <p className="ta-connection-banner" role="status">{reconnecting ? "Reconnecting to the room…" : connectionState === "connecting" ? "Connecting to the room…" : "Connection lost: your actions will not reach the room"}</p> : null}
        <div className="ta-stage"><div className="ta-stage-inner">{children}</div></div>
      </section>
    </main>
  );
}
