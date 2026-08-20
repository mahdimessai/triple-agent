import type { ReactNode } from "react";
import type { ClientCommand, Faction, RoomProjection } from "../protocol";
import type { PendingCommand } from "../use-room";
import { getRole } from "../roles";
import { ArtStamp, InkButton, PaperTitle } from "../ui";
import type { ArtName } from "../assets";

export type ResultsScreenProps = {
  projection: RoomProjection;
  pending: PendingCommand | null;
  onSend(command: ClientCommand): void;
};

function factionClass(faction: Faction | undefined): string {
  if (faction === "VIRUS") return "text-ta-red";
  if (faction === "SERVICE") return "text-[#1d5b79]";
  return "text-black";
}

function ContinueButton({ projection, pending, onSend, label = "Continue" }: ResultsScreenProps & { label?: string }) {
  const busy = pending?.kind === "results.continue";
  if (!projection.private.can_submit) {
    return <p className="ta-condensed py-2 text-center text-xs tracking-[0.16em] text-white/80">WAITING FOR THE ROOM TO CONTINUE</p>;
  }
  return (
    <InkButton variant="orange" className="w-full" onClick={() => onSend({ kind: "results.continue" })} disabled={Boolean(pending && !busy)} busy={busy} busyLabel="Continuing…">
      {label}
    </InkButton>
  );
}

function RevealCard({ eyebrow, title, children, artName }: { eyebrow: string; title: string; children?: ReactNode; artName?: ArtName }) {
  return (
    <div className="ta-paper p-5 text-center">
      {artName ? <ArtStamp artName={artName} alt="" className="mx-auto h-28 w-auto object-contain" /> : null}
      <p className="ta-condensed mt-2 text-xs tracking-[0.2em] text-black/60">{eyebrow}</p>
      <h3 className="ta-display mt-1 text-3xl sm:text-4xl">{title}</h3>
      {children}
    </div>
  );
}

