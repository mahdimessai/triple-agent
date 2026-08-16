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
    effect: "You are a Rogue Agent. The other VIRUS Agents do not know that you are a double agent",
  },
  {
    id: "FAKE_RED",
    name: "Triple Agent",
    faction: "SERVICE",
    artName: "roleTripleAgent",
    description: "The VIRUS agents think you are one of them. You are not.",
    effect: "You are a triple agent. The VIRUS double agents think you are on their side, but you are actually working for the Service.",
  },
  {
    id: "LYING_RED",
    name: "Deep Cover Agent",
    faction: "VIRUS",
    artName: "roleDeepCover",
    description: "You are operating under deep cover for VIRUS.",
    effect: "You are operating under deep cover. Anytime someone tries to check your status they will see you as a Service agent.",
  },
  {
    id: "LYING_BLUE",
    name: "Suspicious Agent",
    faction: "SERVICE",
    artName: "roleSuspicious",
    description: "Your past includes some ties to suspicious figures.",
    effect: "Your past includes some ties to suspicious figures. Anytime someone tries to check your status, they will see you as a VIRUS agent",
  },
  {
    id: "LOYAL_BLUE",
    name: "Service Loyalist",
    faction: "SERVICE",
    artName: "roleServiceLoyalist",
    description: "You are a die-hard loyalist and will not be turned.",
    effect: "You are a die hard loyalist. Any operation that attempts to change your team from a Service agent will be cancelled.",
  },
  {
    id: "LOYAL_RED",
    name: "VIRUS Loyalist",
    faction: "VIRUS",
    artName: "roleVirusLoyalist",
    description: "You are a die-hard loyalist and will not be turned.",
    effect: "You are a die hard loyalist. Any operation that attempts to change your team from a VIRUS agent will be cancelled.",
  },
] as const satisfies readonly RoleDefinition[];

export type RoleId = (typeof roleCatalog)[number]["id"];

export function getRole(id: string) {
  return roleCatalog.find((role) => role.id === id);
}
