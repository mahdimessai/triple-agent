# Triple Agent frontend review checklist

Use this after substantial frontend work. Apply judgment rather than treating every item as a mechanical rule.

## Ownership and architecture

- [ ] `web/app` contains routing/platform composition rather than game implementation.
- [ ] New Triple Agent behavior has one obvious home under `features/triple-agent`.
- [ ] Protocol code has no React, transport, session, or screen dependencies.
- [ ] Transport code has no React or screen dependencies.
- [ ] Session code does not depend on screens.
- [ ] Screens do not instantiate sockets or perform raw backend requests.
- [ ] Feature `index.ts` files expose a small intentional API.
- [ ] No generic abstraction was added without a concrete ownership problem.

## Runtime and protocol safety

- [ ] New external JSON starts as `unknown` and is runtime validated.
- [ ] Persisted data is validated before reuse.
- [ ] New protocol variants are represented in canonical TypeScript unions.
- [ ] Exhaustive phase/command/event switches remain exhaustive.
- [ ] Private server projection rules remain server-authoritative and tested where appropriate.

## React

- [ ] State lives near the interaction or lifecycle that owns it.
- [ ] Effects synchronize real external systems rather than mirror render state.
- [ ] `memo`, `useMemo`, and `useCallback` have a concrete reason when present.
- [ ] Components are split by semantic responsibility, not arbitrary line count.
- [ ] Loading, pending, error, disconnected, and empty states are deliberate.
- [ ] Destructive actions have an intentional UX consequence.

## Accessibility

- [ ] Interactive controls use semantic elements.
- [ ] Icon-only controls have accessible names.
- [ ] Dialogs are labeled and handle keyboard/focus behavior deliberately.
- [ ] Important status/error messages are exposed appropriately to assistive technology.
- [ ] Keyboard navigation remains usable after component extraction/refactoring.

## TypeScript and code quality

- [ ] No new `any`, `@ts-ignore`, or unexplained unchecked assertions.
- [ ] Domain concepts use specific names instead of generic `utils`/`helpers` buckets.
- [ ] Derived values are computed rather than mirrored in state unnecessarily.
- [ ] Comments explain non-obvious reasons, browser constraints, or invariants rather than narrating obvious code.
- [ ] The diff does not introduce a second competing architectural vocabulary.

## Tests and verification

- [ ] Pure policies/reducers have focused tests when changed.
- [ ] Protocol parsing tests cover invalid as well as valid runtime values when the contract changes.
- [ ] Transport tests cover wire behavior when transport changes.
- [ ] Architecture tests enforce dependency direction rather than stylistic preferences.
- [ ] Existing critical behavior tests still pass.
- [ ] `make verify` has been run successfully from the repository root.

## Final review questions

- [ ] Can another engineer find the changed capability quickly?
- [ ] Can the dependency direction be understood without reading every file?
- [ ] Is the game still server-authoritative?
- [ ] Is client-side code justified by realtime/browser behavior?
- [ ] Did the change make the code easier to change without making it more ceremonious?
