"use client";

import { useEffect, useRef } from "react";
import type { RoomNotice } from "../use-room";
import { ArtStamp, InkButton } from "../ui";

export type EntryMode = "create" | "join";

export type RoomInvite = { code: string; host?: string };

export type TitleScreenProps = {
  mode: EntryMode;
  playerName: string;
  joinCode: string;
  busy: boolean;
  error: string | null;
  notice: RoomNotice | null;
  invite?: RoomInvite | null;
  onDismissInvite?(): void;
  onModeChange(mode: EntryMode): void;
  onPlayerNameChange(value: string): void;
  onJoinCodeChange(value: string): void;
  onCreate(): void;
  onJoin(): void;
  onDismissNotice(): void;
};

function normalizeJoinCode(value: string): string {
  return value.toUpperCase().replace(/[^A-Z0-9]/g, "").slice(0, 6);
}

export function TitleScreen({
  mode,
  playerName,
  joinCode,
  busy,
  error,
  notice,
  invite = null,
  onDismissInvite,
  onModeChange,
  onPlayerNameChange,
  onJoinCodeChange,
  onCreate,
  onJoin,
  onDismissNotice,
}: TitleScreenProps) {
  const codeRef = useRef<HTMLInputElement | null>(null);
  const hasName = playerName.trim().length > 0;
  const joining = mode === "join" && !invite;
  /* A greyed-out button with no stated reason reads as broken, so each entry
     action says what it is still waiting for. */
  const createBlocked = hasName ? undefined : "Enter your name first";
  const joinBlocked = !hasName
    ? "Enter your name first"
    : joinCode.length !== 6
    ? "Enter the 6-character code"
    : undefined;

  useEffect(() => {
    if (joining) codeRef.current?.focus();
  }, [joining]);

  return (
    <main className="ta-viewport">
      <section className="ta-device">
        <div className="ta-stage"><div className="ta-stage-inner">
          <div className="ta-rise ta-screen ta-title-screen items-center text-center">
            <div className="ta-title-art flex w-full flex-col items-center gap-5">
              <p className="ta-paper ta-display ta-angle-right w-full px-4 py-3 text-[clamp(1.8rem,4vw,3.2rem)]">Triple Agent!</p>
              <ArtStamp artName="logo" alt="Tasty Rook logo" className="ta-title-logo h-28 w-auto object-contain sm:h-32" priority />
              <div className="relative flex w-full items-end justify-center">
                <ArtStamp artName="twoMen" alt="Two agents" className="ta-title-figures h-32 w-auto object-contain opacity-90 sm:h-40" priority />
                <div className="absolute -bottom-2 left-1/2 h-2 w-44 -translate-x-1/2 rounded-full bg-black/50 blur-xs" />
              </div>
            </div>

            <div className="ta-title-actions flex w-full flex-col gap-4 text-left">
              {invite ? (
                /* Arriving on an invite link: the code is already known, so the only
                   thing left to ask for is a nickname. */
                <div className="ta-paper px-4 py-4">
                  <p className="ta-condensed text-[0.65rem] tracking-[0.2em] text-black/60">YOU HAVE BEEN INVITED</p>
                  <p className="ta-display mt-1 text-2xl leading-none">
                    {invite.host ? `${invite.host}'s game` : `Room ${invite.code}`}
                  </p>
                  <label className="ta-condensed mt-4 block text-xs tracking-[0.18em] text-black/60" htmlFor="player-name">
                    NICKNAME
                  </label>
                  <input
                    id="player-name"
                    className="ta-text-input mt-1 w-full border-0 border-b-4 border-black bg-transparent px-1 py-2 text-2xl uppercase tracking-[0.1em]"
                    value={playerName}
                    onChange={(event) => onPlayerNameChange(event.target.value.toUpperCase())}
                    placeholder="YOUR NAME"
                    autoComplete="nickname"
                    maxLength={24}
                    disabled={busy}
                    required
                  />
                  {error ? (
                    <p className="ta-sans mt-3 border-l-4 border-ta-red px-2.5 py-1.5 text-sm leading-snug text-ta-red" role="alert">
                      {error}
                    </p>
                  ) : null}
                  <InkButton
                    variant="orange"
                    className="mt-4 w-full px-3 text-center leading-tight"
                    onClick={onJoin}
                    disabled={!hasName || busy}
                    busy={busy}
                    busyLabel="Joining room…"
                  >
                    {hasName ? "Join game" : "Enter your name first"}
                  </InkButton>
                  <button
                    type="button"
                    className="ta-condensed mt-3 w-full text-center text-xs tracking-[0.14em] text-black/55 underline"
                    onClick={onDismissInvite}
                    disabled={busy}
                  >
                    NOT YOU? START OVER
                  </button>
                </div>
              ) : (
                <>
                  <label className="sr-only" htmlFor="player-name">Name</label>
                  <input
                    id="player-name"
                    className="ta-name-input w-full"
                    value={playerName}
                    onChange={(event) => onPlayerNameChange(event.target.value.toUpperCase())}
                    placeholder="NAME"
                    autoComplete="nickname"
                    maxLength={24}
                    disabled={busy}
                    required
                  />
                  {error && !joining ? <p className="ta-sans text-sm text-white" role="alert">{error}</p> : null}
                  <div className="flex w-full flex-col gap-3">
                    <InkButton variant="orange" className="px-3 text-center leading-tight" onClick={onCreate} disabled={!hasName || busy} busy={busy && !joining} busyLabel="Creating room…">{createBlocked ?? "Start a room"}</InkButton>
                    <InkButton onClick={() => onModeChange("join")} disabled={busy}>Join a room</InkButton>
                  </div>
                </>
              )}
            </div>

            {joining ? (
              <div className="ta-join-backdrop" role="presentation" onClick={busy ? undefined : () => onModeChange("create")}>
                <div className="ta-paper ta-join-dialog" role="dialog" aria-modal="true" aria-labelledby="join-room-title" onClick={(event) => event.stopPropagation()}>
                  <p className="ta-condensed text-xs tracking-[0.18em] text-black/60">ROOM CODE</p>
                  <h2 id="join-room-title" className="ta-display mt-1 text-3xl">Join a room</h2>
                  <label className="ta-condensed mt-5 block text-xs tracking-[0.18em] text-black/60" htmlFor="join-room-code">ENTER 6-CHARACTER CODE</label>
                  <input
                    id="join-room-code"
                    ref={codeRef}
                    className="ta-text-input mt-1 w-full border-0 border-b-4 border-black bg-transparent px-1 py-2 text-2xl uppercase tracking-[0.18em]"
                    value={joinCode}
                    onChange={(event) => onJoinCodeChange(normalizeJoinCode(event.target.value))}
                    onKeyDown={(event) => {
                      if (event.key === "Enter" && joinCode.length === 6 && hasName && !busy) onJoin();
                      if (event.key === "Escape" && !busy) onModeChange("create");
                    }}
                    placeholder="XZ04NW"
                    autoComplete="off"
                    maxLength={6}
                    disabled={busy}
                  />
                  {error ? (
                    <p className="ta-sans mt-3 border-l-4 border-ta-red bg-ta-red/12 px-2.5 py-1.5 text-sm leading-snug text-ta-red" role="alert">
                      {error}
                    </p>
                  ) : null}
                  <div className="mt-5 flex gap-3">
                    <InkButton variant="orange" className="flex-1 px-3 text-center leading-tight" onClick={onJoin} disabled={joinCode.length !== 6 || !hasName || busy} busy={busy} busyLabel="Joining room…">{joinBlocked ?? "Join room"}</InkButton>
                    <InkButton className="flex-1" onClick={() => onModeChange("create")} disabled={busy}>Cancel</InkButton>
                  </div>
                </div>
              </div>
            ) : null}
          </div>

          {notice ? (
            <div className="ta-join-backdrop" role="presentation" onClick={onDismissNotice}>
              <div className="ta-paper ta-join-dialog text-center" role="dialog" aria-modal="true" aria-labelledby="room-notice-title" onClick={(event) => event.stopPropagation()}>
                <p className="ta-condensed text-xs tracking-[0.2em] text-black/60">NOTICE</p>
                <h2 id="room-notice-title" className="ta-display mt-1 text-3xl text-ta-red">
                  {notice.kind === "kicked" ? "Kicked from Lobby" : "Room Session Ended"}
                </h2>
                <p className="ta-condensed mt-4 text-base leading-tight">{notice.message}</p>
                <div className="mt-6"><InkButton variant="orange" className="w-full" onClick={onDismissNotice}>Understood</InkButton></div>
              </div>
            </div>
          ) : null}
        </div></div>
      </section>
    </main>
  );
}
