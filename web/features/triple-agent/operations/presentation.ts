import type { OperationDefinition } from "@/components/triple-agent/operation-catalog";
import type { OperationResult, RoomProjection } from "@/features/triple-agent/protocol/types";

/**
 * The server sends the answer an operation bought: a name, an agency, or both.
 * Turn it into the sentence the player is actually owed; the bare `message`
 * only restates the instruction, and `code` is an internal enum.
 */
export function operationResultText(result: OperationResult, projection: RoomProjection): string {
  const nameFor = (id?: string) => id ? projection.public.players.find((player) => player.id === id)?.name ?? "an unknown player" : undefined;
  const target = nameFor(result.target_player_id);
  const other = nameFor(result.other_player_id);
  const named = (result.target_player_ids ?? []).map((id) => nameFor(id)).filter(Boolean) as string[];

  switch (result.code) {
    case "FACTION_REVEALED":
      return target && result.target_faction ? `${target} is ${result.target_faction}.` : result.message;
    case "FACTIONS_EXCHANGED":
      return target ? `You exchanged agencies with ${target}. You are now ${result.your_faction ?? "unchanged"}.` : result.message;
    case "AGENCY_SHARED":
      return other && result.other_faction ? `${other} is ${result.other_faction}.` : result.message;
    case "AGENCY_CONFESSED":
      return target ? `You showed your agency to ${target}.` : result.message;
    case "AGENCY_ASSIGNED":
      return `Your agency is ${result.your_faction ?? "unchanged"}.`;
    case "GRUDGE_TARGET_ASSIGNED":
      return target ? `You win only if ${target} is imprisoned.` : result.message;
    case "INFATUATION_TARGET_ASSIGNED":
      return target ? `You win only if ${target} wins.` : result.message;
    case "DOUBLE_VOTE_ASSIGNED":
      return target ? `${target}'s accusation now counts twice.` : result.message;
    case "EXTRA_SUSPICION_ASSIGNED":
      return target ? `${target} carries one extra accusation vote.` : result.message;
    case "VOTE_SHIELD_ASSIGNED":
      return target ? `${target} is shielded from one vote.` : result.message;
    case "PLAYER_SILENCED":
      return target ? `${target} has been silenced.` : result.message;
    case "ONE_VIRUS_ONE_SERVICE":
      return named.length === 2 ? `Between ${named[0]} and ${named[1]}, one is VIRUS and the other is not.` : result.message;
    case "SAME_FACTION":
      return named.length === 2 ? `${named[0]} and ${named[1]} belong to the same agency.` : result.message;
    case "SAME_INITIAL_AGENCY":
      return named.length === 2 ? `${named[0]} and ${named[1]} started on the same agency.` : result.message;
    case "DIFFERENT_INITIAL_AGENCY":
      return named.length === 2 ? `${named[0]} and ${named[1]} started on different agencies.` : result.message;
    case "JOINED_VIRUS":
      return target ? `${target} is VIRUS. You have joined VIRUS.` : result.message;
    case "TARGET_SERVICE":
      return target ? `${target} is not VIRUS. You remain in your agency.` : result.message;
    case "AT_LEAST_ONE_VIRUS":
      return target ? `Between you and ${target}, at least one is VIRUS.` : result.message;
    case "NO_VIRUS_FOUND":
      return target ? `Neither you nor ${target} is VIRUS.` : result.message;
    case "CURED":
      return "Your agency changed to SERVICE.";
    case "INFECTED":
      return "Your agency changed to VIRUS.";
    case "DEFECTED":
      return `You have defected. Your agency is now ${result.your_faction ?? "changed"}.`;
    case "STAYED":
      return "You chose to stay with your current agency.";
    case "OBJECTIVE_ASSIGNED":
      return "You win only if you are imprisoned.";
    default:
      if (named.length === 2 && result.target_faction) return `Between ${named[0]} and ${named[1]}, one is ${result.target_faction}.`;
      if (target && result.target_faction) return `${target} is ${result.target_faction}.`;
      if (other && result.other_faction) return `${other} is ${result.other_faction}.`;
      return result.message;
  }
}

