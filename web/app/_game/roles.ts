import type { ArtName } from "./assets";

export type RoleDefinition = {
  id: string;
  name: string;
  faction: "SERVICE" | "VIRUS";
  artName: ArtName;
  description: string;
  effect: string;
};

export const roles = [
  {
    id: "FAKE_BLUE",
    name: "Rogue Agent",
    faction: "VIRUS",
    artName: "roleRogueAgent",
    description: "You work for VIRUS, but the other VIRUS agents were never told about you.",
    effect: "The other VIRUS agents do not know that you are a double agent.",
  },
  {
    id: "FAKE_RED",
    name: "Triple Agent",
    faction: "SERVICE",
    artName: "roleTripleAgent",
    description: "The VIRUS agents think you are one of them. You are not.",
    effect: "The VIRUS agents think you are on their side, but you actually work for SERVICE.",
  },
  {
    id: "LYING_RED",
    name: "Deep Cover Agent",
    faction: "VIRUS",
    artName: "roleDeepCover",
    description: "You are operating under deep cover for VIRUS.",
    effect: "Agency checks report you as SERVICE.",
  },
  {
    id: "LYING_BLUE",
    name: "Suspicious Agent",
    faction: "SERVICE",
    artName: "roleSuspicious",
    description: "Your past includes some ties to suspicious figures.",
    effect: "Agency checks report you as VIRUS.",
  },
  {
    id: "LOYAL_BLUE",
    name: "Service Loyalist",
    faction: "SERVICE",
    artName: "roleServiceLoyalist",
    description: "You are a die-hard loyalist and will not be turned.",
    effect: "Operations that would move you away from SERVICE are cancelled.",
  },
  {
    id: "LOYAL_RED",
    name: "VIRUS Loyalist",
    faction: "VIRUS",
    artName: "roleVirusLoyalist",
    description: "You are a die-hard loyalist and will not be turned.",
    effect: "Operations that would move you away from VIRUS are cancelled.",
  },
] as const satisfies readonly RoleDefinition[];

export type RoleId = (typeof roles)[number]["id"];

export function getRole(id: string | undefined): RoleDefinition | undefined {
  if (!id) return undefined;
  return roles.find((role) => role.id === id);
}
