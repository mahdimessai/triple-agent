import type { ClientCommand, RoomProjection } from "../protocol";
import type { PendingCommand } from "../use-room";
import { getRole } from "../roles";
import { ArtStamp, InkButton, PaperTitle } from "../ui";

export type RoleScreenProps = {
  projection: RoomProjection;
  pending: PendingCommand | null;
  onSend(command: ClientCommand): void;
};

type Teammate = { id: string; name: string };

function VirusRoster({ roster, teamSize }: { roster: Teammate[]; teamSize: number }) {
  const expectedOthers = Math.max(teamSize - 1, 0);
  const shown = roster.length;
  const discrepancy = shown > expectedOthers ? "more" : shown < expectedOthers ? "fewer" : "none";

  return (
    <div className="ta-paper mt-3 p-4">
      <p className="ta-condensed text-xs tracking-[0.2em] text-black/60">THE VIRUS NETWORK</p>
      <p className="ta-sans mt-1 text-base leading-snug">
        {expectedOthers === 0 ? "You are the only VIRUS agent on record." : `There ${expectedOthers === 1 ? "is 1 other VIRUS agent" : `are ${expectedOthers} other VIRUS agents`} working with you.`}
      </p>
      {shown === 0 ? <p className="ta-sans mt-2 text-base leading-snug text-black/70">You do not know their identity.</p> : (
        <ul className="mt-2 grid gap-1">{roster.map((teammate) => <li className="ta-display text-2xl leading-tight" key={teammate.id}>{teammate.name}</li>)}</ul>
      )}
      {discrepancy === "none" ? null : (
        <p className="ta-sans mt-3 text-base leading-snug text-ta-red">
          {discrepancy === "more" ? "Wait... there are more names on this list than there should be." : "Wait... there are fewer names on this list than there should be."}
        </p>
      )}
    </div>
  );
}

export function RoleScreen({ projection, pending, onSend }: RoleScreenProps) {
  const personal = projection.private;
  const faction = personal.faction ?? personal.initial_faction;
  const isVirus = faction === "VIRUS";
  const role = getRole(personal.role);
  const roleName = personal.role_name ?? role?.name;
  const roleDescription = personal.role_description ?? role?.description;
  const roleEffect = personal.role_effect ?? role?.effect;
  const showRoster = Boolean(personal.virus_roster) || (isVirus && (personal.virus_team_size ?? 0) > 0);
  const waitingOn = projection.public.pending_role_acks ?? 0;
  const busy = pending?.kind === "role.acknowledge";

  return (
    <div className="ta-rise ta-screen">
      <PaperTitle>Private role reveal</PaperTitle>
      <div className="ta-paper overflow-hidden p-5 text-center">
        <p className="ta-condensed text-xs tracking-[0.2em] text-black/60">YOUR ASSIGNMENT</p>
        <ArtStamp artName={isVirus ? "virusLogo" : "serviceLogo"} alt={`${isVirus ? "VIRUS" : "SERVICE"} agency`} className="mx-auto mt-4 h-44 w-auto object-contain" />
        <h3 className={`ta-display mt-2 text-4xl ${isVirus ? "text-ta-red" : "text-[#1d5b79]"}`}>{isVirus ? "VIRUS" : "SERVICE"}</h3>
        <p className="ta-sans mx-auto mt-3 max-w-sm text-base leading-snug">
          {isVirus ? "Stay hidden. Get the players to imprison one of their own." : "Find and imprison a VIRUS player before the network disappears."} The server has delivered this role only to you.
        </p>
      </div>

      {roleName ? (
        <div className="ta-paper mt-3 p-4">
          <div className="flex items-start justify-between gap-3">
            <div>
              <p className="ta-condensed text-xs tracking-[0.2em] text-black/60">SPECIAL ROLE</p>
              <h4 className="ta-display mt-1 text-3xl leading-none">{roleName}</h4>
              {roleDescription ? <p className="ta-sans mt-2 text-base leading-snug">{roleDescription}</p> : null}
              {roleEffect ? <p className="ta-sans mt-1 text-base leading-snug text-black/70">{roleEffect}</p> : null}
            </div>
            {role ? <ArtStamp artName={role.artName} alt="" className="h-20 w-auto shrink-0 object-contain" /> : null}
          </div>
        </div>
      ) : null}

      {showRoster ? <VirusRoster roster={personal.virus_roster ?? []} teamSize={personal.virus_team_size ?? 0} /> : null}

      <div className="mt-3 flex items-center justify-between gap-3">
        <span className="ta-condensed text-xs tracking-[0.16em]">PRIVATE ROLE</span>
        <InkButton
          onClick={() => onSend({ kind: "role.acknowledge" })}
          disabled={!personal.can_submit || Boolean(pending && !busy)}
          busy={busy}
          busyLabel="Saving…"
        >
          {personal.can_submit ? "I understand" : waitingOn > 0 ? `Waiting for ${waitingOn}` : "Waiting for players"}
        </InkButton>
      </div>
    </div>
  );
}
