---
name: nextjs-beautiful-code
description: Build and review TypeScript Next.js frontends with clear feature ownership, deliberate client boundaries, runtime validation, strict typing, focused React components, dependency direction, tests, and repository-native quality gates. Use for frontend features, refactors, reviews, architecture work, and cleanup in this repository.
---

# Next.js Beautiful Code

Write code that is easy to locate, reason about, validate, change, and remove. Optimize for clear ownership and dependency direction rather than maximum abstraction or minimum file size.

## Start by reading the repository

Before a non-trivial change:

1. Read the nearest `AGENTS.md` files.
2. Find the route or feature entry point.
3. Trace existing data flow and imports.
4. Identify the server/client boundary.
5. Find canonical protocol/domain types and runtime validators.
6. Find the nearest analogous implementation.
7. Discover the repository's actual verification commands.

Repository conventions beat personal preference unless the existing pattern is the specific problem being fixed.

## Triple Agent architecture

This repository is a realtime game, not a CRUD dashboard.

- Go is authoritative for room and game state.
- `web/app` owns Next.js routing and platform integration only.
- `web/features/triple-agent` owns the game frontend.
- The realtime `Game` client boundary is intentional because it owns WebSocket/browser interaction.
- Do not introduce Server Actions, a client state library, query library, schema library, or generic service/repository layers without a concrete need.

### Dependency direction

`protocol` is the runtime trust boundary. It imports no React, transport, session, or screens.

`transport` owns HTTP and WebSocket mechanics. It may depend on protocol, but not on React or screens.

`session` owns reducer state, persistence, reconnect policy, and React lifecycle orchestration. It may depend on protocol and transport, but not on screens.

`screens` consume trusted projections and emit typed user intent. They do not create sockets or make raw backend requests.

`game.tsx` composes session state and current screen. Keep the exhaustive phase selection obvious.

Feature-local UI primitives may know the Triple Agent visual language but must not know transport details.

## TypeScript

- Keep `strict` code genuinely strict.
- Use `unknown` for data crossing an untrusted runtime boundary.
- Narrow before data enters trusted application code.
- Preserve discriminated unions for phases, commands, connection events, notices, and reducer actions.
- Prefer exhaustive switches with `never` checks.
- Do not use `any`, `@ts-ignore`, or unchecked assertions to silence problems.
- Derive types from canonical sources when practical. Do not duplicate equivalent domain unions casually.

## React

Keep state near the interaction that owns it.

Effects are for synchronization with external systems such as WebSockets, browser lifecycle events, storage, audio, focus, and platform APIs. Do not use effects to mirror ordinary render state.

Do not add `memo`, `useMemo`, or `useCallback` ritualistically. Use them when referential identity or computation cost matters.

Split components when a semantic concept has its own state, behavior, accessibility contract, or meaningful independent responsibility. Do not split JSX just to chase a line-count target.

Keep accessibility behavior with the component that owns the interaction. Dialogs need deliberate labeling, keyboard behavior, focus entry, and focus return.

## Modules and helpers

Name modules after concepts rather than generic buckets. Prefer `reconnect-policy.ts`, `room-storage.ts`, and `room-socket.ts` over `utils.ts` or `helpers.ts` when the concept is specific.

Keep `index.ts` files as small intentional public APIs, not barrel dumps.

A large static catalog can remain a large file when it is one coherent concept. Do not create one file per constant or operation to make the tree look architectural.

## Runtime boundaries

Validate external values at entry points:

- HTTP JSON
- WebSocket messages
- local/session storage JSON
- URL/query input
- environment variables where relevant

The frontend validator protects runtime integrity. It does not replace the server's responsibility to prevent private-state disclosure.

## Testing

Test at the smallest layer that owns the risk.

- Pure reducer and policy behavior: unit tests.
- Protocol parsing: malformed and valid runtime fixtures.
- Transport: authentication, serialization, resync, error/close behavior.
- Architecture: dependency direction, not stylistic ideology.
- Cross-browser multiplayer behavior: end-to-end tests when the project adopts the required browser test tooling.

Do not write tests that duplicate guarantees already provided more strongly by TypeScript or the build.

## Lint and architecture rules

Use lint for likely code mistakes, TypeScript for type safety, architecture tests for module-direction invariants, and behavioral tests for runtime behavior.

Do not ban classes, dynamic imports, directory names, or other syntax globally unless there is a concrete repository invariant behind the rule.

## Verification

From the repository root, run:

```sh
make verify
```

That is the completion gate. It runs Go tests plus frontend typecheck, lint, tests, and production build.

Do not claim verification passed unless it actually ran successfully.

## Review standard

Before finalizing a substantial frontend change, read `references/review-checklist.md`.

A strong diff should make these answers obvious:

- Where does the feature belong?
- What is trusted versus untrusted data?
- Which module owns browser/network lifecycle?
- Which modules are allowed to depend on which others?
- Is the client boundary justified?
- Are state and effects owned by the right component/module?
- Are abstractions representing real concepts?
- Are failures and destructive actions intentional?
- Is the public module surface small?
- Did `make verify` pass?

Never manufacture complexity to demonstrate clean architecture.