export const operationBriefs: Record<string, string> = {
  HiddenAgenda: "New orders arrive: a side switch, a new win condition, or intel on one agent.", Swap: "Trade agencies with one other player.", Injection: "Attempt to recruit a target into your agency.", Share: "Show one agency fact to a chosen recipient.", Detector: "Check two players for a possible VIRUS link.", Strain: "Gain a private win condition tied to imprisonment.", Grudge: "You are dealt a target you need to see imprisoned.", Infatuation: "Your win is bound to another player's victory.", Flip: "Your agency may change under a hidden instruction.", HiddenOneRandom: "Receive a private agency tip.", OneRandom: "Receive an anonymous private agency tip.", OneOfTwo: "Learn which of two players is VIRUS.", TwoFriends: "Learn which two players began together.", Undercover: "Inspect a target and possibly join them.", InfoForTwo: "Share a VIRUS check with another player.", ChooseVoteShield: "Give a target a vote penalty or shield.", Defect: "Choose whether to defect from your agency.", Power: "Give one player a double accusation vote.", Vote: "Give one player an extra accusation vote against them.", Confirm: "Confirm your current agency privately.", NegativeVote: "Protect one player from a vote.", Ambassador: "Recovered variant with a server-defined effect.", Brig: "Recovered variant with a server-defined status effect.", EarlyVote: "Recovered variant that changes accusation timing.", Hunter: "Recovered variant that prepares a target search.", LastEvent: "Recovered variant that marks the final event.",
};

/**
 * What the room is told when an operation is dealt. The table needs to know
 * what the recipient is about to learn, that is the information the bluffing
 * is built on, without learning the answer itself.
 */
const roomBriefings: Record<string, { room: (name: string) => string; you: string }> = {
  OneRandom: { room: (name) => `${name}'s source knows the agency of one other player and reveals it to them.`, you: "Your source knows the agency of one other player and reveals it to you." },
  HiddenOneRandom: { room: (name) => `${name} receives a private tip naming one other player's agency.`, you: "You receive a private tip naming one other player's agency." },
  Swap: { room: (name) => `${name} secretly exchanges agencies with one player of their choice.`, you: "You secretly exchange agencies with one player of your choice." },
  Share: { room: (name) => `${name} shows their agency to one player of their choice.`, you: "You show your agency to one player of your choice." },
  Detector: { room: (name) => `${name} checks two players and learns whether either one is VIRUS.`, you: "You check two players and learn whether either one is VIRUS." },
  OneOfTwo: { room: (name) => `${name} learns which of two players is VIRUS.`, you: "You learn which of two players is VIRUS." },
  TwoFriends: { room: (name) => `${name} learns whether two players started on the same side.`, you: "You learn whether two players started on the same side." },
  Undercover: { room: (name) => `${name} investigates one player and may join them.`, you: "You investigate one player and may join them." },
  InfoForTwo: { room: (name) => `${name} and one player of their choice share the same VIRUS check.`, you: "You and one player of your choice share the same VIRUS check." },
  Defect: { room: (name) => `${name} chooses in secret whether to defect from their agency.`, you: "You choose in secret whether to defect from your agency." },
  Confirm: { room: (name) => `${name} privately confirms their own current agency.`, you: "You privately confirm your own current agency." },
  // The five hidden operations are announced under one cover, so the room hears
  // the same sentence whichever one was dealt.
  HiddenAgenda: {
    room: (name) => `${name} gets new orders from up top. They could switch sides, gain a new win condition, or simply learn another agent's agency.`,
    you: "You get new orders from up top. You could switch sides, gain a new win condition, or simply learn another agent's agency.",
  },
  // Kept for the operations workbench and settings previews, where each hidden
  // operation is shown on its own rather than behind the cover.
  Strain: { room: (name) => `${name} receives a private objective tied to being imprisoned.`, you: "You receive a private objective tied to being imprisoned." },
  Grudge: { room: (name) => `${name} is dealt one player they need to see imprisoned.`, you: "You are dealt one player you need to see imprisoned." },
  Infatuation: { room: (name) => `${name} is bound to another player's victory.`, you: "You are bound to another player's victory." },
  Flip: { room: (name) => `${name}'s agency may quietly change.`, you: "Your agency may quietly change." },
  Injection: { room: (name) => `${name} attempts to recruit one player into their agency.`, you: "You attempt to recruit one player into your agency." },
  Power: { room: (name) => `${name} gives one player a double accusation vote.`, you: "You give one player a double accusation vote." },
  Vote: { room: (name) => `${name} puts extra suspicion on one player.`, you: "You put extra suspicion on one player." },
  NegativeVote: { room: (name) => `${name} shields one player from a vote.`, you: "You shield one player from a vote." },
  ChooseVoteShield: { room: (name) => `${name} assigns one player a vote penalty or a shield.`, you: "You assign one player a vote penalty or a shield." },
};

export function roomBriefing(operation: OperationDefinition, recipient: string, isRecipient: boolean, fallback: string) {
  const briefing = roomBriefings[operation.id];
  if (!briefing) return fallback;
  return isRecipient ? briefing.you : briefing.room(recipient);
}

export function operationBrief(operation: OperationDefinition) {
  return operationBriefs[operation.id] ?? operation.publicUpdate;
}
