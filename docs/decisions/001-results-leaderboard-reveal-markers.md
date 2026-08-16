# ADR-001: Reveal roles and defections on the final leaderboard

## Status

Accepted

## Date

2026-08-16

## Context

The final leaderboard currently exposes each player's current faction and vote
count, but it omits the special role dealt to the player and whether a Defector
operation changed their agency. The server already keeps both facts in the
player state, while the recovered sprite set contains distinct art for every
special role and for blue/red defections.

## Decision

Add `role` and `defection` metadata to public leaderboard entries only when the
room reaches a results phase. The results screen will keep the current faction
label as the primary identity, then show a special-role badge and a defection
badge to its right when applicable. Normal Service/VIRUS roles remain unmarked;
special roles and either defection marker are revealed for every player at the
same final-results boundary.

The public contract uses the existing role IDs and the explicit
`BLUE_DEFECTOR` / `RED_DEFECTOR` status names already written by the Defector
resolver. Each badge uses the recovered sprite mapped to that role or status,
with a text label for accessibility and a tooltip for quick inspection.

## Alternatives Considered

### Derive role and defection only in the frontend

Rejected because the current public leaderboard does not contain enough
information to distinguish special roles or a defection reliably, and exposing
private projection fields earlier would leak game information.

### Show only text labels

Rejected because the game already communicates roles and agencies through its
visual language; the recovered sprites make the final board scannable without
adding a second dense text column.

### Reveal the markers during the live match

Rejected because roles and defections are intentionally hidden until the final
results. The server continues to keep these fields out of the public projection
before the results phases.

## Consequences

- The server and TypeScript projection contracts gain two optional leaderboard
  fields, while older clients can continue reading the existing fields.
- The final leaderboard can show both a role and a defection for one player,
  which matters when a non-loyal special-role holder later defects.
- The new recovered sprite paths are unique so deployed image optimization does
  not reuse stale transformed assets from the former filenames.
- Results-phase payloads reveal the full role/defection state by design; earlier
  phase payloads remain unchanged.
