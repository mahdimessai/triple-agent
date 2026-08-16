import type { ArtName } from "../asset-registry";

export type MockScreenGroup = "FLOW" | "OPERATIONS" | "VOTING" | "RESULTS";
export type MockPrivacy = "shared" | "private" | "mixed";
export type MockScreenStatus = "implemented" | "planned";
export type MockPhase =
  | "LOBBY"
  | "ROLE_REVEAL"
  | "OPERATION_INPUT"
  | "OPERATION_RESULT"
  | "DISCUSSION"
  | "VOTE_INPUT"
  | "RESULTS_INTRO"
  | "VOTE_RESULTS"
  | "IMPRISONMENT_REVEAL"
  | "AGENCY_REVEAL"
  | "OUTCOME_REVEAL"
  | "LEADERBOARD"
  | "OUT_OF_LOOP"
  | "END";

export type MockScreenRenderer =
  | "title"
  | "setup"
  | "briefing"
  | "role-service"
  | "role-virus"
  | "operations-intro"
  | "hidden-agenda"
  | "operation-waiting"
  | "anonymous-description"
  | "anonymous-private-result"
  | "anonymous-public-explanation"
  | "discussion"
  | "accusation-guard"
  | "accusation-input"
  | "results-intro"
  | "vote-results"
  | "imprisonment-reveal"
  | "agency-reveal"
  | "outcome-reveal"
  | "leaderboard"
  | "out-of-loop"
  | "end"
  | "planned";

export type MockScreenDefinition = {
  id: string;
  number: string;
  label: string;
  group: MockScreenGroup;
  privacy: MockPrivacy;
  phase?: MockPhase;
  title: string;
  description: string;
  artName: ArtName;
  render: MockScreenRenderer;
  status: MockScreenStatus;
};

