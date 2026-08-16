import { useState } from "react";
import { ArtStamp } from "@/components/ui/art-stamp";
import { InkButton } from "@/components/ui/ink-button";
import { PaperTitle } from "@/components/ui/paper-title";
import type { RoomProjection } from "@/components/triple-agent/server-client";

export function AccusationScreen({ projection, loading = false, onNext }: { projection?: RoomProjection; loading?: boolean; onNext: (targetId?: string) => void }) {
  const [selected, setSelected] = useState("");
  const liveTargets = projection?.public.players.filter((player) => player.id !== projection.private.player_id) ?? [];
  const targets = liveTargets.length ? liveTargets : ["PLAYER A", "PLAYER B", "PLAYER C", "PLAYER D"].map((name) => ({ id: name, name }));
  const alreadyVoted = projection ? !projection.private.can_submit : false;
  const locked = alreadyVoted || loading;
  const waitingOn = projection?.public.players.filter((player) => !player.vote_submitted).length ?? 0;

  return (
    <div className="ta-rise ta-screen">
      <PaperTitle>Accusation time</PaperTitle>
      <div className="ta-paper p-5 text-center">
        <ArtStamp artName="accusation" alt="Accusation illustration" className="mx-auto h-32 w-auto object-contain" />
        <p className="ta-condensed mt-2 text-base leading-tight">Choose the player you believe should be imprisoned. Your target stays hidden until the reveal.</p>
      </div>
      <div className="grid gap-2">
        {targets.map((target) => (
          <button className="ta-paper ta-choice-row flex items-center justify-between px-4 py-3 text-left transition-transform" aria-pressed={selected === target.id} data-selected={selected === target.id} disabled={locked} key={target.id} onClick={() => setSelected(target.id)} type="button">
            <span className="ta-condensed text-lg">{target.name}</span>
            <span className={`h-4 w-4 rounded-full border-2 border-black ${selected === target.id ? "bg-ta-red" : "bg-transparent"}`} />
          </button>
        ))}
      </div>
      {alreadyVoted ? (
        <div className="ta-paper p-4 text-center">
          <p className="ta-condensed text-xs tracking-[0.16em] text-black/60">YOUR ACCUSATION IS IN</p>
          <p className="ta-display mt-1 text-2xl">SUBMITTED</p>
          <p className="ta-condensed mt-2 text-sm text-black/65">{waitingOn > 0 ? `Waiting for ${waitingOn} more ${waitingOn === 1 ? "player" : "players"} to vote.` : "Waiting for the room to finish voting."}</p>
        </div>
      ) : (
        <div className="flex items-center justify-between gap-3">
          <span className="ta-condensed text-xs tracking-[0.16em]">TARGET HIDDEN FROM ROOM</span>
          <InkButton variant="orange" disabled={!selected} loading={loading} loadingLabel="Submitting vote…" onClick={() => onNext(selected)}>Submit vote</InkButton>
        </div>
      )}
    </div>
  );
}
