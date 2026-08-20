"use client";

import { useMemo, useState } from "react";
import type { ClientCommand, RoomProjection } from "../protocol";
import type { PendingCommand } from "../use-room";
import { countDeckOperations } from "../operations";
import { InkButton } from "../ui";
import { SettingsPanel } from "./settings";

const DEFAULT_MIN_PLAYERS = 5;

/* Host controls are glyphs, not worded buttons: at nine seats a "MAKE HOST" and a
   "KICK" label alongside the status badge cannot fit one roster line inside the
   lobby's narrow column without shoving the name down to two characters. */
function HostGlyph() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" className="h-4 w-4" fill="currentColor">
      <path d="M3 7l4.5 3.5L12 4l4.5 6.5L21 7l-1.8 11H4.8L3 7z" />
    </svg>
  );
}

function KickGlyph() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" className="h-4 w-4" fill="none" stroke="currentColor" strokeWidth={3} strokeLinecap="square">
      <path d="M6 6l12 12M18 6L6 18" />
    </svg>
  );
}

function ShareGlyph() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth={2.2} strokeLinecap="square" strokeLinejoin="miter">
      <path d="M12 3.5v11" />
      <path d="M7.5 8 12 3.5 16.5 8" />
      <path d="M5 14.5v6h14v-6" />
    </svg>
  );
}

/* The room code sits right beside its own copy affordance, so the control is
   the clipboard glyph alone; it flips to a tick for the confirmation window. */
function CopyGlyph({ copied }: { copied: boolean }) {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth={2.2} strokeLinecap="square" strokeLinejoin="miter">
      {copied ? (
        <path d="M4 12.5 9.5 18 20 6.5" />
      ) : (
        <>
          <rect x="8.5" y="3.5" width="12" height="15" />
          <path d="M15.5 20.5h-12v-15h2.5" />
        </>
      )}
    </svg>
  );
}

export type LobbyScreenProps = {
  projection: RoomProjection;
  joinCode: string;
  copied: boolean;
  pending: PendingCommand | null;
  onSend(command: ClientCommand): void;
  onCopyRoomCode(): void;
  onShareLink(): void;
  linkShared?: boolean;
  error?: string | null;
};

function disambiguate(players: Array<{ name: string; seat: number }>): string[] {
  const counts = new Map<string, number>();
  for (const player of players) {
    counts.set(player.name, (counts.get(player.name) ?? 0) + 1);
  }
  const seen = new Map<string, number>();
  return players.map((player) => {
    if ((counts.get(player.name) ?? 0) <= 1) return player.name;
    const index = (seen.get(player.name) ?? 0) + 1;
    seen.set(player.name, index);
    return `${player.name} (${index})`;
  });
}

