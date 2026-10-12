# Spec: Issue-detail property popovers use the shared Menu pickers

## What
Replace the hand-rolled status, estimate, priority and assignee popovers in `PropertySidebar` with `Menu`-based pickers. Add `aria-haspopup` and `aria-expanded` to every property trigger. Remove the duplicated keyboard navigation from `IssueDetail`.

## Why
The issue-detail sidebar is the main place where properties are edited, and screen readers get no menu semantics there. The shared pickers already have those semantics. xpo-9a7907 created them, and xpo-f16b80 and LabelPicker already moved the cycle and labels popovers onto them; status, estimate, priority and assignee were left behind.

## How
1. **`ui/AssigneePicker.tsx`** (new). It is a `Menu bare autoFocus={false}` with a `MenuFilter` ("Set assignee..."), a "No assignee" radio `MenuItem` with a dashed placeholder avatar (`<Avatar size="sm" />` with no name), one radio `MenuItem` per person with `icon={<Avatar size="sm"/>}`, `checked={person === current}` and `truncate`, and a `MenuLabel` saying "No matching people" when the list is empty. "No assignee" shows whenever the filter is empty and is checked when nobody is assigned, like "No estimate" and "No priority". It replaced "Remove assignee" after tophat feedback. Props: `people`, `current`, `onSelect(person | null)`, `onClose`. It owns its own filter text, so the sidebar no longer needs `assigneeSearch`. The filtering logic currently in the sidebar's `knownPeople` moves into the picker, and the sidebar passes the unfiltered list. Export it from `ui/index`. It is extracted from `ContextMenu`'s inline `AssigneePanel`, which now renders it too. Like the other pickers, it calls `onClose` after `onSelect`. It skips `onSelect` when the choice is unchanged, so re-picking the current person or "No assignee" doesn't save a no-op update.
2. **`PropertySidebar.tsx`**:
   - Status, estimate and priority: `Popover` → `PopoverPanel` → `<XPicker current=… onSelect={handleXChange} onClose={() => setOpenPopover(null)} />`, the same wrapper that labels and cycle use. The `PopoverHeader`s are dropped, matching how cycle and labels look. The existing `handle*Change` callbacks stay as they are.
   - Assignee: `Popover` → `PopoverPanel` → `AssigneePicker`.
   - Remove the `popoverIndex` and `setPopoverIndex` props.
   - Each property trigger `Button` (status, estimate, priority, relations, assignee, labels, cycle) gets `aria-haspopup="menu"` and `aria-expanded={openPopover === key}`, through `popoverTriggerProps(key, openPopover)` in `IssueDetail/popover-trigger.ts`. The icon-only labels "+" trigger also gets `aria-label="Add label"`.
3. **`MenuItem` radio semantics**: add a `radio?: boolean` prop to `MenuItem`. A checkable item (`checked !== undefined`) renders `role="menuitemradio"` when `radio` is set, and `menuitemcheckbox` otherwise; `aria-checked` is set in both cases. Behaviour is unchanged: radio items are still treated as checkable in `Menu`'s metadata, so `Menu` doesn't auto-close and the pickers close themselves as they do today. `SubMenu`'s first-item selector also matches `[role='menuitemradio']`. Single-choice callers set `radio`: `StatusPicker`, `PriorityPicker`, `EstimatePicker`, `CyclePicker`, `AssigneePicker`, and the assignee list in `ContextMenu`. Multi-select and toggle callers (`LabelPicker`, filters, view toggles) stay as checkboxes.
4. **`IssueDetail.tsx`**: remove the `popoverIndex` state. When a popover is open, the keyboard handler now returns early for every popover, so `Menu` handles arrows, Enter, numbers and Escape. The only exception is Escape, which keeps its existing close path. The s/e/p/a/l openers stay, minus their `setPopoverIndex` calls. `Menu` puts initial focus on the first item, not the checked one. Before this change, the sidebar started on the current value. This is accepted for consistency with the board and context menu. The ArrowUp/ArrowDown/Enter entries are removed from the issue-detail shortcut list, because `Menu` registers those keys itself.