export const mockScreenRegistry = [
  {
    id: "title-registration",
    number: "01",
    label: "Title / registration",
    group: "FLOW",
    privacy: "shared",
    title: "Title and player registration",
    description: "Enter a name and start or join a server-backed room.",
    artName: "twoMen",
    render: "title",
    status: "implemented",
  },
  {
    id: "setup-lobby",
    number: "02",
    label: "Setup / lobby",
    group: "FLOW",
    privacy: "shared",
    phase: "LOBBY",
    title: "Setup and lobby roster",
    description: "Collect five ready players before the host starts the match.",
    artName: "passport",
    render: "setup",
    status: "implemented",
  },
  {
    id: "mission-briefing",
    number: "03",
    label: "Mission briefing",
    group: "FLOW",
    privacy: "shared",
    title: "Mission briefing",
    description: "Explain the agency count, the objective, and private server state.",
    artName: "passport",
    render: "briefing",
    status: "implemented",
  },
  {
    id: "role-service",
    number: "04",
    label: "Role: Service",
    group: "FLOW",
    privacy: "private",
    phase: "ROLE_REVEAL",
    title: "Service role reveal",
    description: "Show the current player their private Service assignment.",
    artName: "serviceLogo",
    render: "role-service",
    status: "implemented",
  },
  {
    id: "role-virus",
    number: "05",
    label: "Role: VIRUS",
    group: "FLOW",
    privacy: "private",
    phase: "ROLE_REVEAL",
    title: "VIRUS role reveal",
    description: "Show the current player their private VIRUS assignment and teammates.",
    artName: "virusLogo",
    render: "role-virus",
    status: "implemented",
  },
  {
    id: "operations-intro",
    number: "06",
    label: "Operations / intro",
    group: "FLOW",
    privacy: "mixed",
    phase: "OPERATION_INPUT",
    title: "Operations phase introduction",
    description: "Explain public operation activity and private payload delivery.",
    artName: "operations",
    render: "operations-intro",
    status: "implemented",
  },
  {
    id: "operation-waiting",
    number: "07",
    label: "Operation / waiting",
    group: "OPERATIONS",
    privacy: "shared",
    phase: "OPERATION_INPUT",
    title: "Operation waiting",
    description: "Hold the room in an explicit waiting state while the server advances.",
    artName: "clock",
    render: "operation-waiting",
    status: "implemented",
  },
  {
    id: "hidden-agenda",
    number: "08",
    label: "Operation / hidden agenda",
    group: "OPERATIONS",
    privacy: "private",
    phase: "OPERATION_INPUT",
    title: "Hidden agenda cover",
    description: "Announce a hidden operation to the room under its shared cover name, with the real orders kept private.",
    artName: "hiddenAgenda",
    render: "hidden-agenda",
    status: "implemented",
  },
  {
    id: "anonymous-tip-description",
    number: "09",
    label: "Anonymous Tip / description",
    group: "OPERATIONS",
    privacy: "mixed",
    phase: "OPERATION_INPUT",
    title: "Anonymous Tip description",
    description: "Show the public operation description before the private answer.",
    artName: "anonymousTip",
    render: "anonymous-description",
    status: "implemented",
  },
  {
    id: "anonymous-tip-private-result",
    number: "10",
    label: "Anonymous Tip / private result",
    group: "OPERATIONS",
    privacy: "private",
    phase: "OPERATION_RESULT",
    title: "Anonymous Tip private result",
    description: "Deliver the target and agency result directly to the active player.",
    artName: "anonymousTip",
    render: "anonymous-private-result",
    status: "implemented",
  },
  {
    id: "anonymous-tip-public-explanation",
    number: "11",
    label: "Anonymous Tip / public explanation",
    group: "OPERATIONS",
    privacy: "shared",
    phase: "OPERATION_RESULT",
    title: "Anonymous Tip public explanation",
    description: "Return the room to a safe public explanation state after the result.",
    artName: "anonymousTip",
    render: "anonymous-public-explanation",
    status: "implemented",
  },
  {
    id: "discussion",
    number: "12",
    label: "Discussion / timer",
    group: "VOTING",
    privacy: "shared",
    phase: "DISCUSSION",
    title: "Discussion timer and history",
    description: "Give the table a readable public history while the timer counts down.",
    artName: "clock",
    render: "discussion",
    status: "implemented",
  },
  {
    id: "accusation-guard",
    number: "13",
    label: "Accusation / guard",
    group: "VOTING",
    privacy: "mixed",
    phase: "VOTE_INPUT",
    title: "Accusation privacy guard",
    description: "Explain that the accusation is private and held until every player submits.",
    artName: "accusation",
    render: "accusation-guard",
    status: "implemented",
  },
  {
    id: "accusation-input",
    number: "14",
    label: "Accusation / input",
    group: "VOTING",
    privacy: "private",
    phase: "VOTE_INPUT",
    title: "Accusation target input",
    description: "Show the target list with a clear single-selection state and submission affordance.",
    artName: "accusation",
    render: "accusation-input",
    status: "implemented",
  },
  {
    id: "results-introduction",
    number: "15",
    label: "Results / introduction",
    group: "RESULTS",
    privacy: "shared",
    phase: "RESULTS_INTRO",
    title: "Results introduction",
    description: "Set expectations before each result is revealed in order.",
    artName: "results",
    render: "results-intro",
    status: "implemented",
  },
  {
    id: "vote-results",
    number: "16",
    label: "Results / vote totals",
    group: "RESULTS",
    privacy: "shared",
    phase: "VOTE_RESULTS",
    title: "Vote totals",
    description: "Reveal the public tally before exposing the imprisoned player or faction.",
    artName: "results",
    render: "vote-results",
    status: "implemented",
  },
  {
    id: "imprisonment-reveal",
    number: "17",
    label: "Results / imprisonment",
    group: "RESULTS",
    privacy: "shared",
    phase: "IMPRISONMENT_REVEAL",
    title: "Imprisonment reveal",
    description: "Name the imprisoned player in a discrete stage before revealing the faction.",
    artName: "imprisoned",
    render: "imprisonment-reveal",
    status: "implemented",
  },
  {
    id: "agency-reveal",
    number: "18",
    label: "Results / agency",
    group: "RESULTS",
    privacy: "shared",
    phase: "AGENCY_REVEAL",
    title: "Agency reveal",
    description: "Reveal the imprisoned player’s agency with enough context to understand the consequence.",
    artName: "serviceLogo",
    render: "agency-reveal",
    status: "implemented",
  },
  {
    id: "outcome-reveal",
    number: "19",
    label: "Results / winner",
    group: "RESULTS",
    privacy: "shared",
    phase: "OUTCOME_REVEAL",
    title: "Winner reveal",
    description: "Use a deliberate winner flip state so the result cannot be mistaken for an intermediate reveal.",
    artName: "results",
    render: "outcome-reveal",
    status: "implemented",
  },
  {
    id: "leaderboard",
    number: "20",
    label: "Results / leaderboard",
    group: "RESULTS",
    privacy: "shared",
    phase: "LEADERBOARD",
    title: "Leaderboard",
    description: "Make the final score order scannable before offering a rematch or exit.",
    artName: "results",
    render: "leaderboard",
    status: "implemented",
  },
  {
    id: "out-of-loop",
    number: "21",
    label: "Results / out of loop",
    group: "RESULTS",
    privacy: "mixed",
    phase: "OUT_OF_LOOP",
    title: "Out-of-loop state",
    description: "Explain what an eliminated player can still see and what remains private.",
    artName: "handcuffs",
    render: "out-of-loop",
    status: "implemented",
  },
  {
    id: "match-complete",
    number: "22",
    label: "Results / match complete",
    group: "RESULTS",
    privacy: "shared",
    phase: "END",
    title: "Match complete and rematch",
    description: "Close the loop with a clear end state and a contained Play again action.",
    artName: "playAgain",
    render: "end",
    status: "implemented",
  },
] as const satisfies readonly MockScreenDefinition[];

export type MockScreenId = (typeof mockScreenRegistry)[number]["id"];

export const mockScreenGroups: readonly MockScreenGroup[] = ["FLOW", "OPERATIONS", "VOTING", "RESULTS"];

export function getMockScreen(id: string | undefined) {
  return mockScreenRegistry.find((screen) => screen.id === id) ?? mockScreenRegistry[0];
}
