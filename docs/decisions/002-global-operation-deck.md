# ADR-002: Use a global shuffled operation deck

## Status

Accepted

## Date

2026-08-17

## Context

Operation selection was random per recipient. Each player tracked their own
dealt operations, so the same operation could repeat before another enabled
operation had appeared. The operation phase also ended after one pass through
the players, which meant a pool larger than the table could never be served in
full.

The room already treats Hidden Agenda as one public operation with several
private envelopes, and some operations have a minimum event order. Those rules
must remain intact while global operation uniqueness is added.

## Decision

At match start, build one deck from the enabled, live, player-compatible
operation slots and shuffle it with the server's deterministic random state.
Consume the deck in order. A named operation occupies one slot; the complete
Hidden Agenda envelope group occupies one slot and chooses its envelope only
when that slot is dealt.

Player assignment uses a separate shuffled recipient queue. The queue cycles
when the deck contains more slots than players. The operation phase deals the
larger of the deck size and the player count, so every enabled slot appears at
least once and a smaller pool only repeats after it is exhausted to give every
player a turn.

Eligibility is never relaxed to force a draw. Disabled, recovered, too-small,
or future-event operations remain unavailable. If the configured pool has no
operation legal for the first event, match start fails with
`ErrNoEligibleOperations`.

When a new deck cycle is shuffled, the first slot is swapped with another slot
when necessary to avoid repeating the previous cycle's final slot immediately.

## Alternatives Considered

### Random draw per player

Rejected because per-player history does not provide global uniqueness and can
skip enabled operations indefinitely.

### One operation per player

Rejected because it cannot serve a configured pool larger than the player
count. It also makes the frontend's operation-pool promise false.

### Independent deck per player

Rejected because it repeats operations across players and requires more state
without improving fairness.

## Consequences

- The server owns operation ordering; the frontend only configures the pool and
  renders the current projection.
- Deck state and recipient cursor must survive reconnect/resync unchanged.
- Per-player history remains only for recipient-specific rules such as one
  Hidden Agenda envelope per player; it no longer selects named operations.
- Tests must cover deck exhaustion, reshuffle boundaries, event-order gates,
  disabled explicit requests, hidden grouping, rematches, and reconnects.
