# Spec: Releases — assign issues to releases

> **Read the spec of the parent epic xpo-4c92d4 first. It is authoritative.** The backend comes from xpo-f89e55 and the drag and drop in the Releases view from xpo-9a7a77. Read their specs.

## What
All the ways to put issues into a release: a release picker in the property sidebar, a "Release ▸" context menu item, the `r` shortcut and a command palette action. Plus the "Also move open children?" prompt for epics (also on drops in the Releases view), the reopen hint, the release in the new-issue form, and release chips on backlog rows and board cards.

## Why
Membership is an explicit property (no cut-line markers), so assigning must be fast, and the release an issue is in must be visible where people work on issues.

## Decisions copied from the epic
- **The picker lists only OPEN releases** and always lists Current and Next. Picking a placeholder saves it (backend).
- **Copy-on-create:** a new sub-issue copies its parent's OPEN release. The picker shows the copied value, which the user can change.
- **Changing an epic's release asks "Also move its N open children?"** Done children never move. Reuse the look of the existing `moveChildrenPrompt` for status changes (xpo-f2fcb8, xpo-886099).
- **Reopen hint:** an open issue in a RELEASED release shows "In released release 1.5. Move to Current?" with a one-click action. Nothing moves automatically.
- **`r` is the main path.** Drag in the Releases view (xpo-9a7a77) is secondary. There are no sidebar drop targets.
- The display name is the label, or `#7` if there is no label.

