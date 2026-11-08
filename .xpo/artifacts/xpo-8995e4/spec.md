# Milestones: UI/UX polish

## What

A clean-up pass on the Milestones view (built in xpo-9a7a77 and xpo-974eba, renamed in xpo-d12b64). It covers the six items in the issue and nine more from a headless-Chrome audit on 2026-11-07 (A–I below). It also adds items from review: the goal moves to a second line, and the issue hover card moves to the issue title.

## Why

The audit ran on demo data at 1440, 1024, 768, 560 and 420 px. The issues it found:

- Below about 925 px the topbar search overlaps "New Milestone", and the button wraps to two lines. Below about 850 px the search also overlaps the heading. Labels has the same problem.
- At 768 px the row cuts milestone names mid-letter ("Mood Detec") with no ellipsis, and issue titles shrink to "Custom…".
- A search without matches collapses every row and says nothing.
- An expanded completed milestone stays at 60% opacity unless you hover it. The burndown and the list are hard to read.
- ⌘K cannot reach Milestones, Cycles, Labels or My Issues, and it says "Dashboard" where the sidebar says "Overview".
- Expanded rows reset on reload, and Current starts collapsed.
- The drag handle, `⋯` and the chevron are 18×18 px targets.

## Acceptance Criteria

**From the issue**
- [ ] An expanded milestone card draws its issue list on `--color-surface`, one level below the card (`--color-surface-1`). Rows hover with `--color-hover-surface`. The No milestone list uses the same surface.
- [ ] The burndown of an expanded completed milestone stays on the card surface (`--color-surface-1`), between the header and the issue list (from review).
- [ ] "New Milestone" is a `Button` with `variant="secondary"` and `size="xs"` and a Plus icon. "New Label" in Labels gets the same button.
- [ ] `Button` has an `xs` size: 28 px high (`h-7 px-3 text-xs`), the height of the other topbar buttons.
- [ ] `Button` has `whitespace-nowrap`. The topbar buttons stay on one line at every width.
- [ ] The No milestone row has no hint: only the name and the count. (The issue asked to reword the hint; review removed it.)
- [ ] An expanded open milestone with no issues shows an empty state: "No issues in this milestone yet." and, below it, "Drag an issue here, or press R on an issue to move it."
- [ ] ⌘K lists Milestones in Navigation, with the sidebar's Milestone icon.

**Goal line (from review)**
- [ ] A collapsed milestone row shows only the name, then the progress pill. The goal does not show inline.
- [ ] A pointer that rests on the name in a collapsed row shows the goal as a tooltip.
- [ ] An expanded row shows the goal as a second line under the name, in muted `text-xs`. The goal wraps and does not truncate.
- [ ] The Dashboard section, the milestone picker and the hover card keep the inline goal.

**Issue hover card (from review)**
- [ ] The issue hover card opens only when the pointer rests on the issue title. The rest of the row (ID, status, breadcrumb, labels, estimate, assignee) opens no card.
- [ ] The card opens above the title, aligned to its start (`top-start`). It flips below when there is no room above.

**A – TopBar**
- [ ] At 768 px and wider, the heading, search, filter button and "New Milestone" do not overlap. The search shrinks first.
- [ ] Labels and the other views with a TopBar get the same fix.

**B – Narrow rows**
- [ ] A milestone name that does not fit truncates with an ellipsis.
- [ ] Below a container width of 640 px, milestone rows hide the warning and date columns, and issue rows hide their labels.
- [ ] At 768 px, the milestone name and at least 20 characters of each issue title stay visible.

**C – Search without matches**
- [ ] When the search or filters match no issue, the view shows "No issues match." above the stack.

**D – Completed milestones**
- [ ] A collapsed completed milestone shows at 60% opacity. An expanded one shows at full opacity.

