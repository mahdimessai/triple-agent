export const RECONNECT_GRACE_PERIOD_MS = 300_000;

const INITIAL_RECONNECT_DELAY_MS = 500;
const MAX_RECONNECT_DELAY_MS = 8_000;
const RECONNECT_JITTER_MS = 250;

export function reconnectDelay(attempt: number, randomValue = Math.random()): number {
  const exponential = Math.min(MAX_RECONNECT_DELAY_MS, INITIAL_RECONNECT_DELAY_MS * 2 ** attempt);
  return exponential + Math.floor(randomValue * RECONNECT_JITTER_MS);
}
