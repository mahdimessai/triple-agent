export type Phase =
  | "LOBBY"
  | "ROLE_REVEAL"
  | "OPERATION_INPUT"
  | "OPERATION_RESULT"
  | "OPERATION_INTERLUDE"
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

export type Faction = "SERVICE" | "VIRUS" | "NONE";

export type PlayerProjection = {
  id: string;
  name: string;
  seat: number;
  ready: boolean;
  connected: boolean;
  vote_submitted: boolean;
};

export type OperationResult = {
  code?: string;
  target_player_id?: string;
  target_player_ids?: string[];
  target_faction?: Faction;
  other_player_id?: string;
  other_faction?: Faction;
  your_faction?: Faction;
  message: string;
};

export type OperationInputKind = "NONE" | "ONE_TARGET" | "TWO_TARGETS" | "CHOICE" | "PRIVATE_INFO";

export type RoomProjection = {
  type: "room.projection";
  public: {
    room_id: string;
    host_id: string;
    phase: Phase;
    version: number;
    players: PlayerProjection[];
    settings: {
      discussion_timer_enabled: boolean;
      discussion_seconds: number;
      enabled_operations: string[];
      min_players?: number;
      max_players?: number;
      interlude_seconds?: number;
      virus_count?: number;
      enabled_roles?: string[];
    };
    active_player_id?: string;
    operation?: {
      kind: string;
      name: string;
      input_kind: OperationInputKind;
      target_count?: number;
      active_player_id: string;
      active_player_name: string;
      input_owner_id?: string;
      step?: number;
      public_instruction: string;
    };
    discussion_deadline?: string;
    vote_totals?: Record<string, number>;
    imprisoned_player_id?: string;
    revealed_faction?: Faction;
    winner?: Faction;
    leaderboard?: Array<{
      player_id: string;
      name: string;
      faction: Faction;
      role?: string;
      defection?: "BLUE_DEFECTOR" | "RED_DEFECTOR";
      votes: number;
      result: "WINNER" | "LOSER" | "DRAW";
    }>;
    activity?: string;
    pending_role_acks?: number;
    discussion_ready_count?: number;
  };
  private: {
    player_id: string;
    role?: string;
    initial_faction?: Faction;
    faction?: Faction;
    apparent_faction?: Faction;
    operation_result?: OperationResult;
    operation_instruction?: string;
    role_name?: string;
    role_description?: string;
    role_effect?: string;
    /** Sent only to agents who believe they are on the VIRUS side. The names are
     * the roster, the size is how many agents really started on VIRUS; a Rogue
     * or Triple Agent makes the two disagree. */
    virus_roster?: Array<{ id: string; name: string; seat: number; connected: boolean }>;
    virus_team_size?: number;
    /** Set only for hidden operations, and only for the recipient: the real
     * operation behind the public Hidden Agenda cover. */
    operation_kind?: string;
    operation_name?: string;
    legal_target_ids?: string[];
    choices?: string[];
    vote_submitted: boolean;
    can_submit: boolean;
  };
};