**E – Command palette navigation**
- [ ] The ⌘K Navigation group comes from `VIEWS` (sidebar order, the sidebar's `navLabel` and icons). It lists the `main` and `personal` views.
- [ ] Cycles shows only when cycles are enabled. Drafts and Agent stay out: the palette has its own chat items.

**F – Progress hover card**
- [ ] The progress pill of a milestone with no issues opens no hover card.
- [ ] The hover card header shows one count, in the pill's mode: "3/6 done" in issues mode, "5/13 pts" in points mode (from review).

**G – Expanded state**
- [ ] The Milestones view saves which rows are expanded in localStorage (`exponential-milestones-expanded`), and restores them on reload.
- [ ] Without a saved state, Current and No milestone start expanded.

**H – Hit targets**
- [ ] The drag handle, the `⋯` button and the chevron have a 24×24 px hit area. The row layout does not move.

**I – Alignment**
- [ ] The issue ID of each issue row starts at the same x as the milestone name.
- [ ] The milestone cards line up with the topbar (from review): the left card edge with the "Milestones" heading (20 px), the right card edge with the right edge of "New Milestone" (12 px).

- [ ] `make test` passes.

## Flow

1. Write the tests first for each step, then make them pass.
2. Add `whitespace-nowrap` to the base class of `ui/Button.tsx`.
3. In `ui/TopBar.tsx`, keep `left` and `right` at their content width. Let the `center` slot shrink (`min-w-0`, `shrink`).
4. In `Milestones.tsx` and `Labels/Labels.tsx`, replace the hand-built primary buttons with `Button` (`secondary`, `xs`, Plus icon). Add the `xs` size to `ui/Button.tsx`.
5. In `MilestoneRow.tsx`, give `MilestoneRowBody` a goal line. Collapsed: name, pill, and a `Tooltip` with the goal on the name. Expanded: the goal on a second line under the name. The other columns align with the first line.
6. Make the name button in `MilestoneRowBody` truncate (`min-w-0 truncate`).
7. Add `@container` to the stack. Hide the warning and date columns, and the issue-row labels, below `@[40rem]` (640 px).
8. Move the issue list of `MilestoneRowBody` and the No milestone list onto `--color-surface`. Give `MilestoneRowBody` a `summary` slot that renders on the card above the list, and pass `PastBurndown` there. Inset the list so the issue ID aligns with the name. Change the `IssueRow` hover to `--color-hover-surface`.
9. In `MilestoneRowBody`, apply `opacity-60 hover:opacity-100` only while the completed milestone is collapsed.
10. Replace the empty text in `MilestoneIssueList` for open milestones with the two-line empty state.
11. Remove the hint from `NoMilestoneRow`: no inline text, no second line, no tooltip.
12. In `MilestoneStack`, show "No issues match." when narrowing leaves every list empty.
13. In `MilestoneProgress`, open no hover card when `milestone.total === 0`. Pass the pill's mode to `MilestoneSummaryCard`, which shows only that count in its header.
14. Save the expanded set in localStorage. Seed it with Current and No milestone when nothing is saved. Move the No milestone expanded state into the same set (key `none`).
15. Grow the hit area of the handle, `⋯` and chevron with padding and a matching negative margin.
16. Move `VIEW_ICONS` from `Layout/Sidebar.tsx` to a shared module. Build the ⌘K Navigation group from `VIEWS` and `VIEW_ICONS`, and pass `cyclesEnabled` to `CommandPalette`.
17. In `MilestoneIssues.tsx`, move the hover intent from the `IssueRow` div to the title span. Anchor `IssuePreview` on the title with `placement="top-start"`.
18. In `Milestones.tsx`, set the content padding to `pl-[9px] pr-px`. With the 11 px scrollbar gutter on both edges (`scroll-stable`), the cards start at 20 px and end 12 px from the right, as the topbar items do.
19. Check every state on a temporary `#/tophat` page and the demo board at 1440, 1024 and 768 px. Delete the tophat page before the commit.

## Decisions

- **The goal tooltip sits on the name, not the whole row.** The row already has hover cards on the progress pill and tooltips on the warning. A row-wide tooltip would fight them.
- **The goal line only changes the Milestones view rows.** The Dashboard section, the picker and the hover card keep the inline goal: they have space, and they are not a table.
- **The issue hover card follows the title, not the row.** A row-wide trigger opened the card while the pointer moved across rows, and `right-start` on the row put it over the sidebar. On the title, above it, the card sits next to what it describes and leaves the rows below free.
- **Topbar buttons are 28 px high.** The filter and other topbar buttons are `h-7`. `sm` (32 px) stood out next to them, so `Button` gains `xs`.
- **The hover card shows the count the pill shows.** Both counts made the header long, and the milestone title truncated. The pill's click still switches between the two.
- **The burndown belongs to the milestone, not to its issues.** It summarises the whole milestone, so it sits on the card layer. Only the issue list goes one level down.
- **The cards line up with the topbar, not the names with the heading.** The old `px-6` plus the scrollbar gutter put the cards 35 px in, past the 20 px heading. Aligning the names instead would push the cards out past the heading.
- **The TopBar fix is shared, not Milestones-only.** Labels shows the same overlap. One fix in `TopBar` covers every view.
- **Narrow rows hide columns by container width, not viewport width.** The sidebar and the agent panel change the content width at the same viewport width.
- **⌘K Navigation comes from `VIEWS`.** A hand-kept list already drifted (no Cycles, Labels or My Issues, and "Dashboard" against "Overview"). Deriving it stops the drift.
- **No milestone has no hint.** Its name and the drop behaviour say enough. A hint line under it reads like a milestone goal.
- **The empty state names the R shortcut.** The drag hint alone does not help when No milestone is empty.

## Edge Cases

- MEDIUM: A saved expanded set can name milestones that were deleted or completed. The view ignores numbers it does not find.
- MEDIUM: While the search or filters narrow the lists, the saved expansion does not change (as today).
- LOW: A milestone without a goal shows no second line and no tooltip.
- LOW: "No issues match." shows only when every list is empty, including No milestone and the completed milestones shown.
- LOW: A drag that starts on the title still closes the hover card.

## Assumptions

- "Milestone description" in the review means the milestone goal. Milestones have no other description field.
- Tailwind 4 container queries (`@container`, `@[40rem]:`) are available. The project uses `tailwindcss` 4.3.

## Out of scope (separate issues if wanted)

- Reorder by keyboard (only `PointerSensor` is set up).
- A sidebar that collapses on narrow screens.
- ⌘K does not open while a text field has focus. A registry test covers this, so it can be deliberate.
- J: a fixed column for the progress pill.
- Agent's "New chat" topbar button stays `sm` (32 px).