5. **Bare `Menu` inset** (tophat feedback): bare menus had no inner padding, so the rounded item rows ran flush to the host panel's edges. Bare menus now get `p-1`. With `maxHeight`, bare menus scroll with `overflow-y-auto` instead of `scroll-stable`, because its `scrollbar-gutter: stable both-edges` doubled the side inset (seen on the assignee and cycle pickers). Framed menus keep `scroll-stable px-0`. `PopoverPanel` drops its own `py-1` when its direct child is a menu (`has-[>[role=menu]]:py-0`), so the inset is an even 4px on all sides. Non-menu panel content (`InlineDropdown`, `NewIssueModal` lists) keeps the panel padding. This affects every bare picker (sidebar, board, context menu).

6. **Checked item text** (tophat feedback): a checked `MenuItem` now uses `--color-text-primary` instead of secondary, so the current value reads as selected even when it isn't focused. The focused and active styles still take precedence.

## Testing (no DOM dependencies)
Vitest runs in the node environment. Tests use `renderToStaticMarkup` from `react-dom/server`, which is already installed via `react-dom`, and assert on the HTML string. The shared helper `src/render-markup.tsx` (`renderMarkup`, `tagsWithRole`) wraps the element in the Query, Toast and KeyboardNav providers:
- `StatusPicker`, `EstimatePicker`, `PriorityPicker`, `AssigneePicker`: the markup contains `role="menu"`; every option is `role="menuitemradio"`; exactly one option, the current value, has `aria-checked="true"`.
- `MenuItem`: renders `menuitemradio` with `radio`, `menuitemcheckbox` without it, and plain `menuitem` without `checked`. `LabelPicker` still renders `menuitemcheckbox`.
- `AssigneePicker`: "No assignee" is a radio option, checked when `current` is empty, with a dashed placeholder avatar; the empty state is rendered.
- `PropertySidebar` triggers: render with `openPopover=null` (no portal) inside a `QueryClientProvider`. Every property trigger has `aria-haspopup="menu"` and `aria-expanded="false"`. Rendering with `openPopover="status"` cannot be done server-side, because `Popover` portals into `document.body`. So the `aria-expanded="true"` case is covered by testing `popoverTriggerProps` directly.

Interaction behaviour (arrow keys, filter typing) is not tested automatically. It is checked manually in the tophat.

## Edge cases
- `Menu`'s number shortcuts replace the old status-only 1–5 handling, and now apply to estimate and priority too.
- The "Move children too?" prompt after a status change still works, because `handleStatusChange` is unchanged.

## Acceptance criteria
- [ ] The status, estimate, priority and assignee popovers render `role="menu"` with `menuitemradio` options.
- [ ] Single-choice pickers (status, priority, estimate, cycle, assignee, and the context-menu assignee list) use `menuitemradio`; labels and toggles stay `menuitemcheckbox`.
- [ ] The current value has `aria-checked="true"`.
- [ ] Every sidebar property trigger has `aria-haspopup="menu"`, and `aria-expanded` reflects the open state.
- [ ] The s/e/p/a/l shortcuts open the popovers. Arrow keys, Enter, number shortcuts and Escape work through `Menu`.
- [ ] Each picker's menu carries an `aria-label`.
- [ ] Tests use `renderToStaticMarkup`. No new devDependencies are added.
- [ ] `make test` passes.

## Decisions
1. Visual popover headers are dropped to match cycle and labels. Each picker's `Menu` gets an `aria-label` ("Change status", "Change estimate", "Set priority", "Change assignee"), so screen readers still announce the purpose. Pickers accept an optional `ariaLabel` prop that is passed through to `Menu`.
2. `menuitemradio` for single-choice pickers is in scope, at the user's request. It is implemented as an opt-in `radio` prop on `MenuItem` so that multi-select callers are unaffected.
