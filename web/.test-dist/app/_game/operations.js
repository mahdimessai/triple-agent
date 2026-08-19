"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.packOperationIds = exports.hiddenAgendaMemberIds = exports.liveOperationIds = exports.operations = void 0;
exports.countDeckOperations = countDeckOperations;
exports.getOperation = getOperation;
exports.operationIdForServerKind = operationIdForServerKind;
exports.operationBrief = operationBrief;
exports.roomBriefing = roomBriefing;
exports.operationResultText = operationResultText;
exports.operations = [
    {
        id: "HiddenAgenda",
        name: "Hidden Agenda",
        status: "enabled",
        artName: "hiddenAgenda",
        category: "hidden",
        publicUpdate: "The active player has new orders from up top. They could switch sides, gain a new win condition, or learn another agent's agency.",
        privatePrompt: "Read your orders in secret.",
        input: "No input; private orders",
        brief: "New orders arrive: a side switch, a new win condition, or intel on one agent.",
        roomBriefing: "{recipient} gets new orders from up top. They could switch sides, gain a new win condition, or simply learn another agent's agency.",
        recipientBriefing: "You get new orders from up top. You could switch sides, gain a new win condition, or simply learn another agent's agency.",
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
        brief: "The active player chooses another agent and secretly swaps agencies with them.",
        roomBriefing: "{recipient} secretly exchanges agencies with one player of their choice.",
        recipientBriefing: "You secretly exchange agencies with one player of your choice.",
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
        brief: "Attempt to recruit a target into your agency.",
        roomBriefing: "{recipient} attempts to recruit one player into their agency.",
        recipientBriefing: "You attempt to recruit one player into your agency.",
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
        brief: "The active player shows one other agent which agency they work for.",
        roomBriefing: "{recipient} shows their agency to one player of their choice.",
        recipientBriefing: "You show your agency to one player of your choice.",
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
        brief: "Choose two agents. Intel reveals whether either of them is VIRUS.",
        roomBriefing: "{recipient} checks two players and learns whether either one is VIRUS.",
        recipientBriefing: "You check two players and learn whether either one is VIRUS.",
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
        brief: "HIDDEN AGENDA: the active player now wins only if they are imprisoned.",
        roomBriefing: "{recipient} receives a private objective tied to being imprisoned.",
        recipientBriefing: "You receive a private objective tied to being imprisoned.",
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
        brief: "HIDDEN AGENDA: the active player wins only if their secret target is imprisoned.",
        roomBriefing: "{recipient} is dealt one player they need to see imprisoned.",
        recipientBriefing: "You are dealt one player you need to see imprisoned.",
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
        brief: "HIDDEN AGENDA: the active player wins only if the object of their affection wins.",
        roomBriefing: "{recipient} is bound to another player's victory.",
        recipientBriefing: "You are bound to another player's victory.",
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
        brief: "HIDDEN AGENDA: the active player secretly switches agency.",
        roomBriefing: "{recipient}'s agency may quietly change.",
        recipientBriefing: "Your agency may quietly change.",
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
        brief: "HIDDEN AGENDA: the active player learns the agency of one other agent.",
        roomBriefing: "{recipient} receives a private tip naming one other player's agency.",
        recipientBriefing: "You receive a private tip naming one other player's agency.",
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
        brief: "The active player's source reveals the agency of one other agent.",
        roomBriefing: "{recipient}'s source knows the agency of one other player and reveals it to them.",
        recipientBriefing: "Your source knows the agency of one other player and reveals it to you.",
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
        brief: "Two names are shown: one is VIRUS and the other is not.",
        roomBriefing: "{recipient} learns that one of two named players is VIRUS and the other is not.",
        recipientBriefing: "You learn that one of two named players is VIRUS and the other is not.",
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
        brief: "The active player sees two agents who worked for the same agency at the start.",
        roomBriefing: "{recipient} is shown two agents who worked for the same agency at the start.",
        recipientBriefing: "You are shown two agents who worked for the same agency at the start.",
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
        brief: "Investigate one agent. If they are VIRUS, the active player joins them.",
        roomBriefing: "{recipient} investigates one player and may join them.",
        recipientBriefing: "You investigate one player and may join them.",
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
        brief: "The active player and one chosen agent learn whether either works for VIRUS.",
        roomBriefing: "{recipient} and one player of their choice share the same VIRUS check.",
        recipientBriefing: "You and one player of your choice share the same VIRUS check.",
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
        brief: "Assign one player an accusation penalty or a shield.",
        roomBriefing: "{recipient} assigns one player a vote penalty or a shield.",
        recipientBriefing: "You assign one player a vote penalty or a shield.",
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
        brief: "The active player may defect and join the other agency.",
        roomBriefing: "{recipient} chooses in secret whether to defect from their agency.",
        recipientBriefing: "You choose in secret whether to defect from your agency.",
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
        brief: "Give one player a double accusation vote.",
        roomBriefing: "{recipient} gives one player a double accusation vote.",
        recipientBriefing: "You give one player a double accusation vote.",
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
        brief: "Give one player an extra accusation vote against them.",
        roomBriefing: "{recipient} puts extra suspicion on one player.",
        recipientBriefing: "You put extra suspicion on one player.",
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
        brief: "Confirm your current agency privately.",
        roomBriefing: "{recipient} privately confirms their own current agency.",
        recipientBriefing: "You privately confirm your own current agency.",
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
        brief: "Protect one player from one accusation vote.",
        roomBriefing: "{recipient} shields one player from a vote.",
        recipientBriefing: "You shield one player from a vote.",
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
        brief: "Recovered variant with a server-defined effect.",
        roomBriefing: "{recipient} receives a server-defined Ambassador operation.",
        recipientBriefing: "You receive a server-defined Ambassador operation.",
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
        brief: "Recovered variant with a server-defined status effect.",
        roomBriefing: "{recipient} receives a server-defined Brig operation.",
        recipientBriefing: "You receive a server-defined Brig operation.",
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
        brief: "Recovered variant that changes accusation timing.",
        roomBriefing: "{recipient} receives an Early Vote timing operation.",
        recipientBriefing: "You receive an Early Vote timing operation.",
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
        brief: "Recovered variant that prepares a target search.",
        roomBriefing: "{recipient} receives a server-defined Hunter operation.",
        recipientBriefing: "You receive a server-defined Hunter operation.",
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
        brief: "Recovered variant that marks the final event.",
        roomBriefing: "{recipient} receives the final event marker.",
        recipientBriefing: "You receive the final event marker.",
    },
];
const anonymousTip = exports.operations.find((operation) => operation.id === "OneRandom");
exports.liveOperationIds = new Set([
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
exports.hiddenAgendaMemberIds = new Set([
    "Strain",
    "Grudge",
    "Infatuation",
    "Flip",
    "HiddenOneRandom",
]);
exports.packOperationIds = new Set([
    "Swap",
    "Undercover",
    "InfoForTwo",
    "ChooseVoteShield",
    "Defect",
]);
/* The Hidden Agenda envelopes are not operations in their own right: they are the
   ways the single Hidden Agenda operation resolves. Anything reporting a deck size
   counts the cover once and ignores the envelopes. */
function countDeckOperations(enabledIds) {
    let count = 0;
    let hiddenAgenda = false;
    for (const id of enabledIds) {
        if (id === "HiddenAgenda" || exports.hiddenAgendaMemberIds.has(id)) {
            hiddenAgenda = true;
            continue;
        }
        count += 1;
    }
    return count + (hiddenAgenda ? 1 : 0);
}
function getOperation(id) {
    if (!id)
        return anonymousTip;
    return exports.operations.find((operation) => operation.id === id) ?? anonymousTip;
}
function operationIdForServerKind(kind) {
    if (kind === "Swap" || kind === "SpyTransfer")
        return "Swap";
    if (kind && exports.operations.some((operation) => operation.id === kind))
        return kind;
    return "OneRandom";
}
function operationBrief(operation) {
    return operation.brief;
}
function roomBriefing(operation, recipient, isRecipient, fallback) {
    if (isRecipient)
        return operation.recipientBriefing || fallback;
    return (operation.roomBriefing || fallback).replaceAll("{recipient}", recipient);
}
function operationResultText(result, projection) {
    const nameFor = (id) => id
        ? projection.public.players.find((player) => player.id === id)?.name ?? "an unknown player"
        : undefined;
    const target = nameFor(result.target_player_id);
    const other = nameFor(result.other_player_id);
    const named = (result.target_player_ids ?? []).map((id) => nameFor(id)).filter(Boolean);
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
            return named.length === 2 ? `${named[0]} and ${named[1]} worked for the same agency at the start.` : result.message;
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
            if (named.length === 2 && result.target_faction)
                return `Between ${named[0]} and ${named[1]}, one is ${result.target_faction}.`;
            if (target && result.target_faction)
                return `${target} is ${result.target_faction}.`;
            if (other && result.other_faction)
                return `${other} is ${result.other_faction}.`;
            return result.message;
    }
}
