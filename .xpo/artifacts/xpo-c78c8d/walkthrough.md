# Walkthrough: keyboard registry stops warning about stacked copies of a binding

## What changed

`KeyboardRegistry.dispatch` (`web/src/keyboard/registry.ts`) decides the conflict warning in a new way. Before, it compared the winner only with the runner-up and warned when both had the same priority. Now it searches all matches at the winner's priority for a binding with a different id. It warns about that binding only.

## Why

Each open `Menu` registers the same binding ids (`menu.ArrowDown`, `menu.Escape`, …) at `overlay` priority. With a submenu open, the parent copy was always the runner-up, so each key press logged a false conflict. The newest registration wins on purpose, so a copy of the winner is stacking, not a conflict.

The search goes past the runner-up for one reason. A different binding can sit behind a stacked copy, for example in third place. A check of the runner-up only would hide that real conflict.

The rule that picks the winner did not change: highest priority first, then the newest registration.

## Tests

`web/src/keyboard/registry.test.ts` has a new "conflict warnings" block with four cases. The second and fourth cases passed before the fix. The first and third cases failed before the fix.

## Acceptance criteria

- [x] Stacked registrations of the same binding id at equal priority do not warn. Evidence: test "does not warn when the same binding id is stacked at equal priority". The test also checks that the newest copy runs.
- [x] Different binding ids at equal priority still warn, including when a same-id copy sits between them. Evidence: tests "warns when different binding ids share a key at equal priority" and "warns about a different binding id behind a stacked copy of the winner".
- [x] Bindings at a lower priority than the winner never warn. Evidence: test "does not warn about a lower-priority binding on the same key".
- [x] Registry tests cover these cases. Evidence: 11 of 11 tests pass in `registry.test.ts`.
- [x] `make test` output no longer shows the conflict warnings from `FilterMenu.dom.test.tsx`. Evidence: `make test` exits 0, and its log has 0 lines with "Keyboard shortcut conflict".
