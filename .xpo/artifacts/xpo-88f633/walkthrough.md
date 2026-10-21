# Walkthrough: Releases — edit, complete and delete from the Releases view

## What was built
Open releases have no detail page, so their actions now live on their rows in the Releases view.
- Each saved open release has a `⋯` menu with Edit, Complete release and Delete. On Current, Complete release comes first.
- Edit reuses the New release dialog.
- Complete and Delete share one confirmation dialog. It names the release and asks where its issues should go.

Review reshaped a lot along the way:
- **Row header:** a click on a row's header expands it, and only the release name links to Issues.
- **Row layout:** the columns were reordered.
- **Hover card:** the card on issue rows stopped repeating what the row already shows.
- **Move picker:** a picker replaced the move/remove radios. That needed a backend change so that complete and delete can move issues to any open release.
- **Dropdown style:** dropdown triggers got one shared field style.
- **Dialog stacking:** a stacking bug that hid dropdowns inside dialogs was fixed (xpo-17f094).

## How the pieces fit

### Backend: a move target for complete and delete
`LocalTransport.CompleteRelease` and `DeleteRelease` (`internal/exponential/release.go`) now take `(ref, mode, to)`. The mode is still `move_to_next` or `remove`. `moveMode` checks that `to` is only used together with a move.

`releaseWriter.moveTarget(r, to)` decides where the issues go:
- **With `to`:** it resolves the ref with `ResolveRelease`. It refuses `none`, the release itself and RELEASED releases. A placeholder target is saved through `w.save`, which also refuses a locked Next.
- **Without `to`:** the target is the open release right after `r`, else the one right before it. Only when `r` is the last open release does it save a new Current placeholder.

This replaces the old rule, "the Next slot once `r` is gone". The old rule surprised the user: deleting a saved Next with nothing behind it moved its issues into a brand-new empty release, and the dialog could only say "Next (a new release)". The default changed for every surface, so the CLI, MCP and the board agree. `TestReleasesAPI_Lifecycle` was updated, because it relied on completing the last release creating #3.

The surfaces pass `to` through:
- **Server:** the complete body is `{"issues", "to"}`; delete takes `?issues=&to=`.
- **MCP:** `ReleaseToolInput.To`, with an updated `issues` description.
- **CLI:** `--to` on `xpo release complete` and `xpo release delete`.

### Frontend: the row
`ReleaseRowBody` (`ReleaseRow.tsx`) uses a 7-column grid: handle, title, warning, date, badge, actions, chevron.
- **Title cell:** `ReleaseTitle` takes a `name` slot, so the row can make only the name a button, and an `extra` slot for the progress ring. The row passes `badge={false}` and renders `SlotBadge` in its own column next to the menu. The No release and past rows fill those cells with empty spans, so every chevron lines up on the right.
- **Header clicks:** the header `onClick` toggles the row when `headerClickToggles` (`releases-utils.ts`) allows it. It ignores clicks on the header's own buttons, and clicks that bubble through React from portals. React events bubble through portals even though the DOM doesn't, so a click in the `⋯` menu or a dialog would otherwise toggle the row. The `header.contains(target)` check is what catches those.
- **Menu:** `ReleaseMenu` (`Releases.tsx`) renders `releaseMenuItems(release)` from `release-action-utils.ts`. The dialogs render outside the header.

### Frontend: the dialogs
- **Edit:** `NewReleaseDialog` takes an optional `release`. It sends all three fields, and an empty string clears one on the server.
- **Complete and delete:** `ReleaseActionDialog` handles both. Its text lives in `release-action-utils.ts` as pure functions, so it can be tested without a DOM:
  - `releaseDialogCopy` gives the title, picker label and button.
  - `releaseIntro` gives the question with the release name and goal in bold.
  - `moveHelp` gives the help text for the picked option: how many issues go where, whether done issues move, and that it can't be undone.
  - They return `HelpPart[]` (strings and `{ strong }` runs), which the dialog renders.
- **Picker options:** `moveOptions(board, release)` (`releases-utils.ts`) builds them: the other saved open releases with their slot, then No release, or "A new release" when nothing else exists. Its default mirrors the server's neighbour rule. `moveRequest` turns the choice into the API call: `none` means remove, `new` sends no `to`, and a release ID sends `to`.
- **Issue counts:** they come from the issue list (`releaseMemberCounts`), not `release.total`, because the server moves parents too.

