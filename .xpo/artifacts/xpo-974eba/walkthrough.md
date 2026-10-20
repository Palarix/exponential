# Walkthrough: Releases — assign issues to releases

## What was built
This story adds every way to put an issue into a release from the web app:
- A release picker in the issue sidebar, with a reopen hint for issues left in a shipped release.
- A release chip on backlog rows and board cards. The chip is a picker trigger.
- The `r` shortcut in the backlog, the board and the issue page.
- A "Release ▸" context menu item and a "Move to release…" command palette action.
- A release button in the new-issue form, which shows the release a sub-issue copies from its parent.
- One "Move sub-issues too?" prompt for epics, on every path, including drops in the Releases view.

## How the pieces fit together

### One assign path
`ReleaseAssignProvider` (`web/src/components/Releases/ReleaseAssign.tsx`) wraps the app in `App.tsx`. Its context (`release-assign-context.ts`) exposes two functions:
- `assign(issueIds, release | null)` checks for open children to move. If there are some, it shows the prompt. Then it sends the updates.
- `openPicker(issueIds)` opens a centred picker. Only the command palette uses it, because the palette has no row to anchor to.

The sidebar, the chip, the context menu, `r`, the palette, the reopen hint and Releases-view drops all call `assign`. So all of them ask about children the same way. Outside the provider the context is a no-op, so markup tests render rows and cards without it.

### Pure planning (`release-assign-utils.ts`)
The components hold no assignment logic. The utils compute it:
- `pickerOptions(board, currentId)` lists Current, Next and Upcoming. A locked Next is disabled. When `currentId` names a RELEASED release, that release comes first, disabled, so the picker still shows where the issue is.
- `childrenPrompt` finds the open descendants (any depth) that are not in the target yet, and counts the done ones. An issue whose ancestor is also assigned is covered by the ancestor, so the prompt asks once per epic.
- `planReleaseMoves` turns a choice into server moves and optimistic patches. With children, each epic sends one update with `withChildren`, and its descendants send none.
- `reopenHint` and `releaseChipTooltip` drive the sidebar hint and the chip.

### Server and mutation
`POST /api/draft` takes `with_children` next to the UPDATE payload and passes it to `UpdateIssueOpts`. The server then writes one feed entry for the epic and its children (epic decision 19). `addDraft` sends the flag. `assignReleaseOptions` patches the cache at once, rolls back on a refusal, and refetches issues and releases. A pick can save a placeholder, so the releases must refetch too.

### Picker and popovers
`ReleasePicker` (`components/Releases/ReleasePicker.tsx`) shows each release with `ReleaseTitle` from the Releases view: name, " – goal" and the Current/Next `SlotBadge`. `MenuItem` gained a `content` prop for this, and `label` stays the accessible name. `ReleasePickerPopover` wraps the picker in a popover and assigns through the provider. The backlog row, the board and the sidebar all use it.

`ReleaseChip` is a trigger button with `data-release-trigger`, like the assignee avatar on cards:
- On a backlog row, the row opens popover type `release`. The popover anchors to the chip, or to the row when the issue has no release.
- On the board, the card popover type `release` anchors to `[data-release-trigger]`, with a fallback to the card.

## Key decisions and why
- **No multi-selection.** The app has no row selection, so every path acts on one issue. The assign code takes a list of IDs, so xpo-27911a only has to pass its selection in.
- **The chip edits, it does not navigate.** An earlier version linked to the release page. In review, the chip became a picker trigger, because the row's other chips all edit.
- **`r` opens a popover, like `s`, `l`, `e`.** An earlier version opened a centred modal for rows. In review, it moved to the row popover, so all property keys feel the same.
- **"Release ▸" in the context menu.** The siblings are nouns (Status, Cycle, …), and the label never changes with the issue's state. The picker shows the state. Verbs stay in the palette and the filter placeholder.
- **The issue's shipped release shows in the picker.** Only open releases accept issues, but without that row the picker showed nothing checked for a reopened issue.
- **`r` has no conflict.** Only the Inbox binds `r`, in its own view scope.

## Non-obvious details
- **Portal events bubble through React.** `ReleasePickerPopover` stops `click` and `pointerdown` on a wrapper. Without the wrapper, a pick inside the portalled popover also clicked the row and opened the issue. It does not stop `mousedown`, so the popover's outside-click close still works.
- **Picker width.** The picker is `w-80 max-w-full`. The context menu flyout is capped at 280px with `overflow-hidden`. A fixed 320px picker made the browser scroll the flyout sideways when the filter took focus, which hid the left padding.
- **Circular imports.** `ReleasePicker` imports `Menu` from `../ui/Menu`, not from the `../ui` barrel, because `ui/ContextMenu` imports the picker.
- **New-issue form.** `form.release` is empty until the user picks, so the server copies the parent's open release. "No release" sends `none`, which wins over the copy. `IssueDraftPayload.release` keeps the choice in drafts.

## Acceptance criteria
- [x] The picker lists only OPEN releases to pick, including Current and Next placeholders, each as name – goal with the Current/Next badge. An issue in a RELEASED release sees it first, checked and disabled. Evidence: `release-assign-utils.test.ts` (`pickerOptions`), `ReleasePicker.test.tsx`; checked on the demo board.
- [x] The chip, the context menu, `r` and the command palette assign a release to the issue they act on (multi-selection: xpo-27911a). The chip and `r` open the picker as a popover. Evidence: `ReleaseChip.test.tsx`, `board-popover.test.ts`; checked on the demo board.
- [x] Changing an epic's release asks whether to move its open children. Done children never move. Evidence: `childrenPrompt` and `planReleaseMoves` tests, `ReleaseAssign.test.tsx`, `TestHandleDraft_UpdateWithChildren`, `assignReleaseOptions` tests.
- [x] A new sub-issue shows its parent's copied release. Evidence: `issue-form.test.ts` (`shownRelease`).
- [x] An open issue in a RELEASED release shows the reopen hint with "Move to Current". Evidence: `PropertySidebar.test.tsx` ("PropertySidebar release"); checked on the demo board with upe-000008.
- [x] Backlog rows and board cards of issues in a release show the release chip; a click opens the release picker. Evidence: `BacklogIssueRow.test.tsx`, `BoardCard.test.tsx`; checked on the demo board.
- [x] `make test` passes. Evidence: 1371 frontend tests and the Go suite pass.
