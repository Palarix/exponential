# Walkthrough: issue-detail property popovers use the shared Menu pickers

## What was wrong
The issue was filed against the shared pickers, but they were already accessible. `StatusPicker`, `PriorityPicker`, `EstimatePicker`, `LabelPicker` and `CyclePicker` have been built on `Menu` since xpo-9a7907. The real gap was `IssueDetail/PropertySidebar.tsx`, the main place where properties are edited:
- Its status, estimate, priority and assignee popovers were hand-rolled `<button>` lists with no roles.
- Its trigger buttons had no `aria-haspopup` or `aria-expanded`.
- `IssueDetail` kept its own `popoverIndex` keyboard navigation for these popovers, duplicating `Menu`'s.

## What changed

### 1. Radio semantics in `MenuItem` (`ui/Menu.tsx`)
`MenuItem` used to render `menuitemcheckbox` for any item with a `checked` prop, so a screen reader announced the status menu as if you could tick several statuses. A new opt-in `radio` prop renders `menuitemradio` instead. `aria-checked` is set either way.

Only the rendered role changes. In `Menu`'s metadata, radio items still count as "checkable", so `activateItem` still doesn't auto-close on them. The pickers close themselves, exactly as before.

`radio` is set by the single-choice callers: `StatusPicker`, `PriorityPicker`, `EstimatePicker`, `CyclePicker` and `AssigneePicker`. Multi-select and toggle callers (`LabelPicker`, filter menus, view toggles) stay checkboxes. `SubMenu`'s first-item focus selector now also matches `[role='menuitemradio']`, or a flyout of radio items would get no initial focus.

### 2. Accessible names
Each picker takes an optional `ariaLabel`, passed to `Menu`'s `aria-label`. The defaults are "Change status", "Set priority", "Change estimate", "Change cycle" and "Change assignee". These replace the visual `PopoverHeader`s the sidebar used to show. Those headers were dropped (your call) to match how the cycle and labels popovers already looked.

### 3. `AssigneePicker` (new, `ui/AssigneePicker.tsx`)
This is extracted from `ContextMenu`'s inline `AssigneePanel`, which now just renders it. It contains:
- a `MenuFilter`,
- a "No assignee" radio with a dashed placeholder avatar (`<Avatar size="sm" />` with no name already draws one),
- one radio per person,
- a "No matching people" label when the filter matches nobody.

During tophat, "Remove assignee" became "No assignee". It now shows whenever the filter is empty and is checked when nobody is assigned, like "No estimate" and "No priority".

`select()` skips `onSelect` when the choice hasn't changed, so re-picking the current person (or "No assignee" on an unassigned issue) doesn't save an empty update. It always calls `onClose`.

### 4. `PropertySidebar`
- **Popovers:** status, estimate, priority and assignee now use the same `Popover → PopoverPanel → <XPicker>` wrapper that labels and cycle already used. The `handle*Change` callbacks are unchanged, so the "move children too?" prompt after a status change still works.
- **Assignee search:** the sidebar's `assigneeSearch` state is gone. The picker owns its filter, and the sidebar passes the unfiltered `collectKnownPeople` list.
- **Trigger attributes:** every property trigger (status, estimate, priority, relations, assignee, cycle, labels) spreads `popoverTriggerProps(key, openPopover)` from `IssueDetail/popover-trigger.ts`. That gives `aria-haspopup="menu"` and `aria-expanded`. It's a separate module so it can be unit-tested, because the open state can't be rendered server-side (see Testing).
- **Labels button:** the icon-only "+" button got `aria-label="Add label"`.
- **Props:** `popoverIndex` and `setPopoverIndex` are removed.

### 5. `IssueDetail` keyboard handler
When any popover is open, the handler now just returns: `if (openPopover) return;`. `Menu` registers its own keyboard handler at `overlay` priority, above issue-detail's `control`, so it owns arrows, Enter, number shortcuts and Escape. The labels and cycle popovers already worked this way.