### Shared UI changes
- **`ui/layers.ts`** puts dialogs at `z-[60]` and popovers at `z-[70]`. Popovers used to be at `z-50`, so any dropdown in a dialog opened behind it (xpo-17f094).
- **`ui/field-trigger.ts`** gives bordered dropdown triggers one look, used by `InlineDropdown` and the New issue label picker. It is a 4% wash of the text colour (`--color-field-rest`, 7% on hover) with `--color-tint-border`. It is lighter than buttons (8%), so a field reads as an input.
- **`InlineDropdown`** gained `fullWidth`. Its list matches the trigger's width, measured when the list opens, because reading a ref during render fails lint.
- **`IssuePreview`** gained a `details` variant (assignee, priority, estimate, labels) for board rows. `IssuePreviewCard` is split out so it can be tested.

## Decisions
- **No placeholder rows,** so the issue's "placeholders have Edit only" rule has no surface here. Every row with a menu is a saved OPEN release.
- **No inline Complete button:** it moved into Current's menu as the first item (review).
- **A picker, not move/remove radios** (review). The neighbour default never creates a release when another one exists.
- **The mode keeps the name `move_to_next`,** so existing CLI and MCP calls still work, even though it now means "move".
- **Dialog spacing is out of scope.** The uneven footer padding and the two dialog frames (`Modal`, `DialogPanel`) are xpo-c6bb65.

## Acceptance criteria
- [x] Every saved open release row has a `⋯` menu with Edit, Complete release and Delete; RELEASED rows have none. Evidence: `Releases.test.tsx` "gives every open release an actions menu and past releases none".
- [x] Current's menu lists Complete release first; the others list Edit first. No inline Complete button. Evidence: `releaseMenuItems` tests; "has no inline Complete button".
- [x] Edit opens the release dialog prefilled; saving updates label, goal and target date, and an emptied field clears it. Evidence: tried by the user on the demo board; the server clears on `""` (`validateReleaseLabel`/`validateTargetDate` accept empty).
- [x] Complete always asks to confirm and shows the picker with the default target. Evidence: `releaseDialogCopy`/`moveHelp` tests; tried on the demo board.
- [x] Delete asks to confirm and shows the picker when the release has issues. Evidence: same tests; the user's screenshots.
- [x] The picker lists the other saved open releases and No release (or A new release and No release), never the release itself. Evidence: `moveOptions` tests.
- [x] The server, MCP and CLI accept `to`; the default is the neighbour after, else before, else a new release. Evidence: `TestCompleteRelease_ToMovesToTheGivenRelease`, `TestDeleteRelease_ToMovesToTheGivenRelease`, `TestDeleteRelease_LastMovesToTheReleaseBeforeIt`, `TestDeleteRelease_UpcomingMovesToItsNeighbour`, `TestReleaseMoveTo_Refusals`, `TestReleasesAPI_CompleteAndDeleteMoveTo`, `TestReleaseTool_CompleteMovesTo`.
- [x] A click on a row header expands or collapses it; only the name of an open release opens Issues. Evidence: `headerClickToggles` tests; "makes only an open release's name a link".
- [x] The chevron is rightmost, the `⋯` menu just left of it, the badge just left of the menu, and the progress right after the title. Evidence: `ReleaseRow.test.tsx` order tests; "keeps the No release count in the title and its chevron rightmost".
- [x] The dialogs are titled "Complete release?" / "Delete release?" and open with an intro naming the release and the affected issues. Evidence: `releaseIntro` tests.
- [x] Bordered dropdowns share one field style, including the New issue label picker. Evidence: `field-trigger.test.ts`, `InlineDropdown.test.tsx`.
- [x] The picker spans the dialog's width, and the help text says how many issues go where, whether done issues move, and that the action can't be undone. Evidence: `MovePicker` "spans the dialog's width"; `moveHelp` tests.
- [x] The hover card on a Releases issue row shows assignee, priority, estimate and labels, and not the ID, title or status. Evidence: `IssuePreview.test.tsx`.
- [x] Unit tests cover the move target (Go), `moveOptions`, the dialogs and the row menu, and `make test` passes. Evidence: 1526 frontend tests plus the Go suite pass.