function VoteTotals({ projection }: { projection: RoomProjection }) {
  const totals = projection.public.vote_totals ?? {};
  return (
    <div className="ta-paper p-4">
      <div className="flex items-center justify-between border-b-2 border-black/20 pb-3">
        <span className="ta-condensed text-xs tracking-[0.16em]">ACCUSATION TOTALS</span>
        <span className="ta-condensed text-xs tracking-[0.16em]">VOTES</span>
      </div>
      <div className="mt-3 grid gap-2">
        {projection.public.players.map((player) => (
          <div className="flex items-center justify-between gap-3 border-t-2 border-black/15 pt-2" key={player.id}>
            <span className="ta-sans text-lg">{player.name}</span>
            <span className="ta-display text-2xl">{totals[player.id] ?? 0}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function Leaderboard({ projection }: { projection: RoomProjection }) {
  const entries = projection.public.leaderboard ?? [];
  return (
    <div className="ta-paper p-4">
      <div className="flex items-center justify-between border-b-2 border-black/20 pb-3">
        <span className="ta-condensed text-xs tracking-[0.16em]">PARTICIPATING PLAYERS</span>
        <span className="ta-condensed text-xs tracking-[0.16em]">ORGANIZATION</span>
      </div>
      <div className="mt-3 grid gap-2">
        {entries.map((player) => {
          const specialRole = getRole(player.role);
          const defectorArt = player.defection === "BLUE_DEFECTOR" ? "defectorBlue" : player.defection === "RED_DEFECTOR" ? "defectorRed" : null;
          return (
            <div className="flex items-center justify-between gap-3 border-t-2 border-black/15 pt-2" key={player.player_id}>
              <div className="flex min-w-0 items-center gap-3">
                <ArtStamp artName={player.faction === "VIRUS" ? "virusLogo" : "serviceLogo"} alt="" className="h-8 w-8 shrink-0 object-contain" />
                <div className="min-w-0">
                  <span className="ta-sans truncate text-lg">{player.name}</span>
                  {player.player_id === projection.public.imprisoned_player_id ? <span className="ta-condensed ml-2 rounded bg-black px-1.5 py-0.5 text-[0.65rem] tracking-[0.12em] text-white">IMPRISONED</span> : null}
                </div>
              </div>
              <div className="flex shrink-0 items-center gap-2">
                <span className={`ta-sans text-xs tracking-[0.12em] ${factionClass(player.faction)}`}>{player.faction}</span>
                {specialRole ? <ArtStamp artName={specialRole.artName} alt={specialRole.name} className="h-7 w-auto object-contain" /> : null}
                {defectorArt ? <ArtStamp artName={defectorArt} alt="Defector" className="h-7 w-auto object-contain" /> : null}
                <span className="ta-condensed text-xs text-black/50">· {player.votes} {player.votes === 1 ? "VOTE" : "VOTES"}</span>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function imprisonedName(projection: RoomProjection): string {
  const id = projection.public.imprisoned_player_id;
  if (!id) return "NO ONE";
  return projection.public.players.find((player) => player.id === id)?.name ?? "UNKNOWN AGENT";
}

export function ResultsScreen(props: ResultsScreenProps) {
  const { projection, pending, onSend } = props;
  const phase = projection.public.phase;
  const imprisoned = imprisonedName(projection);
  const faction = projection.public.revealed_faction;
  const winner = projection.public.winner;

  let content: ReactNode;
  switch (phase) {
    case "RESULTS_INTRO":
      content = (
        <>
          <RevealCard eyebrow="ACCUSATIONS CLOSED" title="ROUND RESULTS" artName="results">
            <p className="ta-sans mt-2 text-sm text-black/65">{projection.public.activity ?? "The room's votes are locked. The result will now be declassified."}</p>
          </RevealCard>
          <ContinueButton {...props} label="Reveal votes" />
        </>
      );
      break;
    case "VOTE_RESULTS":
      content = <><VoteTotals projection={projection} /><ContinueButton {...props} label="Reveal imprisonment" /></>;
      break;
    case "IMPRISONMENT_REVEAL":
      content = (
        <>
          <RevealCard eyebrow="ACCUSATION OUTCOME" title={projection.public.imprisoned_player_id ? `${imprisoned} WAS IMPRISONED` : "NO ONE WAS IMPRISONED"} artName="imprisoned">
            <p className="ta-sans mt-2 text-sm text-black/65">{projection.public.imprisoned_player_id ? `The room chose ${imprisoned}. Their true agency is still classified.` : "The vote did not produce a prisoner."}</p>
          </RevealCard>
          <ContinueButton {...props} label="Declassify agency" />
        </>
      );
      break;
    case "AGENCY_REVEAL":
      content = (
        <>
          <RevealCard eyebrow="TRUE AGENCY" title={faction ?? "CLASSIFIED"}>
            {faction === "VIRUS" || faction === "SERVICE" ? <ArtStamp artName={faction === "VIRUS" ? "virusLogo" : "serviceLogo"} alt={`${faction} agency`} className="mx-auto mt-4 h-32 w-auto object-contain" /> : null}
            <p className={`ta-display mt-2 text-3xl ${factionClass(faction)}`}>{faction ?? "UNKNOWN"}</p>
          </RevealCard>
          <ContinueButton {...props} label="Reveal outcome" />
        </>
      );
      break;
    case "OUTCOME_REVEAL":
      content = (
        <>
          <RevealCard eyebrow="FINAL OUTCOME" title={winner === "VIRUS" ? "VIRUS WINS" : winner === "SERVICE" ? "THE SERVICE WINS" : "DRAW"} artName="results">
            <p className="ta-sans mt-2 text-sm text-black/65">{projection.public.activity ?? "The server has resolved the match."}</p>
          </RevealCard>
          <ContinueButton {...props} label="Show leaderboard" />
        </>
      );
      break;
    case "LEADERBOARD":
      content = <><Leaderboard projection={projection} /><ContinueButton {...props} label="Finish round" /></>;
      break;
    case "OUT_OF_LOOP":
      content = (
        <>
          <RevealCard eyebrow="ROUND STATUS" title="OUT OF LOOP" artName="results"><p className="ta-sans mt-2 text-sm text-black/65">{projection.public.activity ?? "This player is no longer participating in the current reveal sequence."}</p></RevealCard>
          <ContinueButton {...props} />
        </>
      );
      break;
    case "END": {
      const rematchBusy = pending?.kind === "match.rematch";
      const isHost = projection.public.host_id === projection.private.player_id;
      content = (
        <>
          <RevealCard eyebrow="MATCH COMPLETE" title={winner === "VIRUS" ? "VIRUS WINS" : winner === "SERVICE" ? "THE SERVICE WINS" : "ROUND COMPLETE"} artName="playAgain" />
          {projection.public.leaderboard?.length ? <Leaderboard projection={projection} /> : null}
          {isHost ? (
            <InkButton variant="orange" className="w-full" onClick={() => onSend({ kind: "match.rematch" })} disabled={Boolean(pending && !rematchBusy)} busy={rematchBusy} busyLabel="Starting match…">Play again</InkButton>
          ) : <p className="ta-condensed py-2 text-center text-xs tracking-[0.16em] text-white/80">WAITING FOR HOST TO START NEXT MATCH</p>}
        </>
      );
      break;
    }
    default:
      content = null;
  }

  return <div className="ta-rise ta-screen"><PaperTitle>Round results</PaperTitle><div className="grid gap-4">{content}</div></div>;
}