## Decisions for this story
1. **Release chip.** An issue in a release shows a chip on its backlog row and board card: the lucide `Package` icon (the sidebar's Releases icon) in the accent colour, then the release's display name (the issue's `release` field), in a fully rounded pill with a faint border, like the row indicator pills (`indicatorPill`). Issues without a release show no chip. **The chip is a picker trigger, like the other chips on the row** (status, labels, assignee): a click opens the release picker as a popover anchored to the chip; it never opens the issue or starts a drag. Its tooltip says "Release 1.2" (and "· released" for a RELEASED release). (Review: an earlier version linked to the release page; the row's other chips all edit, so this one does too.)
2. **No multi-selection yet.** The web app has no row selection (only a focused row and a right-clicked row). The context menu acts on the right-clicked issue, `r` on the focused row or card (or the open issue), and the palette action on the open issue. The assign code takes a list of issue IDs, so multi-selection (xpo-27911a) only has to pass its selection in.
3. **One assign path.** A `ReleaseAssignProvider` in `App.tsx` owns assigning: `assign(issueIds, release | null)` computes the children prompt, shows it, and sends the updates. Every entry point (sidebar, chip, context menu, `r`, palette, reopen hint, Releases view drop) goes through it, so all of them ask about children the same way. `openPicker(issueIds)` opens a centred picker; only the command palette uses it, since it has no row to anchor to.
4. **Children prompt.** "Move sub-issues too?" lists the open descendants (any depth) that are not already in the target, says how many done ones stay, and offers Abort / Just this issue / Move all. "Move all" sends one update with `with_children` per epic, so the server writes one feed entry (epic decision 19). The web endpoint `POST /api/draft` accepts `with_children` next to `payload` for UPDATE. A selected issue whose ancestor is also selected is covered by the ancestor's prompt and is not asked about again.
5. **Picker contents.** Each row shows the release as the Releases view does (`ReleaseTitle`): the name in medium weight, " – goal" muted when there is a goal, and the Current/Next `SlotBadge`. Current and Next come first, then Upcoming; placeholders show their number (`#3`). A locked Next shows disabled with "Save Current first". "No release" shows when the issue has a release. **An issue still in a RELEASED release sees that release first, checked and disabled, with a muted "Released" pill**, so the picker always shows where the issue is; it can only move to an open release (review). A filter field narrows by name, goal, `#N` or slot. The picked release is sent as its ID, or `@current`/`@next` for a placeholder. The picker lives in `components/Releases/ReleasePicker.tsx`, beside the title and badge it reuses.
6. **Shortcut key is lowercase `r`**, like `s`, `l`, `e`, `p`, `a`. Only the Inbox binds `r` (mark all read), in its own view scope, so there is no conflict. `r` is bound in the backlog, the board and the issue page, and listed in keyboard help. **Like the other property keys, `r` opens the picker as a popover:** on a backlog row and a board card it anchors to the release chip (or to the row/card when the issue has no release); on the issue page it opens the sidebar's release picker.
7. **New-issue form.** "More options" gets a release button. Until the user picks one, it shows the parent's OPEN release (the value the server will copy). Picking "No release" for a sub-issue sends `none`, so the server copies nothing. The choice is saved in issue drafts.
8. **Optimistic update.** The moved issues (and, with "Move all", their open descendants) show the new release at once; a placeholder pick shows its name and the ID arrives with the refetch. Releases and issues refetch after the write.
9. **Naming.** The context menu item is the noun **"Release ▸"**, like its siblings (Status, Priority, Assignee, Labels, Estimate, Cycle), and it never changes with the issue's state: the picker shows the state (checkmark, "No release", the released release). Verbs stay where actions are listed: the command palette's "Move to release…" and the picker's filter placeholder "Move to release...", like the cycle picker's "Move to cycle...". (Review.)

## Flow
1. Server: accept `with_children` on UPDATE in `handleDraft` (`internal/server/handlers.go`), and `release` in `IssueDraftPayload`.
2. Pure helpers in `web/src/components/Releases/release-assign-utils.ts`: picker options and filtering, release refs, open descendants, the children prompt plan, the update plan, the reopen hint, the chip tooltip.
3. `addDraft` takes `with_children`; an `assignReleaseOptions` mutation sends the moves, patches optimistically, and refetches issues and releases.
4. `ReleasePicker` (rows from `ReleaseTitle`; `MenuItem` takes rich `content` with `label` as its accessible name) and `ReleasePickerPopover` (the picker in a popover, assigning through the provider). `pickerOptions(board, currentId)` adds the issue's RELEASED release first.
5. `ReleaseAssignProvider` (children prompt + centred picker for the palette) in `App.tsx`.
6. Property sidebar: release row with the picker, and the reopen hint.
7. Context menu: "Release ▸" (shortcut hint `R`).
8. `r` and the chip open the popover in the backlog (row popover type `release`) and on the board (card popover type `release`, anchored to `[data-release-trigger]`); `r` on the issue page opens the sidebar picker.
9. Releases view drops go through `assign`.
10. `ReleaseChip` (trigger) on backlog rows and board cards.
11. Command palette action "Move to release…" for the open issue.
12. New-issue form release button.
13. Tests first for the picker filtering and rows, the children prompt plan, the reopen hint, the chip trigger and `with_children`.
14. Run `make test`.

## Edge cases
- **MEDIUM:** The assigned issues hold an epic and some of its children. Ask the children prompt once, for the epic only.
- **MEDIUM:** A pick or drop targets a release that became RELEASED in the meantime. The server rejects it. Show the error (toast) and roll back.
- **MEDIUM:** An issue sits in a RELEASED release. The picker shows it checked and disabled; picking it is impossible.
- **LOW:** The `r` key conflicts with an existing shortcut. Only Inbox uses it, in its own view.
- **LOW:** A long release label or goal. The chip and the picker row truncate it; the badge stays visible.

## Acceptance criteria
- [ ] The picker lists only OPEN releases to pick, including Current and Next placeholders, each as name – goal with the Current/Next badge. An issue in a RELEASED release sees it first, checked and disabled.
- [ ] The chip, the context menu, `r` and the command palette assign a release to the issue they act on (multi-selection: xpo-27911a). The chip and `r` open the picker as a popover.
- [ ] Changing an epic's release (picker, chip, menu, `r`, palette or a drop in the Releases view) asks whether to move its open children. Done children never move.
- [ ] A new sub-issue shows its parent's copied release.
- [ ] An open issue in a RELEASED release shows the reopen hint with "Move to Current".
- [ ] Backlog rows and board cards of issues in a release show the release chip (Package icon + name); a click opens the release picker.
- [ ] `make test` passes.
