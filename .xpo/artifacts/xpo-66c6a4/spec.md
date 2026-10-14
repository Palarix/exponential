# Dashboard Active Epics: no nested buttons

## What
Each Active Epics row (`web/src/components/Dashboard/Dashboard.tsx:453`) is a `<button>` that contains `<CopyableId>`, itself a `<button>`. Nested buttons are invalid HTML and React logs an error on every Dashboard load.

## How
- Extract the row into an `EpicRow` component (`web/src/components/Dashboard/EpicRow.tsx`) so its markup can be tested with `renderMarkup`.
- The row becomes a `<div>` with a mouse `onClick` that opens the epic, the same pattern as the MyIssues rows.
- The status icon and title sit inside a real `<button>`, which opens the epic. It gives keyboard and screen-reader users a focus target. `CopyableId` is a sibling of that button, not a descendant.
- `CopyableId` already calls `stopPropagation`, so clicking the ID copies it and doesn't open the epic.
- The look stays the same: the row keeps its hover background and layout.

## Decision
Why not `role="button"` on the row div? That still nests one interactive control inside another (axe `nested-interactive`), and a keydown on the focused ID would bubble up and open the epic. A real inner button avoids both problems.

## Acceptance criteria
- [ ] Active Epics rows no longer nest interactive elements.
- [ ] Clicking the row opens the epic. Clicking the ID copies it without opening the epic.
- [ ] No nested-button console error on Dashboard load.
- [ ] A test asserts the row markup has no `<button>` inside another `<button>`.