export function LobbyScreen({
  projection,
  joinCode,
  copied,
  pending,
  onSend,
  onCopyRoomCode,
  onShareLink,
  linkShared = false,
  error = null,
}: LobbyScreenProps) {
  const [mobileTab, setMobileTab] = useState<"roster" | "settings">("roster");

  const roster = projection.public.players;
  const selfId = projection.private.player_id;
  const hostId = projection.public.host_id;
  const isHost = hostId === selfId;
  const me = roster.find((player) => player.id === selfId);

  const displayNames = useMemo(() => disambiguate(roster), [roster]);
  const readyCount = useMemo(() => roster.filter((player) => player.ready).length, [roster]);
  const minPlayers = projection.public.settings.min_players ?? DEFAULT_MIN_PLAYERS;
  const missingPlayers = Math.max(0, minPlayers - roster.length);
  const everyoneReady = roster.length >= minPlayers && roster.every((player) => player.ready);
  const canStart = isHost && missingPlayers === 0 && everyoneReady;
  const enabledOpsCount = useMemo(
    () => countDeckOperations(projection.public.settings.enabled_operations ?? []),
    [projection.public.settings.enabled_operations]
  );

  const readyBusy = pending?.kind === "lobby.ready";
  const startBusy = pending?.kind === "match.start";

  /* Why the match cannot start is the start button's own business, so the reason
     becomes its label rather than a caption underneath it. */
  const waitingOnReady = roster.length - readyCount;
  const blockedReason =
    missingPlayers > 0
      ? `${missingPlayers} more ${missingPlayers === 1 ? "player" : "players"} needed`
      : !everyoneReady
      ? `Waiting for ${waitingOnReady} ${waitingOnReady === 1 ? "player" : "players"} to ready up`
      : undefined;

  return (
    <div className="ta-rise ta-screen ta-screen--wide ta-screen--lobby">
      {/* Mobile View Segmented Switch */}
      <div className="flex lg:hidden w-full border-2 border-black bg-ta-paper p-1 gap-1.5 shadow-[3px_3px_0_var(--ta-shadow)]">
        <button
          type="button"
          className={`ta-condensed flex-1 py-2 text-xs tracking-widest uppercase transition-all ${
            mobileTab === "roster"
              ? "bg-ta-ink text-ta-paper border-2 border-black shadow-[2px_2px_0_var(--ta-shadow)]"
              : "bg-transparent text-black/75 hover:bg-black/5"
          }`}
          onClick={() => setMobileTab("roster")}
        >
          ROSTER ({roster.length})
        </button>
        <button
          type="button"
          className={`ta-condensed flex-1 py-2 text-xs tracking-widest uppercase transition-all ${
            mobileTab === "settings"
              ? "bg-ta-ink text-ta-paper border-2 border-black shadow-[2px_2px_0_var(--ta-shadow)]"
              : "bg-transparent text-black/75 hover:bg-black/5"
          }`}
          onClick={() => setMobileTab("settings")}
        >
          SETTINGS ({enabledOpsCount} OPS)
        </button>
      </div>

      <div className="grid w-full gap-5 lg:grid-cols-12 lg:items-start">
        {/* Left Column: Room Info, Player Roster & Actions */}
        <div className={`w-full lg:col-span-4 lg:sticky lg:top-4 ${mobileTab !== "roster" ? "hidden lg:block" : "block"}`}>
          <div className="grid gap-3">
            {/* Room Code Box */}
            <div className="ta-paper px-4 py-3.5">
              <p className="ta-display text-xl leading-none">Invite players</p>
              <p className="ta-condensed mt-2.5 text-[0.65rem] tracking-[0.2em] text-black/60">ROOM CODE</p>
              {/* Code and copy sit on one borderless line: the glyph plus the word is
                  the whole affordance, so nothing needs a box drawn around it. */}
              <div className="flex min-w-0 items-center gap-3">
                <p className="ta-display truncate text-[clamp(1.75rem,7vw,2.75rem)] leading-none tracking-[0.12em]">{joinCode}</p>
                {/* Centred in whatever room is left beside the code, so it never
                    hugs the last character or the card edge. */}
                <span className="flex flex-1 justify-center">
                <button
                  className="ta-copy-code"
                  onClick={onCopyRoomCode}
                  type="button"
                  aria-label={copied ? `Room code ${joinCode} copied` : `Copy room code ${joinCode}`}
                >
                  <CopyGlyph copied={copied} />
                  <span className="ta-condensed text-xs tracking-[0.12em]" role="status">{copied ? "Copied" : "Copy"}</span>
                </button>
                </span>
              </div>
              <button className="ta-share-button mt-3" onClick={onShareLink} type="button">
                <ShareGlyph />
                <span>{linkShared ? "Link copied" : "Share link"}</span>
              </button>
            </div>

            {/* Roster List */}
            <div className="grid gap-2">
              {roster.map((player, index) => {
                const isPlayerHost = player.id === hostId;
                const canManage = isHost && player.id !== selfId;
                // One line, always: the name truncates, the status badge is pinned
                // to the right edge, and nothing reflows into a second row, so every
                // seat keeps the same height.
                return (
                  <div className="ta-paper flex flex-nowrap items-center gap-2 p-3" key={player.id}>
                    <div className="flex min-w-0 items-center gap-2">
                      {/* Your own seat is marked by the colour of the name rather than a
                          badge; the screen-reader text carries what the colour says. */}
                      <span className={`ta-condensed truncate text-lg ${player.id === selfId ? "text-ta-blue" : ""}`}>
                        {player.seat}. {displayNames[index]}
                        {player.id === selfId ? <span className="sr-only"> (you)</span> : null}
                      </span>
                      {isPlayerHost ? (
                        <span className="ta-roster-host shrink-0" role="img" aria-label="Host" title="Host">
                          <HostGlyph />
                        </span>
                      ) : null}
                    </div>
                    <div className="ml-auto flex shrink-0 items-center gap-1.5">
                      {canManage ? (
                        <>
                          <button
                            className="ta-roster-action"
                            onClick={() => onSend({ kind: "lobby.transfer_host", target_id: player.id })}
                            disabled={Boolean(pending)}
                            type="button"
                            aria-label={`Make ${displayNames[index]} the host`}
                            title="Make host"
                          >
                            <HostGlyph />
                          </button>
                          <button
                            className="ta-roster-action ta-roster-action--kick"
                            onClick={() => onSend({ kind: "lobby.kick_player", target_id: player.id })}
                            disabled={Boolean(pending)}
                            type="button"
                            aria-label={`Kick ${displayNames[index]}`}
                            title="Kick from lobby"
                          >
                            <KickGlyph />
                          </button>
                        </>
                      ) : null}
                      <span className="ta-ready-badge" data-connected={player.connected} data-ready={player.ready}>
                        {player.connected ? (player.ready ? "READY" : "NOT READY") : "OFFLINE"}
                      </span>
                    </div>
                  </div>
                );
              })}
              {Array.from({ length: missingPlayers }, (_, index) => (
                <div className="ta-empty-seat" key={`empty-${index}`}>
                  <span className="ta-condensed text-sm tracking-[0.14em]">{roster.length + index + 1}. WAITING FOR A PLAYER</span>
                </div>
              ))}
            </div>

            {/* Room errors belong beside the controls that caused them rather
                than in a banner at the far top of the screen. */}
            {error ? (
              <p className="ta-paper ta-sans border-l-4 border-ta-red px-3 py-2 text-sm leading-snug text-ta-red" role="alert">
                {error}
              </p>
            ) : null}

            {/* Ready Toggle Action */}
            <InkButton
              className="w-full"
              onClick={() => onSend({ kind: "lobby.ready" })}
              disabled={Boolean(pending && !readyBusy)}
              busy={readyBusy}
              busyLabel="Saving…"
            >
              {me?.ready ? "Not ready" : "I'm ready"}
            </InkButton>

            {/* Start Match Action */}
            {isHost ? (
              <InkButton
                variant="orange"
                className="w-full px-3 text-center leading-tight"
                onClick={() => onSend({ kind: "match.start" })}
                disabled={!canStart || Boolean(pending && !startBusy)}
                busy={startBusy}
                busyLabel="Starting match…"
              >
                {blockedReason ?? "Start match"}
              </InkButton>
            ) : (
              <p className="ta-sans text-center text-xs text-white/80">{blockedReason ?? "Waiting for the host to start the match"}</p>
            )}

          </div>
        </div>

        {/* Right Column: Live Settings */}
        <div className={`w-full lg:col-span-8 ${mobileTab !== "settings" ? "hidden lg:block" : "block"}`}>
          <SettingsPanel
            projection={projection}
            pending={pending}
            onSend={onSend}
            showHeader={false}
          />
        </div>
      </div>
    </div>
  );
}
