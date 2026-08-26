import type { OperationDefinition } from "../../operations";
import type { RoleDefinition } from "../../roles";

export type ToggleHandler = (id: string, enabled: boolean) => void;

export type InspectedItem =
  | { type: "operation"; operation: OperationDefinition; label: string; enabled: boolean; disabled: boolean }
  | { type: "role"; role: RoleDefinition; enabled: boolean; disabled: boolean }
  | null;
