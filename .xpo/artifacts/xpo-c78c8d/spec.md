# Keyboard registry: don't warn when a binding stacks over a copy of itself

## What

`KeyboardRegistry.dispatch` (`web/src/keyboard/registry.ts`) warns "Keyboard shortcut conflict" whenever the top two matching bindings share a priority. When a submenu is open, two `Menu` instances register the same binding ids (`menu.ArrowDown`, `menu.Escape`, …) at `overlay` priority, so every key press in the submenu warns, though the newest registration wins on purpose.

## Why

The warning exists to catch two *different* shortcuts fighting over the same key at the same priority. A binding overriding another copy of itself is stacking, not a conflict. The false warnings clutter `make test` output and hide real conflicts.

## How

In `dispatch`, after sorting the matches, look at every match at the winner's priority, not only the runner-up. Warn if any of them has a binding id different from the winner's; name that binding in the warning. Matches with the same id as the winner are ignored.

Looking past the runner-up covers this case: submenu `menu.ArrowDown` wins, parent `menu.ArrowDown` is second, and some other overlay binding `x.ArrowDown` is third. That is still a real conflict and should still warn.

Winner selection does not change: highest priority, then newest registration.

## Acceptance criteria

- [ ] Stacked registrations of the same binding id at equal priority do not warn
- [ ] Different binding ids at equal priority still warn, including when a same-id copy sits between them in the order
- [ ] Bindings at a lower priority than the winner never warn (unchanged)
- [ ] Registry tests cover these cases
- [ ] `make test` output no longer shows the conflict warnings from `FilterMenu.dom.test.tsx`
