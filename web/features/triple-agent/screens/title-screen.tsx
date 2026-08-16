"use client";

import { useEffect, useRef } from "react";
import { ArtStamp } from "@/components/ui/art-stamp";
import { InkButton } from "@/components/ui/ink-button";

export function TitleScreen({ playerName, setPlayerName, roomCode, setRoomCode, joining, roomAction, onStart, onJoin, onOpenJoin, onCancelJoin, error }: { playerName: string; setPlayerName: (value: string) => void; roomCode: string; setRoomCode: (value: string) => void; joining: boolean; roomAction?: "creating" | "joining"; onStart: () => void; onJoin: () => void; onOpenJoin: () => void; onCancelJoin: () => void; error?: string }) {
  const roomCodeRef = useRef<HTMLInputElement | null>(null);
  const hasPlayerName = playerName.trim().length > 0;
  const creating = roomAction === "creating";
  const joiningRoom = roomAction === "joining";
  // Revealing the field is the whole interaction, so it should take the caret
  // too, otherwise the player hunts for the input they just asked for.
  useEffect(() => { if (joining) roomCodeRef.current?.focus(); }, [joining]);
  return (
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
        <label className="sr-only" htmlFor="player-name">Name</label>
        <input id="player-name" className="ta-name-input w-full" value={playerName} onChange={(event) => setPlayerName(event.target.value.toUpperCase())} placeholder="NAME" autoComplete="nickname" maxLength={24} required />
        {error ? <p className="ta-condensed text-sm text-white">{error}</p> : null}
        <div className="flex w-full flex-col gap-3">
          <InkButton variant="orange" onClick={onStart} disabled={!hasPlayerName} loading={creating} loadingLabel="Creating room…">Start a room</InkButton>
          <InkButton onClick={onOpenJoin} disabled={Boolean(roomAction)}>Join a room</InkButton>
        </div>
      </div>
      {joining ? (
        <div className="ta-join-backdrop" role="presentation" onClick={joiningRoom ? undefined : onCancelJoin}>
          <div className="ta-paper ta-join-dialog" role="dialog" aria-modal="true" aria-labelledby="join-room-title" onClick={(event) => event.stopPropagation()}>
            <p className="ta-condensed text-xs tracking-[0.18em] text-black/60">ROOM CODE</p>
            <h2 id="join-room-title" className="ta-display mt-1 text-3xl">Join a room</h2>
            <label className="ta-condensed mt-5 block text-xs tracking-[0.18em] text-black/60" htmlFor="join-room-code">ENTER 6-CHARACTER CODE</label>
            <input id="join-room-code" ref={roomCodeRef} className="ta-text-input mt-1 w-full border-0 border-b-4 border-black bg-transparent px-1 py-2 text-2xl uppercase tracking-[0.18em]" value={roomCode} onChange={(event) => setRoomCode(event.target.value.toUpperCase().replace(/[^A-Z0-9]/g, "").slice(0, 6))} onKeyDown={(event) => { if (event.key === "Enter" && roomCode.length === 6 && !joiningRoom) onJoin(); if (event.key === "Escape" && !joiningRoom) onCancelJoin(); }} placeholder="XZ04NW" autoComplete="off" maxLength={6} />
            <div className="mt-5 flex gap-3">
              <InkButton variant="orange" className="flex-1" onClick={onJoin} disabled={roomCode.length !== 6 || !hasPlayerName} loading={joiningRoom} loadingLabel="Joining room…">Join room</InkButton>
              <InkButton className="flex-1" onClick={onCancelJoin} disabled={joiningRoom}>Cancel</InkButton>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
