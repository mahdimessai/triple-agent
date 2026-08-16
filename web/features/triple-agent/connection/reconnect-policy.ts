export const INITIAL_RECONNECT_DELAY_MS = 500;
export const MAX_RECONNECT_DELAY_MS = 8000;
export const RECONNECT_JITTER_MS = 250;
export const RECONNECT_GRACE_PERIOD_MS = 300_000;

/**
 * The optional random value makes the policy deterministic in tests while the
 * default keeps the production jitter used by the original client.
 */
export function reconnectDelay(attempt: number, randomValue = Math.random()) {
  const exponentialDelay = Math.min(MAX_RECONNECT_DELAY_MS, INITIAL_RECONNECT_DELAY_MS * 2 ** attempt);
  return exponentialDelay + Math.floor(randomValue * RECONNECT_JITTER_MS);
}

export function reconnectDeadline(now: number, existingDeadline?: number, gracePeriod = RECONNECT_GRACE_PERIOD_MS) {
  return existingDeadline ?? now + gracePeriod;
}

export function reconnectWaitMs(attempt: number, now: number, deadline: number, randomValue = Math.random()) {
  return Math.min(reconnectDelay(attempt, randomValue), Math.max(0, deadline - now));
}
