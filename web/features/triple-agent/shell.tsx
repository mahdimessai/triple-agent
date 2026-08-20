"use client";

import type { ReactNode } from "react";
import type { RoomProjection } from "./protocol";
import type { RoomStatus } from "./use-room";
import { ThemeMusic } from "./music";
import { ArtStamp } from "./ui";

export type GameShellProps = {
  projection: RoomProjection;
  status: RoomStatus;
  error: string | null;
  settingsOpen: boolean;
  onToggleSettings(): void;
  onLeave(): void;
  onHome?: () => void;
  children: ReactNode;
};

function connectionText(status: RoomStatus): string | null {
  switch (status) {
    case "connecting": return "Connecting to the room…";
    case "reconnecting": return "Reconnecting to the room…";
    case "leaving": return "Leaving the room…";
    case "idle": return "Connection lost: your actions will not reach the room";
    case "online": return null;
  }
}

export function GameShell({
  projection,
  status,
  error,
  settingsOpen,
  onToggleSettings,
  onLeave,
  children,
}: GameShellProps) {
  const connection = connectionText(status);
  const phase = projection.public.phase;
  const isPreGame = phase === "LOBBY" || settingsOpen;
  /* The lobby already shows the full settings panel in its own column, so a gear
     that swapped the lobby out for a second copy of that panel only made the
     roster and the start button disappear. It belongs to live phases only. */
  const showSettingsTab = phase !== "LOBBY";

  return (
    <main className="ta-viewport">
      <section className="ta-device">
        <header className="ta-header border-b-4 border-black bg-ta-orange-deep px-3 py-3 text-ta-paper lg:px-5">
          <div className="ta-header-inner">
            <div className="flex min-w-0 items-center gap-3">
              {isPreGame ? (
                <h1
                  className="ta-header-brand"
                  aria-label="Return to main menu"
                >
                  Triple Agent
                </h1>
              ) : (
                <div className="min-w-0">
                  <p className="ta-display truncate text-2xl">Triple Agent</p>
                  <p className="ta-condensed text-[0.62rem] tracking-[0.16em] text-white/70">
                    ROOM {projection.public.room_id}
                  </p>
                </div>
              )}
            </div>
            <nav className="flex items-center gap-2" aria-label="Live room controls">
              <ThemeMusic />
              {showSettingsTab ? (
              <button
                className="ta-tab"
                aria-label={settingsOpen ? "Close room settings" : "Room settings"}
                aria-expanded={settingsOpen}
                data-active={settingsOpen}
                onClick={onToggleSettings}
                type="button"
              >
                <ArtStamp artName="settings" alt="" className="h-5 w-8 object-contain" />
              </button>
              ) : null}
              {phase === "LOBBY" ? (
                <button
                  className="ta-tab px-2 ta-condensed text-xs"
                  onClick={() => {
                    if (window.confirm("Leave this lobby? Your seat will be given up.")) onLeave();
                  }}
                  disabled={status === "leaving"}
                  type="button"
                >
                  {status === "leaving" ? "LEAVING…" : "LEAVE"}
                </button>
              ) : null}
            </nav>
          </div>
        </header>
        {connection ? <p className="ta-connection-banner" role="status">{connection}</p> : null}
        {error ? <p className="ta-connection-banner" role="alert">{error}</p> : null}
        <div className="ta-stage"><div className="ta-stage-inner">{children}</div></div>
      </section>
    </main>
  );
}
