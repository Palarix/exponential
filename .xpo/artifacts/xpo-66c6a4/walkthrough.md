# Dashboard Active Epics: no nested buttons

## What changed
Each row in the Dashboard's Active Epics section was a `<button>`. It contained `CopyableId`, which renders its own `<button>`. HTML forbids buttons inside buttons, so React logged `In HTML, <button> cannot be a descendant of <button>` on every Dashboard load.

The row now lives in its own component, `web/src/components/Dashboard/EpicRow.tsx`. `Dashboard.tsx` renders one `EpicRow` per epic and passes an `onOpen` callback.

## How the row is built
- **Outer `<div>` with `onClick`:** a click anywhere on the row opens the epic, and the row keeps its hover background. MyIssues rows work the same way.
- **Inner `<button>` around the status icon and title:** this gives keyboard and screen-reader users a focusable control that opens the epic. Its handler calls `stopPropagation`, so the outer div doesn't call `onOpen` a second time.
- **`CopyableId` as a sibling of that button:** it is no longer inside it. `CopyableId` already stops propagation, so a click on the ID copies it and doesn't open the epic.

## Why not `role="button"` on the row div
That would make the HTML valid, but one interactive control would still sit inside another (axe rule `nested-interactive`). Also, a keydown on the focused ID would bubble up to the row's key handler and open the epic. A real inner button avoids both problems.

## Tests
`EpicRow.test.tsx` renders the row with `renderMarkup` and walks the `<button>`/`</button>` tags. The test fails if a button opens while another is still open. It covers both the normal row and the stale row, and it checks that the title sits inside a button.

## Acceptance criteria
- [x] Active Epics rows no longer nest interactive elements. Evidence: `EpicRow.test.tsx`, "does not nest the copyable ID button inside another button".
- [x] Clicking the row opens the epic; clicking the ID copies it without opening the epic. Evidence: the user checked this on the Dashboard.
- [x] No nested-button console error on Dashboard load. Evidence: the user checked this on the Dashboard.
- [x] A test asserts the row markup has no `<button>` inside another `<button>`. Evidence: `EpicRow.test.tsx`. `make test` passes: 942 frontend tests plus the Go suite.
