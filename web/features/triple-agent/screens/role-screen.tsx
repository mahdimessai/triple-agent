import { ArtStamp } from "@/components/ui/art-stamp";
import { InkButton } from "@/components/ui/ink-button";
import { PaperTitle } from "@/components/ui/paper-title";
import { getRole } from "@/components/triple-agent/role-catalog";

type Teammate = { id: string; name: string };

/**
 * The VIRUS side is told how many agents started on VIRUS and shown the roster
 * of names. Those two numbers normally agree. A Rogue Agent is kept off the
 * roster and a Triple Agent is planted on it, so when roles are in play the
 * count and the list can disagree, and that gap is the only tell either role
 * gives off. The screen states both plainly and lets the reader draw the
 * conclusion.
 */
function VirusRoster({ roster, teamSize }: { roster: Teammate[]; teamSize: number }) {
  const expectedOthers = Math.max(teamSize - 1, 0);
  const shown = roster.length;
  const discrepancy = shown > expectedOthers ? "more" : shown < expectedOthers ? "fewer" : "none";

  return (
    <div className="ta-paper mt-3 p-4">
      <p className="ta-condensed text-xs tracking-[0.2em] text-black/60">THE VIRUS NETWORK</p>
      <p className="ta-condensed mt-1 text-base leading-tight">
        {expectedOthers === 0
          ? "You are the only VIRUS agent on record."
          : `There ${expectedOthers === 1 ? "is 1 other VIRUS agent" : `are ${expectedOthers} other VIRUS agents`} working with you.`}
      </p>
      {shown === 0 ? (
        <p className="ta-condensed mt-2 text-base leading-tight text-black/70">You do not know their identity.</p>
      ) : (
        <ul className="mt-2 grid gap-1">
          {roster.map((teammate) => (
            <li className="ta-display text-2xl leading-tight" key={teammate.id}>
              {teammate.name}
            </li>
          ))}
        </ul>
      )}
      {discrepancy === "none" ? null : (
        <p className="ta-condensed mt-3 text-base leading-tight text-ta-red">
          {discrepancy === "more"
            ? "Wait... there are more names on this list than there should be."
            : "Wait... there are fewer names on this list than there should be."}
        </p>
      )}
    </div>
  );
}

export function RoleScreen({
  faction,
  roleId,
  roleName,
  roleDescription,
  roleEffect,
  virusRoster,
  virusTeamSize = 0,
  canSubmit = true,
  waitingOn = 0,
  loading = false,
  onNext,
}: {
  faction?: string;
  roleId?: string;
  roleName?: string;
  roleDescription?: string;
  roleEffect?: string;
  virusRoster?: Teammate[];
  virusTeamSize?: number;
  canSubmit?: boolean;
  waitingOn?: number;
  loading?: boolean;
  onNext: () => void;
}) {
  const isVirus = faction === "VIRUS";
  // "Waiting for players" with no number leaves the table guessing whether the
  // game is stuck or someone simply has not looked at their phone yet.
  const waitingLabel = waitingOn > 0 ? `Waiting for ${waitingOn}` : "Waiting for players";
  const role = roleId ? getRole(roleId) : undefined;
  const specialName = roleName ?? role?.name;
  const specialDescription = roleDescription ?? role?.description;
  const specialEffect = roleEffect ?? role?.effect;
  // A Triple Agent is handed the VIRUS roster, so the roster's presence, not
  // the player's real agency, decides whether this block is shown.
  const showRoster = Boolean(virusRoster) || (isVirus && virusTeamSize > 0);

  return (
    <div className="ta-rise ta-screen">
      <PaperTitle>Private role reveal</PaperTitle>
      <div className="ta-paper overflow-hidden p-5 text-center">
        <p className="ta-condensed text-xs tracking-[0.2em] text-black/60">YOUR ASSIGNMENT</p>
        <ArtStamp
          artName={isVirus ? "virusLogo" : "serviceLogo"}
          alt={`${isVirus ? "VIRUS" : "SERVICE"} agency`}
          className="mx-auto mt-4 h-44 w-auto object-contain"
        />
        <h3 className={`ta-display mt-2 text-4xl ${isVirus ? "text-ta-red" : "text-[#1d5b79]"}`}>{isVirus ? "VIRUS" : "SERVICE"}</h3>
        <p className="ta-condensed mx-auto mt-3 max-w-sm text-base leading-tight">
          {isVirus
            ? "Stay hidden. Get the players to imprison one of their own."
            : "Find and imprison a VIRUS player before the network disappears."}{" "}
          The server has delivered this role only to you.
        </p>
      </div>

      {specialName ? (
        <div className="ta-paper mt-3 p-4">
          <div className="flex items-start justify-between gap-3">
            <div>
              <p className="ta-condensed text-xs tracking-[0.2em] text-black/60">SPECIAL ROLE</p>
              <h4 className="ta-display mt-1 text-3xl leading-none">{specialName}</h4>
              {specialDescription ? <p className="ta-condensed mt-2 text-base leading-tight">{specialDescription}</p> : null}
              {specialEffect ? <p className="ta-condensed mt-1 text-base leading-tight text-black/70">{specialEffect}</p> : null}
            </div>
            {role ? <ArtStamp artName={role.artName} alt="" className="h-20 w-auto shrink-0 object-contain" /> : null}
          </div>
        </div>
      ) : null}

      {showRoster ? <VirusRoster roster={virusRoster ?? []} teamSize={virusTeamSize} /> : null}

      <div className="mt-3 flex items-center justify-between gap-3">
        <span className="ta-condensed text-xs tracking-[0.16em]">PRIVATE ROLE</span>
        <InkButton onClick={onNext} disabled={!canSubmit} loading={loading} loadingLabel="Saving…">
          {canSubmit ? "I understand" : waitingLabel}
        </InkButton>
      </div>
    </div>
  );
}
