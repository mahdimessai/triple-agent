import type { ArtName } from "./asset-registry";

/**
 * Special roles are dealt on top of an agency, never instead of one: the holder
 * still wins with the side they actually work for. Mirrors the server's role
 * table in `server/internal/domain/roles.go`; the ids are the wire contract for
 * `lobby.role_enabled`.
 */
export type RoleDefinition = {
  id: string;
  name: string;
  faction: "SERVICE" | "VIRUS";
  artName: ArtName;
  /** What the holder is told about themselves. */
  description: string;
  /** What the role actually changes at the table. */
  effect: string;
};

export const roleCatalog = [
  {
    id: "FAKE_BLUE",
    name: "Rogue Agent",
    faction: "VIRUS",
    artName: "roleRogueAgent",
    description: "You work for VIRUS, but the other VIRUS agents were never told about you.",
    effect: "Left off the VIRUS roster, so your own side reads one name short.",
  },
  {
    id: "FAKE_RED",
    name: "Triple Agent",
    faction: "SERVICE",
    artName: "roleTripleAgent",
    description: "The VIRUS agents think you are one of them. You are not.",
    effect: "Listed on the VIRUS roster and shown it, so their side reads one name long.",
  },
  {
    id: "LYING_RED",
    name: "Deep Cover Agent",
    faction: "VIRUS",
    artName: "roleDeepCover",
    description: "You are operating under deep cover for VIRUS.",
    effect: "Every check on you reports SERVICE.",
  },
  {
    id: "LYING_BLUE",
    name: "Suspicious Agent",
    faction: "SERVICE",
    artName: "roleSuspicious",
    description: "Your past includes some ties to suspicious figures.",
    effect: "Every check on you reports VIRUS.",
  },
  {
    id: "LOYAL_BLUE",
    name: "Service Loyalist",
    faction: "SERVICE",
    artName: "roleServiceLoyalist",
    description: "You are a die-hard loyalist and will not be turned.",
    effect: "Any operation that would move you off SERVICE is cancelled.",
  },
  {
    id: "LOYAL_RED",
    name: "VIRUS Loyalist",
    faction: "VIRUS",
    artName: "roleVirusLoyalist",
    description: "You are a die-hard loyalist and will not be turned.",
    effect: "Any operation that would move you off VIRUS is cancelled.",
  },
] as const satisfies readonly RoleDefinition[];

export type RoleId = (typeof roleCatalog)[number]["id"];

export function getRole(id: string) {
  return roleCatalog.find((role) => role.id === id);
}
