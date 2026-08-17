# ADR-003: Release lobby seats on disconnect

## Status

Accepted

## Date

2026-08-17

## Context

An in-game disconnect needs a grace period so a browser reload or short network
outage can restore the same player state. A lobby has no game state to protect,
so reserving a disconnected seat blocks new players and lets an old host reclaim
leadership after someone else has taken over.

## Decision

When a socket disconnects during `LOBBY`, the server removes that player from
`PlayerOrder` immediately, compacts the remaining seats, and transfers the host
to the first remaining player in join order. The reconnect token cannot attach
that removed identity; joining again creates a new player identity appended to
the lobby. Once the match has started, disconnects retain the seat and the
existing reconnect grace period. Reconnection only restores presence and never
reclaims a host role that was transferred while the player was away.

The Old Photographs operation follows its stated clue: it selects two players
whose visible starting agencies match and always returns the same-agency result.

## Consequences

The lobby client must stop automatic reconnect attempts after a lobby socket
close and return the user to the join flow. In-game clients continue to retry
within the grace period, and old operation result codes remain renderable for
backward-compatible replay handling even though the server no longer emits the
different-agency result.
