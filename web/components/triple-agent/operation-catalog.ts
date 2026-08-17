import type { ArtName } from "./asset-registry";

export type OperationStatus = "enabled" | "disabled-by-config" | "recovered-only";

export type OperationDefinition = {
  id: string;
  name: string;
  status: OperationStatus;
  artName: ArtName;
  category: "information" | "choice" | "hidden" | "public" | "variant";
  publicUpdate: string;
  privatePrompt: string;
  input: string;
};

export const operationCatalog = [
  {
    // The cover identity the room is shown whenever a hidden operation is
    // dealt. It is never dealt itself and never appears in room settings; the
    // server masks Scapegoat, Grudge, Infatuation, Sleeper Agent and Secret Tip
    // behind it so the table cannot tell which one arrived.
    id: "HiddenAgenda",
    name: "Hidden Agenda",
    status: "enabled",
    artName: "hiddenAgenda",
    category: "hidden",
    publicUpdate: "The active player has new orders from up top. They could switch sides, gain a new win condition, or learn another agent's agency.",
    privatePrompt: "Read your orders in secret.",
    input: "No input; private orders",
  },
  {
    id: "Swap",
    name: "Spy Transfer",
    status: "enabled",
    artName: "swap",
    category: "choice",
    publicUpdate: "The server is waiting for the active player to choose an exchange target.",
    privatePrompt: "Choose one player. You and that player secretly exchange agencies.",
    input: "Choose one other player",
  },
  {
    id: "Injection",
    name: "Recruitment",
    status: "disabled-by-config",
    artName: "recruitment",
    category: "hidden",
    publicUpdate: "A recruitment operation is being resolved privately.",
    privatePrompt: "The server determines whether a target joins the active player's agency.",
    input: "Server-resolved target",
  },
  {
    id: "Share",
    name: "Confession",
    status: "enabled",
    artName: "share",
    category: "choice",
    publicUpdate: "The active player is sharing one private agency fact with a chosen player.",
    privatePrompt: "Choose exactly one other player who may view your agency information.",
    input: "Choose one recipient",
  },
  {
    id: "Detector",
    name: "Secret Intel",
    status: "enabled",
    artName: "detector",
    category: "choice",
    publicUpdate: "The active player is reviewing a two-player intelligence check.",
    privatePrompt: "Choose two players. The intel reveals whether either one is VIRUS.",
    input: "Choose two players",
  },
  {
    id: "Strain",
    name: "Operation: Scapegoat",
    status: "enabled",
    artName: "strain",
    category: "hidden",
    publicUpdate: "The active player received a hidden win-condition change.",
    privatePrompt: "You now win only if you are imprisoned. Your official agency stays the same.",
    input: "No input; private objective",
  },
  {
    id: "Grudge",
    name: "Grudge",
    status: "enabled",
    artName: "grudge",
    category: "hidden",
    publicUpdate: "The active player received a private target and a personal objective.",
    privatePrompt: "You are dealt the agent you hate. You now win only if that target is imprisoned.",
    input: "Server-dealt target",
  },
  {
    id: "Infatuation",
    name: "Infatuation",
    status: "enabled",
    artName: "infatuation",
    category: "hidden",
    publicUpdate: "The active player received a private loyalty attachment.",
    privatePrompt: "You now win only if the object of your affection wins the round.",
    input: "Server-dealt target",
  },
  {
    id: "Flip",
    name: "Sleeper Agent",
    status: "enabled",
    artName: "flip",
    category: "hidden",
    publicUpdate: "The server is resolving whether the active player's agency changes.",
    privatePrompt: "Your activation message reveals the agency you work for now.",
    input: "Server-resolved result",
  },
  {
    id: "HiddenOneRandom",
    name: "Secret Tip",
    status: "enabled",
    artName: "secretTip",
    category: "information",
    publicUpdate: "The active player received a private tip about one other player.",
    privatePrompt: "A strange call reveals the agency of one other player.",
    input: "Private result",
  },
  {
    id: "OneRandom",
    name: "Anonymous Tip",
    status: "enabled",
    artName: "anonymousTip",
    category: "information",
    publicUpdate: "The active player received Anonymous Tip and is explaining the operation.",
    privatePrompt: "Your source reveals the agency of one other player.",
    input: "Private result",
  },
  {
    id: "OneOfTwo",
    name: "Danish Intelligence",
    status: "enabled",
    artName: "danishIntel",
    category: "information",
    publicUpdate: "The active player is reviewing a two-name intelligence intercept.",
    privatePrompt: "Two names are shown. One is VIRUS and the other is not.",
    input: "Private result",
  },
  {
    id: "TwoFriends",
    name: "Old Photographs",
    status: "enabled",
    artName: "oldPhotographs",
    category: "information",
    publicUpdate: "The active player is explaining evidence about two players who worked for the same agency at the start.",
    privatePrompt: "The photographs show two players who worked for the same agency at the start.",
    input: "Private result",
  },
  {
    id: "Undercover",
    name: "Deep Undercover",
    status: "enabled",
    artName: "undercover",
    category: "choice",
    publicUpdate: "The active player is following one player undercover.",
    privatePrompt: "Choose a player and discover their true agency. If they are VIRUS, you join them.",
    input: "Choose one player",
  },
  {
    id: "InfoForTwo",
    name: "Unfortunate Encounter",
    status: "enabled",
    artName: "unfortunateEncounter",
    category: "choice",
    publicUpdate: "The active player invited another player into a shared intelligence result.",
    privatePrompt: "Choose one player. Both clients see whether either of you works for VIRUS.",
    input: "Choose one shared recipient",
  },
  {
    id: "ChooseVoteShield",
    name: "Incriminating Evidence",
    status: "enabled",
    artName: "chooseVoteShield",
    category: "choice",
    publicUpdate: "The active player is assigning a public accusation-phase modifier.",
    privatePrompt: "A target receives either one extra accusation vote against them or a one-vote shield.",
    input: "Choose one target and effect",
  },
  {
    id: "Defect",
    name: "Defector",
    status: "enabled",
    artName: "defect",
    category: "choice",
    publicUpdate: "The active player is deciding whether to defect from their current agency.",
    privatePrompt: "Choose whether to defect. The result changes your legal voting and win conditions.",
    input: "Defect or stay",
  },
  {
    id: "Power",
    name: "Vote of Confidence",
    status: "disabled-by-config",
    artName: "power",
    category: "public",
    publicUpdate: "The active player is assigning a double-vote privilege for accusations.",
    privatePrompt: "Choose a player whose accusation vote will count twice.",
    input: "Choose one target",
  },
  {
    id: "Vote",
    name: "Start Rumors",
    status: "disabled-by-config",
    artName: "vote",
    category: "public",
    publicUpdate: "The active player is assigning an extra vote against one player.",
    privatePrompt: "Choose a player who receives one additional vote against them.",
    input: "Choose one target",
  },
  {
    id: "Confirm",
    name: "Paycheck",
    status: "disabled-by-config",
    artName: "checkmark",
    category: "information",
    publicUpdate: "The active player received a private agency confirmation.",
    privatePrompt: "The paycheck confirms the agency the server says you work for.",
    input: "Private result",
  },
  {
    id: "NegativeVote",
    name: "Burn Evidence",
    status: "disabled-by-config",
    artName: "negativeVote",
    category: "public",
    publicUpdate: "The active player is assigning a one-vote shield for accusations.",
    privatePrompt: "Choose a player to protect from one vote of suspicion.",
    input: "Choose one target",
  },
  {
    id: "Ambassador",
    name: "Ambassador",
    status: "recovered-only",
    artName: "ambassador",
    category: "variant",
    publicUpdate: "A recovered Ambassador operation is being displayed as a supported variant.",
    privatePrompt: "The server supplies the Ambassador payload and its legal target set.",
    input: "Server-defined variant input",
  },
  {
    id: "Brig",
    name: "Brig",
    status: "recovered-only",
    artName: "brig",
    category: "variant",
    publicUpdate: "A recovered Brig operation is being displayed as a supported variant.",
    privatePrompt: "The server supplies the Brig status effect and its target.",
    input: "Server-defined variant input",
  },
  {
    id: "EarlyVote",
    name: "Early Vote",
    status: "recovered-only",
    artName: "clock",
    category: "variant",
    publicUpdate: "A recovered Early Vote operation is changing the accusation timing.",
    privatePrompt: "The server tells the room whether this player may submit an early accusation.",
    input: "Server-defined timing input",
  },
  {
    id: "Hunter",
    name: "Hunter",
    status: "recovered-only",
    artName: "detector",
    category: "variant",
    publicUpdate: "A recovered Hunter operation is preparing a target search.",
    privatePrompt: "The server supplies the Hunter's legal target set and result.",
    input: "Server-defined variant input",
  },
  {
    id: "LastEvent",
    name: "Last Event",
    status: "recovered-only",
    artName: "results",
    category: "variant",
    publicUpdate: "A recovered Last Event marker is being displayed before the final reveal.",
    privatePrompt: "The server identifies the final operation and locks the event queue.",
    input: "No input; server phase marker",
  },
] as const satisfies readonly OperationDefinition[];

