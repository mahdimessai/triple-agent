"use client";

import { GameClient } from "@/features/triple-agent/game-client";

/** Compatibility entrypoint for callers that still import the original lab root. */
export function TripleAgentLab() {
  return <GameClient />;
}
