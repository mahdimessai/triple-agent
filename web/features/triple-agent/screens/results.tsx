import type { ClientCommand, Faction, LeaderboardEntry, RoomProjection } from "../protocol";
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

export function formatHiddenAgenda(entry: LeaderboardEntry): { description: string; statusText: string } | null {
  const isWinner = entry.result === "WINNER";
  const statusOutcome = isWinner ? "SUCCESS" : "FAILED";
  const resultLabel = entry.result;

  if (entry.objective_kind === "IMPRISON_SELF") {
    return {
      description: "🎯 Scapegoat: Be Imprisoned",
      statusText: `${statusOutcome} -> ${resultLabel}`,
    };
  }
  if (entry.objective_kind === "IMPRISON_TARGET") {
    const target = entry.objective_name || "Target";
    return {
      description: `🎯 Grudge: Imprison ${target}`,
      statusText: `${statusOutcome} -> ${resultLabel}`,
    };
  }
  if (entry.objective_kind === "TARGET_WINS") {
    const target = entry.objective_name || "Target";
    return {
      description: `🎯 Infatuation: ${target} Wins`,
      statusText: `${statusOutcome} -> ${resultLabel}`,
    };
  }
  if (entry.objective_kind === "RED_DEFECTOR" || entry.defection === "RED_DEFECTOR") {
    return {
      description: "🎯 Red Defector: Defect to Service",
      statusText: `${statusOutcome} -> ${resultLabel}`,
    };
  }
  if (entry.defection === "BLUE_DEFECTOR") {
    return {
      description: "🎯 Blue Defector: Defect to Virus",
      statusText: `${statusOutcome} -> ${resultLabel}`,
    };
  }
  return null;
}

function OutcomeBanner({ projection }: { projection: RoomProjection }) {
  const winner = projection.public.winner;
  const imprisonedId = projection.public.imprisoned_player_id;
  const leaderboard = projection.public.leaderboard ?? [];
  const imprisonedEntry = leaderboard.find((e) => e.player_id === imprisonedId);

  const isScapegoatWin =
    (imprisonedEntry?.objective_kind === "IMPRISON_SELF" && imprisonedEntry.result === "WINNER") ||
    (winner === "NONE" && Boolean(imprisonedId));

  let title = "OPERATION CONCLUDED";
  let subtitle = projection.public.activity ?? "The mission has finished.";
  let artName: ArtName = "results";

  if (isScapegoatWin) {
    title = "OPERATION: SCAPEGOAT SUCCEEDED";
    subtitle = "The Scapegoat was imprisoned; both agencies lose.";
    artName = "imprisoned";
  } else if (winner === "SERVICE") {
    title = "THE SERVICE WINS";
    subtitle = "The Service has successfully neutralized the threat.";
    artName = "serviceLogo";
  } else if (winner === "VIRUS") {
    title = "VIRUS TRIUMPHS";
    subtitle = "VIRUS operatives have subverted the mission.";
    artName = "virusLogo";
  }

  return (
    <div className="ta-paper p-5 text-center">
      <ArtStamp artName={artName} alt="" className="mx-auto h-28 w-auto object-contain" />
      <p className="ta-condensed mt-2 text-xs tracking-[0.2em] text-black/60">EXECUTIVE OUTCOME</p>
      <h3 className="ta-display mt-1 text-3xl sm:text-4xl">{title}</h3>
      <p className="ta-sans mt-2 text-sm text-black/70">{subtitle}</p>
    </div>
  );
}

