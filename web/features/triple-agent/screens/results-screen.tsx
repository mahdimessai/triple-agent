import { useEffect, useMemo, useState } from "react";
import { ArtStamp } from "@/components/ui/art-stamp";
import { InkButton } from "@/components/ui/ink-button";
import { PaperTitle } from "@/components/ui/paper-title";
import { getRole } from "@/components/triple-agent/role-catalog";
import type { Faction, RoomProjection } from "@/components/triple-agent/server-client";

export function ResultsScreen({ projection, loading = false, onRestart }: { projection?: RoomProjection; loading?: boolean; onRestart: () => void }) {
  type LeaderboardEntry = NonNullable<RoomProjection["public"]["leaderboard"]>[number];
  const isHost = projection ? projection.public.host_id === projection.private.player_id : true;
  const imprisoned = useMemo(() => {
    return projection
      ? projection.public.players.find((player) => player.id === projection.public.imprisoned_player_id)
      : { id: "p2", name: "PLAYER B" };
  }, [projection]);

  const imprisonedFaction: Faction = useMemo(() => {
    return projection?.public.revealed_faction
      ?? (projection?.public.leaderboard?.find((p) => p.player_id === projection.public.imprisoned_player_id)?.faction ?? "VIRUS");
  }, [projection]);

  const winner: Faction = projection?.public.winner ?? "SERVICE";

  const fallbackRoster = useMemo<LeaderboardEntry[]>(() => [
    { player_id: "p1", name: "PLAYER A", faction: "SERVICE" as Faction, votes: 0, result: "WINNER" },
    { player_id: "p2", name: "PLAYER B", faction: "VIRUS" as Faction, votes: 3, result: "LOSER" },
    { player_id: "p3", name: "PLAYER C", faction: "SERVICE" as Faction, votes: 1, result: "WINNER" },
    { player_id: "p4", name: "PLAYER D", faction: "SERVICE" as Faction, votes: 0, result: "WINNER" },
    { player_id: "p5", name: "PLAYER E", faction: "VIRUS" as Faction, votes: 1, result: "LOSER" },
  ], []);

  const rosterEntries = (projection?.public.leaderboard && projection.public.leaderboard.length > 0)
    ? projection.public.leaderboard
    : fallbackRoster;

  const [flipState, setFlipState] = useState<"flipping" | "revealed">(() => imprisoned ? "flipping" : "revealed");
  const [face, setFace] = useState<"SERVICE" | "VIRUS">("SERVICE");
  const [showRoster, setShowRoster] = useState(() => !imprisoned);

  useEffect(() => {
    if (!imprisoned) {
      return;
    }

    let index = 0;
    // Slower, dramatic interval (650ms) matching the 3D flip animation
    const interval = window.setInterval(() => {
      index += 1;
      setFace(index % 2 === 0 ? "SERVICE" : "VIRUS");
    }, 650);

    const flipTimer = window.setTimeout(() => {
      window.clearInterval(interval);
      if (imprisonedFaction === "VIRUS" || imprisonedFaction === "SERVICE") {
        setFace(imprisonedFaction);
      }
      setFlipState("revealed");
    }, 3900);

    const rosterTimer = window.setTimeout(() => {
      setShowRoster(true);
    }, 5000);

    return () => {
      window.clearInterval(interval);
      window.clearTimeout(flipTimer);
      window.clearTimeout(rosterTimer);
    };
  }, [imprisoned, imprisonedFaction]);

  return (
    <div className="ta-rise ta-screen">
      <PaperTitle>Round results</PaperTitle>

      {/* Imprisonment Outcome Card */}
      <div className="ta-paper p-5 text-center">
        <ArtStamp artName="imprisoned" alt="Imprisoned player" className="mx-auto h-28 w-auto object-contain" />
        <p className="ta-condensed mt-2 text-xs tracking-[0.2em] text-black/60">ACCUSATION OUTCOME</p>
        <h3 className="ta-display mt-1 text-3xl sm:text-4xl">
          {imprisoned ? `${imprisoned.name} WAS IMPRISONED` : "NO ONE WAS IMPRISONED"}
        </h3>
        {imprisoned ? (
          <p className="ta-condensed mt-1 text-sm text-black/65">
            The room voted to imprison {imprisoned.name}. Declassifying their true agency...
          </p>
        ) : (
          <p className="ta-condensed mt-1 text-sm text-black/65">
            The vote was tied. No player was imprisoned this round.
          </p>
        )}
      </div>

      {/* Slow Dramatic Card Flip Revealing the Voted Player's Agency */}
      {imprisoned ? (
        <div className="ta-paper ta-faction-reveal-wrap overflow-hidden p-5 text-center">
          <p className="ta-condensed text-xs tracking-[0.2em] text-black/60">
            {flipState === "flipping" ? "DECLASSIFYING AGENCY FILE..." : `${imprisoned.name.toUpperCase()}'S TRUE AGENCY`}
          </p>
          <div className="ta-faction-card mx-auto mt-4" data-flipping={flipState === "flipping"} data-revealed={flipState === "revealed"}>
            <div className="ta-faction-card-inner">
              <div className={`ta-faction-face ${face === "VIRUS" ? "ta-faction-virus" : "ta-faction-service"}`}>
                <ArtStamp
                  artName={face === "VIRUS" ? "virusLogo" : "serviceLogo"}
                  alt={`${face} agency`}
                  className="mx-auto h-28 w-auto object-contain"
                />
                <p className={`ta-display mt-2 text-3xl ${face === "VIRUS" ? "text-ta-red" : "text-[#1d5b79]"}`}>
                  {flipState === "revealed" ? (imprisonedFaction ?? face) : face}
                </p>
              </div>
            </div>
          </div>
          <p className="ta-condensed mt-4 text-base leading-tight">
            {flipState === "flipping"
              ? `Searching intelligence archives for ${imprisoned.name}...`
              : imprisonedFaction === "VIRUS"
                ? `${imprisoned.name} was a VIRUS operative!`
                : `${imprisoned.name} was an innocent SERVICE agent!`}
          </p>
        </div>
      ) : null}

      {/* Outcome & All Participating Players with Member Org Logos */}
      {showRoster ? (
        <div className="ta-rise grid gap-4">
          <div className="ta-paper p-5 text-center">
            <p className="ta-condensed text-xs tracking-[0.2em] text-black/60">FINAL OUTCOME</p>
            <h2 className={`ta-display mt-1 text-3xl sm:text-4xl ${winner === "VIRUS" ? "text-ta-red" : winner === "SERVICE" ? "text-[#1d5b79]" : "text-black"}`}>
              {winner === "VIRUS" ? "VIRUS WINS" : winner === "SERVICE" ? "THE SERVICE WINS" : "DRAW"}
            </h2>
            <p className="ta-condensed mt-2 text-sm text-black/65">
              {winner === "VIRUS"
                ? "A Service agent was imprisoned or the virus network stayed hidden."
                : winner === "SERVICE"
                  ? "The Service successfully identified and imprisoned a VIRUS operative."
                  : "The match ended in a draw."}
            </p>
          </div>

          <div className="ta-paper p-4">
            <div className="flex items-center justify-between border-b-2 border-black/20 pb-3">
              <span className="ta-condensed text-xs tracking-[0.16em]">PARTICIPATING PLAYERS</span>
              <span className="ta-condensed text-xs tracking-[0.16em]">ORGANIZATION</span>
            </div>
            <div className="mt-3 grid gap-2">
              {rosterEntries.map((player) => {
                const isVirus = player.faction === "VIRUS";
                const isImprisoned = player.player_id === imprisoned?.id || player.name === imprisoned?.name;
                const specialRole = player.role ? getRole(player.role) : undefined;
                const defectorArtName = player.defection === "BLUE_DEFECTOR"
                  ? "defectorBlue"
                  : player.defection === "RED_DEFECTOR"
                    ? "defectorRed"
                    : undefined;
                return (
                  <div className="flex items-center justify-between gap-3 border-t-2 border-black/15 pt-2" key={player.player_id}>
                    <div className="flex min-w-0 items-center gap-3">
                      <ArtStamp
                        artName={isVirus ? "virusLogo" : "serviceLogo"}
                        alt={`${player.faction} logo`}
                        className="h-8 w-8 shrink-0 object-contain"
                      />
                      <div className="min-w-0">
                        <span className="ta-condensed truncate text-lg font-bold">{player.name}</span>
                        {isImprisoned ? (
                          <span className="ta-condensed ml-2 rounded bg-black px-1.5 py-0.5 text-[0.65rem] tracking-[0.12em] text-white">
                            IMPRISONED
                          </span>
                        ) : null}
                      </div>
                    </div>
                    <div className="flex shrink-0 items-center gap-2">
                      <span className={`ta-condensed text-xs font-bold tracking-[0.12em] ${isVirus ? "text-ta-red" : "text-[#1d5b79]"}`}>
                        {player.faction}
                      </span>
                      {specialRole ? (
                        <span
                          aria-label={`${specialRole.name} role`}
                          className="inline-flex h-8 w-7 items-center justify-center border border-black/15 bg-black/5"
                          title={specialRole.name}
                        >
                          <ArtStamp artName={specialRole.artName} alt={`${specialRole.name} role`} className="h-7 w-auto object-contain" />
                        </span>
                      ) : null}
                      {defectorArtName ? (
                        <span
                          aria-label={player.defection === "BLUE_DEFECTOR" ? "Blue defector" : "Red defector"}
                          className="inline-flex h-8 w-7 items-center justify-center border border-black/15 bg-black/5"
                          title={player.defection === "BLUE_DEFECTOR" ? "Blue defector" : "Red defector"}
                        >
                          <ArtStamp artName={defectorArtName} alt={player.defection === "BLUE_DEFECTOR" ? "Blue defector" : "Red defector"} className="h-7 w-auto object-contain" />
                        </span>
                      ) : null}
                      <span className="ta-condensed text-xs text-black/50">
                        · {player.votes} {player.votes === 1 ? "VOTE" : "VOTES"}
                      </span>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>

          <div className="mt-2 flex flex-col gap-2">
            {isHost ? (
              <InkButton variant="orange" className="w-full" onClick={onRestart} loading={loading} loadingLabel="Starting match…">
                Play again
              </InkButton>
            ) : (
              <p className="ta-condensed py-2 text-center text-xs tracking-[0.16em] text-white/80">
                WAITING FOR HOST TO START NEXT MATCH
              </p>
            )}
          </div>
        </div>
      ) : (
        <p className="ta-condensed animate-pulse text-center text-xs tracking-[0.16em] text-white/80">
          REVEALING FINAL PARTICIPANTS...
        </p>
      )}
    </div>
  );
}