export type OperationId = (typeof operationCatalog)[number]["id"];

export const liveOperationIDs: ReadonlySet<OperationId> = new Set([
  "Grudge",
  "Infatuation",
  "Share",
  "Detector",
  "Strain",
  "Flip",
  "HiddenOneRandom",
  "TwoFriends",
  "OneOfTwo",
  "OneRandom",
  "Swap",
  "Undercover",
  "InfoForTwo",
  "ChooseVoteShield",
  "Defect",
]);

export function getOperation(id: OperationId) {
  return operationCatalog.find((operation) => operation.id === id) ?? anonymousTipOperation;
}

const anonymousTipOperation = operationCatalog.find((operation) => operation.id === "OneRandom")!;

/**
 * The envelopes the Hidden Agenda cover can resolve to. Hidden Agenda is a
 * single operation in the deck: it takes one slot in the server's draw and only
 * once that slot wins does the server pick which of these arrived.
 */
export const hiddenAgendaMemberIDs: ReadonlySet<OperationId> = new Set(["Strain", "Grudge", "Infatuation", "Flip", "HiddenOneRandom"]);

/** Expansion Pack 01: ships switched off, so a host opts in per room. */
export const packOperationIDs: ReadonlySet<OperationId> = new Set(["Swap", "Undercover", "InfoForTwo", "ChooseVoteShield", "Defect"]);