function ImprisonmentDossier({ projection }: { projection: RoomProjection }) {
  const imprisonedId = projection.public.imprisoned_player_id;
  const players = projection.public.players;
  const totals = projection.public.vote_totals ?? {};
  const leaderboard = projection.public.leaderboard ?? [];

  const imprisonedPlayer = players.find((p) => p.id === imprisonedId);
  const imprisonedEntry = leaderboard.find((e) => e.player_id === imprisonedId);
  const faction = imprisonedEntry?.faction ?? projection.public.revealed_faction;

  return (
    <div className="ta-paper p-4">
      <div className="border-b-2 border-black/20 pb-3 text-center">
        <p className="ta-condensed text-xs tracking-[0.2em] text-black/60">IMPRISONMENT DOSSIER</p>
        {imprisonedPlayer ? (
          <div className="mt-2">
            <h4 className="ta-display text-2xl text-ta-red">{imprisonedPlayer.name} WAS IMPRISONED</h4>
            {faction && faction !== "NONE" ? (
              <div className="mt-1 flex items-center justify-center gap-2">
                <span className="ta-condensed text-xs tracking-[0.16em] text-black/60">TRUE AGENCY:</span>
                <span className={`ta-display text-base font-bold ${factionClass(faction)}`}>{faction}</span>
              </div>
            ) : null}
          </div>
        ) : (
          <div className="mt-2">
            <h4 className="ta-display text-xl text-black/80">NO ONE WAS IMPRISONED</h4>
            <p className="ta-sans mt-1 text-xs text-black/60">The vote was tied; no operative was taken into custody.</p>
          </div>
        )}
      </div>

      <div className="mt-3">
        <div className="flex items-center justify-between pb-2">
          <span className="ta-condensed text-xs tracking-[0.16em] text-black/60">ACCUSATION TOTALS</span>
          <span className="ta-condensed text-xs tracking-[0.16em] text-black/60">VOTES</span>
        </div>
        <div className="grid gap-2">
          {players.map((player) => {
            const votes = totals[player.id] ?? 0;
            const isImprisoned = player.id === imprisonedId;
            return (
              <div className="flex items-center justify-between gap-3 border-t border-black/10 pt-2" key={player.id}>
                <div className="flex items-center gap-2">
                  <span className="ta-sans text-base">{player.name}</span>
                  {isImprisoned ? (
                    <span className="ta-condensed rounded bg-black px-1.5 py-0.5 text-[0.65rem] tracking-[0.12em] text-white">
                      IMPRISONED
                    </span>
                  ) : null}
                </div>
                <span className="ta-display text-lg">
                  {votes} {votes === 1 ? "VOTE" : "VOTES"}
                </span>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

function AgentRosterCard({
  entry,
  imprisonedId,
}: {
  entry: LeaderboardEntry;
  imprisonedId?: string;
}) {
  const specialRole = getRole(entry.role);
  const defectorArt =
    entry.defection === "BLUE_DEFECTOR"
      ? "defectorBlue"
      : entry.defection === "RED_DEFECTOR"
        ? "defectorRed"
        : null;

  const agenda = formatHiddenAgenda(entry);
  const isWinner = entry.result === "WINNER";
  const isImprisoned = entry.player_id === imprisonedId;

  return (
    <div className="ta-paper p-3 flex flex-col gap-2">
      <div className="flex items-center justify-between gap-3">
        <div className="flex min-w-0 items-center gap-3">
          {defectorArt ? (
            <ArtStamp artName={defectorArt} alt="Defector" className="h-8 w-auto shrink-0 object-contain" />
          ) : (
            <ArtStamp
              artName={entry.faction === "VIRUS" ? "virusLogo" : "serviceLogo"}
              alt=""
              className="h-8 w-8 shrink-0 object-contain"
            />
          )}
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <span className="ta-sans truncate text-lg font-bold">{entry.name}</span>
              {isImprisoned ? (
                <span className="ta-condensed rounded bg-black px-1.5 py-0.5 text-[0.65rem] tracking-[0.12em] text-white">
                  IMPRISONED
                </span>
              ) : null}
            </div>
            <div className="flex items-center gap-2 mt-0.5">
              <span className={`ta-condensed text-xs tracking-[0.12em] ${factionClass(entry.faction)}`}>
                {entry.defection ? (entry.defection === "RED_DEFECTOR" ? "RED DEFECTOR" : "BLUE DEFECTOR") : entry.faction}
              </span>
              {specialRole ? (
                <span className="ta-condensed rounded bg-black/10 px-1.5 py-0.5 text-[0.7rem] text-black/80">
                  {specialRole.name}
                </span>
              ) : null}
            </div>
          </div>
        </div>

        <div className="flex shrink-0 items-center gap-2">
          {specialRole ? (
            <ArtStamp artName={specialRole.artName} alt={specialRole.name} className="h-7 w-auto object-contain" />
          ) : null}
          <span
            className={`ta-condensed rounded px-2.5 py-1 text-xs tracking-wider text-white font-bold ${
              isWinner ? "bg-emerald-800" : "bg-ta-red"
            }`}
          >
            {entry.result}
          </span>
        </div>
      </div>

      {agenda ? (
        <div className="flex items-center justify-between border-t border-black/10 pt-1.5 text-xs">
          <span className="ta-sans font-medium text-black/80">{agenda.description}</span>
          <span className={`ta-condensed font-bold ${isWinner ? "text-emerald-800" : "text-ta-red"}`}>
            {agenda.statusText}
          </span>
        </div>
      ) : null}
    </div>
  );
}

function AgentsRoster({ projection }: { projection: RoomProjection }) {
  const leaderboard = projection.public.leaderboard ?? [];
  const imprisonedId = projection.public.imprisoned_player_id;

  const winners = leaderboard.filter((entry) => entry.result === "WINNER");
  const losers = leaderboard.filter((entry) => entry.result !== "WINNER");

  return (
    <div className="grid gap-4">
      {/* WINNERS Section */}
      <div>
        <h3 className="ta-display text-2xl text-center text-white tracking-widest my-2">WINNERS</h3>
        <div className="grid gap-2">
          {winners.length > 0 ? (
            winners.map((entry) => (
              <AgentRosterCard entry={entry} imprisonedId={imprisonedId} key={entry.player_id} />
            ))
          ) : (
            <div className="ta-paper p-3 text-center text-sm text-black/60">NO WINNERS THIS ROUND</div>
          )}
        </div>
      </div>

      {/* LOSERS Section */}
      <div>
        <h3 className="ta-display text-2xl text-center text-white tracking-widest my-2">LOSERS</h3>
        <div className="grid gap-2">
          {losers.length > 0 ? (
            losers.map((entry) => (
              <AgentRosterCard entry={entry} imprisonedId={imprisonedId} key={entry.player_id} />
            ))
          ) : (
            <div className="ta-paper p-3 text-center text-sm text-black/60">NO LOSERS THIS ROUND</div>
          )}
        </div>
      </div>
    </div>
  );
}

export function ResultsScreen(props: ResultsScreenProps) {
  const { projection, pending, onSend } = props;
  const isHost = projection.public.host_id === projection.private.player_id;
  const rematchBusy = pending?.kind === "match.rematch";

  return (
    <div className="ta-rise ta-screen">
      <PaperTitle>Round results</PaperTitle>
      <div className="grid gap-4">
        {/* 1. Executive Outcome Banner */}
        <OutcomeBanner projection={projection} />

        {/* 2. Imprisonment Dossier */}
        <ImprisonmentDossier projection={projection} />

        {/* 3. Full Agents Roster & Hidden Agendas (WINNERS and LOSERS) */}
        <AgentsRoster projection={projection} />

        {/* 4. Host Rematch button */}
        <div className="pt-2 pb-6">
          <InkButton
            variant="orange"
            className="w-full py-3.5 text-lg"
            onClick={() => onSend({ kind: "match.rematch" })}
            disabled={!isHost || Boolean(pending && !rematchBusy)}
            busy={rematchBusy}
            busyLabel="Starting rematch…"
          >
            {isHost ? "START REMATCH" : "Waiting for host…"}
          </InkButton>
        </div>
      </div>
    </div>
  );
}
