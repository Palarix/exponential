# Releases: edit, complete and delete from the Releases view

## What
Each open release row in the Releases view gets a `⋯` menu with **Edit**, **Complete release** and **Delete**. On Current, Complete release comes first. Complete and Delete open confirmation dialogs with a picker for where the release's issues go.

## Why
Open releases have no detail page, so their actions have to live on their rows. The server API already has update, complete and delete. Complete and delete now also take a move target (see **Move target**), so the backend changes too.

## How
- **Client and mutations:** add `updateRelease`, `completeRelease` and `deleteRelease` to `api/client.ts`, and hooks for them built on `useReleaseMutation` (it refetches the board and issues). Each success toasts the server's `messages`.
- **Row layout:** the columns are handle, title, warning, target date, Current/Next badge, `⋯` menu, chevron. The chevron is always the rightmost element, with the badge just left of the menu. Rows without a badge or menu (No release, Past, Upcoming) leave those cells empty, so the columns line up. The progress ring (and No release's count pill) moves into the title, right after "Label – Goal".
- **Row actions:** `ReleaseRowBody` gets an `actions` slot for the menu cell. `ReleaseItem` fills it with a `⋯` icon button that opens a `Menu` in a `Popover`. On Current the items are Complete release, Edit, Delete. On the other open releases they are Edit, Complete release, Delete. Delete is destructive. There is no inline Complete button.
- **Edit:** `NewReleaseDialog` takes an optional `release`. With one, it fills the fields, is titled "Edit release", its button says "Save", and it calls update. It sends all three fields. An emptied field is sent as `""`, which clears it.
- **Complete dialog** (`ReleaseActionDialog`): always asks to confirm, because completing is final. Title "Complete release?". An intro names the release in bold, with the goal when it has one: "Do you really want to complete **1.1 – Refinement**? Decide what happens to the 3 open issues currently assigned to this release." Then the picker, labelled "Move open issues to", then the help text.
- **Delete dialog** (same component): title "Delete release?". Intro: "Do you really want to delete **1.1 – Refinement**? Decide what happens to the 3 issues currently assigned to this release." Then the picker, labelled "Move issues to". The confirm button is destructive.
- Without issues to move, the intro is just the question, and there is no picker. With one issue it reads "the issue currently assigned" (or "the open issue").
- **Picker width:** the label sits on its own line, and the dropdown below it spans the dialog's width (`InlineDropdown` gets a `fullWidth` prop).
- **Dropdown style:** bordered dropdown triggers share one field style (`fieldTriggerClass` in `ui/field-trigger.ts`). It has a light wash of the text colour (`--color-field-rest`, 4%; `--color-field-hover`, 7%) and a tint border (`--color-tint-border`), so it shows on any surface without shouting. The required-but-empty state swaps the border for red. Both `InlineDropdown` and the New issue modal's label picker use it, so the Label, Status and Estimate pickers match. Chat history's borderless dropdowns are unchanged.
- **Dialog padding** is out of scope: one frame and even spacing for every dialog is xpo-c6bb65.
- **Help text** follows the picked option and names the target in bold:
  - Delete to a release: "All N issues, including done ones, will move to **1.0 (Current)**. This can't be undone."
  - Delete to No release: "All N issues, including done ones, will be taken out of the release. This can't be undone."
  - Delete to a new release: "All N issues, including done ones, will move to **a new release**. This can't be undone."
  - Delete without issues: "This can't be undone."
  - Complete: "N open issues will move to **1.0 (Current)**." (or "will be taken out of the release", or "will move to **a new release**"). Then "Done issues stay with 1.1. Completing is final: 1.1 moves to Past and can't be reopened."
  - Complete without open issues: just the second part.
- **Picker** (`InlineDropdown`, options from `moveOptions` in `releases-utils.ts`): every other saved open release in board order, labelled with the slot ("1.0 (Current)", "1.2 (Next)", "Team Mode"), then "No release". When no other saved open release exists, "A new release" takes their place. The default is the **default target** below. "No release" sends `issues: remove`. "A new release" sends no `to`. A release sends `to: <id>`.
- **Counts** come from the board's issue list: open = members that are not terminal (CANCELED and DUPLICATE count as terminal); all = every member. This matches what the server moves (it counts parents too, unlike `release.total`).
- **Move target (backend):** `CompleteRelease` and `DeleteRelease` take `(ref, mode, to)`.
  - With mode `move_to_next` and no `to`, the issues go to the **default target**: the open release right after this one, else the one right before it, else a new release (the Current placeholder, saved).
  - With a `to`, they go to that open release. A placeholder `to` is saved, and a locked Next is refused. `to` may not be the release itself or a RELEASED release. `to` with mode `remove` is an error.
  - **Server:** complete takes `{"issues", "to"}` in its body; delete takes `?issues=&to=`.
  - **MCP `release` tool:** a new `to` field for complete and delete; the `issues` description is updated.
  - **CLI:** `--to <release>` on `xpo release complete` and `xpo release delete`.
- **Row click expands:** a click anywhere on a row's header toggles its issues, like the chevron. Only the release name is a link to the filtered Issues view (open releases). On a past release the name toggles too. Clicks on the header's own buttons (handle, name, progress, `⋯`, chevron) and clicks that bubble from portals (menu, popover, dialogs) don't toggle. The No release row also toggles on a header click. The chevron stays the keyboard path.
- **Issue hover card:** the card on a Releases issue row leaves out what the row already shows (ID, title, status). It shows what the row lacks: assignee (or Unassigned), priority, estimate (when set) and labels (when any). `IssuePreview` gets a `details` variant for this. Issue links and relation chips keep the full card.

## Decisions
1. **No placeholder rows.** The view never shows placeholder rows (xpo-9a7a77: "Placeholders don't get a row"). So the "placeholders have Edit only" rule from the issue has no surface here. Every row with a menu is a saved OPEN release, and all three items are enabled. The locked-Next rule doesn't apply either.
2. **RELEASED rows** get no menu.
3. **No inline Complete button** (changed in review): completing Current is the first item of its menu instead. This replaces the issue's "inline button on the Current row".
4. **The `⋯` button is always visible,** not only on hover, so touch and keyboard users can reach it.
5. **A picker instead of move/remove radios** (changed in review). The old "move to Next" meant "the Next slot once this release is gone". After deleting Next with nothing behind it, that was a new empty release. The default is now the neighbour, and any open release can be picked.
6. **The default changes for MCP and CLI too,** so every surface agrees. An Upcoming release's issues used to go to the Next slot; now they go to its neighbour. The mode keeps its name `move_to_next`, so existing calls still work.

## Edge cases
- A release with no issues: no picker. The dialog sends the default.
- Errors (for example a duplicate label on Edit) show as the usual error toast. The dialog stays open.
- The menu and buttons must not start a drag or toggle the row.

## Acceptance criteria
- [ ] Every saved open release row has a `⋯` menu with Edit, Complete release and Delete. RELEASED rows have none.
- [ ] Current's menu lists Complete release first; the other menus list Edit first. No row has an inline Complete button.
- [ ] Edit opens the release dialog prefilled. Saving updates label, goal and target date, and an emptied field clears it.
- [ ] Complete always asks to confirm. With open issues it shows the picker, defaulting to the default target.
- [ ] Delete asks to confirm and shows the picker when the release has issues.
- [ ] The dialogs are titled "Complete release?" and "Delete release?", and open with an intro naming the release and how many issues are affected.
- [ ] Bordered dropdowns share one field style (light wash and tint border), including the New issue label picker, and stay visible on the dialog surface.
- [ ] The picker spans the dialog's width. The help text says how many issues go where (target in bold), whether done issues move, and that the action can't be undone.
- [ ] The picker lists the other saved open releases and No release, or A new release and No release when there is no other release. It never offers the release being completed or deleted.
- [ ] The server, MCP and CLI accept a `to` release. Without one, the issues go to the neighbour after, else before, else a new release.
- [ ] A click on a row header (outside the name and buttons) expands or collapses it. Only a click on the name of an open release opens Issues filtered to it.
- [ ] The chevron is the rightmost element on every row, with the `⋯` menu just left of it. The Current/Next badge sits just left of the menu. The progress ring follows the title.
- [ ] The hover card on a Releases issue row shows assignee, priority, estimate and labels, and not the ID, title or status.
- [ ] Unit tests cover the move target (Go), `moveOptions`, the dialogs and the row menu. `make test` passes.