The s/e/p/a/l openers stay. The issue-detail-only `handleStatusChange`, `handleEstimateChange` and `handlePriorityChange` callbacks, and the ArrowUp/ArrowDown/Enter entries in the shortcut list, are removed.

Side effects:
- Number keys now also work inside the estimate and priority popovers.
- Initial focus lands on the first item rather than the current value, the same as on the board and in the context menu.

### 6. Visual fixes from tophat
- **Inset:** a bare `Menu` (one with no frame of its own, which is how every picker renders) had no padding, so its rounded rows touched the host panel's edges. Bare menus now get `p-1`.
- **No doubled vertical padding:** `PopoverPanel` drops its own `py-1` when its direct child is a menu (`has-[>[role=menu]]:py-0`). Non-menu panel content (`InlineDropdown`, `NewIssueModal` lists) keeps the panel padding.
- **Scrolling menus:** with `maxHeight`, bare menus scroll with `overflow-y-auto` instead of `scroll-stable`. Its `scrollbar-gutter: stable both-edges` stacked on top of `p-1` and visibly widened the assignee and cycle pickers. Framed menus keep `scroll-stable px-0`, where the gutter is their side inset.
- **Checked text:** a checked `MenuItem` now uses primary text color, so the current value stands out even when it isn't focused. Focused and active styles still win.

These apply to every bare picker, including the Board and the context menu.

## Testing: no DOM dependencies
You asked for no jsdom, happy-dom or Testing Library. The new helper `src/render-markup.tsx` renders with `react-dom/server`'s `renderToStaticMarkup`, inside the Query, Toast and KeyboardNav providers. `tagsWithRole(html, role)` pulls out the opening tags for a given role.

Limits of this approach:
- Effects don't run.
- `Popover` portals into `document.body`, so it can't render server-side. Only closed-state markup of the sidebar is observable. The open case is covered by unit-testing `popoverTriggerProps`.
- Interaction (keyboard, filtering) isn't covered automatically; it was checked by hand in the tophat.

The tests are:
- `ui/Pickers.test.tsx` covers `MenuItem` roles, checked text color, `aria-label` pass-through, radio menus for status, priority, estimate, cycle and assignee (exactly one `aria-checked="true"`), "No assignee" behaviour and placeholder, and `LabelPicker` staying `menuitemcheckbox`.
- `IssueDetail/PropertySidebar.test.tsx` checks that all six triggers rendered without cycles have `aria-haspopup="menu"` and `aria-expanded="false"`, that the labels button has its accessible name, and covers `popoverTriggerProps`.

## Acceptance criteria
- [x] The status, estimate, priority and assignee popovers render `role="menu"` with `menuitemradio` options. The sidebar renders the shared pickers, and `Pickers.test.tsx` asserts `menu` plus radio counts for each picker.
- [x] Single-choice pickers use `menuitemradio`; labels and toggles stay `menuitemcheckbox`. The `radio` prop is set on the five single-choice pickers and the context-menu assignee list (via `AssigneePicker`). The `LabelPicker` test asserts no radios.
- [x] The current value has `aria-checked="true"`. `expectRadioMenu` asserts exactly one checked option, and the "No assignee" test covers the unassigned case.
- [x] Every sidebar property trigger has `aria-haspopup="menu"`, and `aria-expanded` reflects the open state. The `PropertySidebar` test checks all six closed triggers; `popoverTriggerProps` tests cover the open and closed states.
- [x] The s/e/p/a/l shortcuts open the popovers, and arrows, Enter, number shortcuts and Escape work through `Menu`. Verified manually in the tophat on the worktree dev server.
- [x] Each picker's menu has an `aria-label`. Asserted for all five single-choice pickers.
- [x] Tests use `renderToStaticMarkup`, and no new devDependencies were added. `package.json` is unchanged.
- [x] `make test` passes: lint, 624 frontend tests and the Go tests.

## Follow-ups
- xpo-dcd4f6: the backlog row assignee avatar should be an `AssigneePicker` trigger, with a dashed placeholder when unassigned. It depends on this issue.
