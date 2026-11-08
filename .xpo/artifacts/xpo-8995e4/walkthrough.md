# Walkthrough: Milestones UI/UX polish

## What was built

A polish pass on the Milestones view. It covers the six items in the issue, nine more from a headless-Chrome audit (A–I), and six rounds of review feedback. Most changes stay in `components/Milestones/`. Four touch shared UI: `Button`, `TopBar`, the popover placement and the ⌘K palette.

## How the pieces fit

### Shared UI

- **`ui/Button.tsx`** has `whitespace-nowrap` and a new `xs` size (`h-7 px-3 text-xs`). Topbar buttons are 28 px high (the filter `IconButton`, Timeline's "Everyone"). `sm` is 32 px and stood out next to them. New Milestone and New Label use `variant="secondary" size="xs"`.
- **`ui/TopBar.tsx`**: the overlap came from the side slots. They were `flex-1 min-w-0` with a zero flex basis, so they shrank below their content and the heading and buttons spilled under the search. With a center slot, the sides are now `shrink-0` at their content width and the center is `min-w-0`, so the search shrinks first. Without a center slot the sides keep `min-w-0`, because IssueDetail and MergeView put long breadcrumbs there. Each slot carries `data-slot`, which `TopBar.test.tsx` reads.
- **`ui/popover-utils.ts`** gains `top-start`: above the anchor, left-aligned, flipping below when there is no room.
- **⌘K Navigation** is built by `CommandPalette/nav-items.tsx` from `VIEWS`, so it follows the sidebar's order, names and icons. The icons moved from `Sidebar.tsx` to `Layout/view-icons.tsx`, and both use them. Cycles shows only when `cyclesEnabled`: `App` now passes the flag to the palette. Agent and Drafts stay out because the palette lists chats itself.

### Milestone rows (`MilestoneRow.tsx`)

The row header is a grid aligned to its first line (`items-start`, every cell `min-h-6`). An expanded title can then take a second line without moving the date, badge, `⋯` or chevron. The shared class strings (`ROW_HEADER`, `ROW_CELL`, `NARROW_HIDDEN`, `ROW_ICON_BUTTON`, `ROW_BODY`) are exported literals, because `react-refresh/only-export-components` rejects computed constant exports.

- **Goal line**: collapsed, the name is a `Tooltip` trigger showing the goal (capped at `max-w-sm`). Expanded, the goal is a `TitleNote` second line. The `title="Show in Issues"` attribute appears only when there is no goal tooltip, so two tooltips never compete.
- **Narrow rows**: the stack is an `@container`. Below 40rem the grid drops the warning and date columns (`@[40rem]:grid-cols-…`, cells `hidden @[40rem]:flex`), and issue rows hide their labels. The name button truncates.
- **Surfaces**: the issue list sits in `ROW_BODY`, an inset `--color-surface` well one level below the card. Issue rows hover with `--color-hover-surface`, and their `pl-7` puts the issue ID exactly under the milestone name. `MilestoneRowBody` has a `summary` slot that renders on the card above the well. A completed milestone's burndown goes there, because it describes the milestone, not its issues.
- **Completed rows** fade to 60% only while collapsed.
- **Progress pill**: a milestone with `total === 0` opens no hover card. The card's header shows only the count in the pill's mode (`mode` is passed into `MilestoneSummaryCard`).
- **Hit targets**: `ROW_ICON_BUTTON` is `p-[5px] -mx-[5px]`: 24×24 to hit, but only the icon's width in the layout.

### The stack (`Milestones.tsx`)

- **Expanded state** is a `Set<ExpandKey>`: milestone numbers plus `"none"` for No milestone. The No milestone row lost its own `useState`. `MilestonesBoard` loads it with `parseExpanded(localStorage…)` and saves on every toggle through `onExpandedChange`. Without saved state, `defaultExpanded(board)` opens a saved Current and No milestone. Numbers of deleted milestones stay in storage but match no row.
- **"No issues match."** shows when the search or filters are active and `anyMatch` finds nothing in the listed milestones or among open issues without one.
- **No milestone** has no hint: review removed it because it read like a milestone goal.
- **Empty state**: "No issues in this milestone yet." plus "Drag an issue here, or press R on an issue to move it." The R shortcut matters when No milestone is empty and there is nothing to drag.
- **Alignment with the topbar**: the content padding is `pl-[9px] pr-px`. The scroll container reserves an 11 px scrollbar gutter on both edges (`scroll-stable`). The cards therefore start at 20 px, like the heading, and end 12 px from the right, like New Milestone.

### Issue rows (`MilestoneIssues.tsx`)

The hover intent moved from the row to the title span, and `IssuePreview` anchors there with `top-start`. Before, the whole row opened the card while the pointer crossed rows, and `right-start` on a full-width row put it over the sidebar.

## Key decisions

- **One TopBar fix for every view.** Labels had the same overlap.
- **Container width, not viewport width.** The sidebar and the agent panel change the content width at the same window size.
- **⌘K derives from `VIEWS`.** The hand-kept list had already drifted ("Dashboard" against "Overview"; no Cycles, Labels or My Issues).
- **Cards align with the topbar, names do not.** Aligning the names with the heading would push the cards out past it.

## Things to know

- The topbar alignment counts on an 11 px scrollbar gutter. With overlay scrollbars (gutter 0) the cards sit 9 px in. Fixing that means changing `scroll-stable` for every view.
- Agent's "New chat" topbar button is still `sm` (32 px).
- Keyboard reorder of milestones (only `PointerSensor`) and a collapsing sidebar on narrow screens are out of scope.

## Acceptance criteria

**From the issue**
- [x] Issue list on `--color-surface`, rows hover `--color-hover-surface`, the No milestone list too. Evidence: `ROW_BODY`; screenshots at 1440 and 768 px.
- [x] The completed milestone's burndown is on the card surface, above the list. Evidence: `Milestones.test.tsx` "draws the burndown on the card, outside the issue list's lower surface"; screenshot.
- [x] New Milestone and New Label are secondary `xs` Buttons with a Plus icon. Evidence: both measured 28 px in the browser.
- [x] `Button` has an `xs` size of 28 px. Evidence: `Button.test.tsx` "has an xs size as high as the other topbar buttons".
- [x] `Button` has `whitespace-nowrap`. Evidence: `Button.test.tsx` "keeps its label on one line"; no wrapped labels from 1440 to 768 px.
- [x] No milestone has no hint. Evidence: `Milestones.test.tsx` "gives No milestone no hint".
- [x] Empty state for an expanded empty milestone. Evidence: `Milestones.test.tsx` "gives an expanded milestone without issues an empty state".
- [x] ⌘K lists Milestones with its icon. Evidence: `nav-items.test.tsx`; palette screenshot.

**Goal line**
- [x] Collapsed: name and pill only. Evidence: `MilestoneRow.test.tsx` "shows only the name and the progress while collapsed…".
- [x] The goal is a tooltip on the name. Evidence: same test (tooltip trigger on the name); tophat screenshot.
- [x] Expanded: the goal is a wrapping second line. Evidence: "shows the goal on a second line under the name while expanded".
- [x] The Dashboard, the picker and the hover card keep the inline goal. Evidence: `MilestoneTitle` is unchanged; `CurrentMilestoneSection.dom.test.tsx` passes.

**Issue hover card**
- [x] Opens only on the title. Evidence: `MilestoneIssues.dom.test.tsx` (title opens it, ID and row do not); browser check.
- [x] Opens above the title and flips below. Evidence: `Popover.test.ts` "top-start placement"; screenshot.

**A – TopBar**
- [x] No overlap at 768 px and wider; the search shrinks first. Evidence: `TopBar.test.tsx`; measured at 1440, 1024, 900 and 768 px.
- [x] Labels and the other views share the fix. Evidence: the change is in `TopBar`; Labels checked at 760 and 520 px on the tophat page.

**B – Narrow rows**
- [x] Names truncate with an ellipsis. Evidence: "truncates a long name with an ellipsis".
- [x] Below 40rem the warning and date columns and the issue labels hide. Evidence: 768 px screenshot.
- [x] At 768 px the names and more than 20 characters of each issue title stay visible. Evidence: 768 px screenshot.

**C – Search without matches**
- [x] "No issues match." for the search and for filters. Evidence: two tests in `Milestones.test.tsx`; `anyMatch` tests.

**D – Completed milestones**
- [x] 60% opacity only while collapsed. Evidence: "shows an expanded past milestone at full opacity".

**E – Command palette**
- [x] Built from `VIEWS` in sidebar order with sidebar names and icons. Evidence: `nav-items.test.tsx`.
- [x] Cycles only when enabled; Agent, Drafts and Preferences left out. Evidence: `nav-items.test.tsx`.

**F – Progress hover card**
- [x] No card for an empty milestone. Evidence: `MilestoneRow.dom.test.tsx`.
- [x] The header shows one count in the pill's mode. Evidence: "shows only the issue count in issues mode" and "…story points in points mode"; checked in both modes on the demo board.

**G – Expanded state**
- [x] Saved in `exponential-milestones-expanded` and restored. Evidence: `parseExpanded` tests; "opens the saved rows instead of the defaults".
- [x] The default is Current and No milestone. Evidence: `defaultExpanded` tests; "opens Current and No milestone by default, and nothing else".

**H – Hit targets**
- [x] 24×24 hit areas, unchanged layout. Evidence: `ROW_ICON_BUTTON`; screenshots show the same column positions.

**I – Alignment**
- [x] Issue IDs start under the milestone name. Evidence: screenshots (both at the same x).
- [x] Cards align with the topbar. Evidence: measured at 1440 and 900 px: card left at 20 px (heading 20), right gap 12 px (button 12).

- [x] `make test` passes: lint, type check, 1705 frontend tests, Go suite.
