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
  onVoteKick?(targetId: string): void;
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
  onVoteKick,
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
        {projection.public.vote_kicks?.map((kick) => {
          if (kick.target_id === projection.private.player_id) return null;
          return (
            <aside
              key={kick.target_id}
              className="flex flex-wrap items-center justify-between gap-2 border-b-4 border-black bg-ta-gold px-3 py-2 text-ta-ink ta-condensed text-xs sm:text-sm tracking-wider"
              role="status"
              aria-live="polite"
            >
              <div className="flex items-center gap-2 font-bold uppercase">
                <span aria-hidden="true">⚠️</span>
                <span>
                  AGENT {kick.target_name} IS OFFLINE{" "}
                  <span className="opacity-80">
                    ({kick.votes}/{kick.required} votes to expel)
                  </span>
                </span>
              </div>
              <div>
                {kick.has_voted ? (
                  <span className="inline-block border-2 border-black/40 bg-black/10 px-2 py-1 text-[0.7rem] uppercase font-bold tracking-widest text-ta-ink/70">
                    [VOTED (WAITING FOR OTHERS)]
                  </span>
                ) : (
                  <button
                    type="button"
                    className="inline-block border-2 border-black bg-ta-red px-2 py-1 text-[0.7rem] uppercase font-bold tracking-widest text-ta-paper hover:bg-black hover:text-white transition-colors cursor-pointer"
                    onClick={() => onVoteKick?.(kick.target_id)}
                  >
                    [EXPEL INACTIVE AGENT]
                  </button>
                )}
              </div>
            </aside>
          );
        })}
        <div className="ta-stage"><div className="ta-stage-inner">{children}</div></div>
      </section>
    </main>
  );
}
