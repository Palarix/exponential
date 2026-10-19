# Spec: Releases — list view and sidebar entry

> **Read the spec of the parent epic xpo-4c92d4 first. It is authoritative.** The backend (API, placeholders, progress) comes from xpo-f89e55. Read its spec for the API shapes.

## What
A "Releases" entry in the sidebar (no sub-items). A Releases view with one column of rows, the future on top: the Upcoming releases (last first), Next, Current, a "No release" row with the open issues that are in no release, and the most recent past releases. The top bar has the issue search, the issue filter and a "New Release" button; search and filter narrow the issue lists of every row. Releases reorder by drag, issues drag between releases and No release, issue lists are flat with compact rows in the backlog's style, past releases expand to show what they delivered, a progress pill opens a breakdown popover, and an order warning flags target dates that contradict the order. A minimal release detail stub at `#/releases/<number>` until xpo-e6ff99 replaces it.

## Why
This is the main place to see and manage releases, and to sort work into them by hand. The release detail page (xpo-e6ff99) opens from it.

## Decisions copied from the epic
- Current and Next always exist on the server, as placeholders if needed. Placeholders always sit after the saved OPEN releases and cannot be completed, reordered or deleted.
- The display name is the label, or `#7` if there is no label (the server's `name` field).
- Stats count leaf issues only (issues with no children). The server computes them.
- Release issue lists are flat: a list shows only the issues in it. A sub-issue shows its parent, with a tooltip that says where the parent is (epic decision 11).
- The burndown shows open scope since `started_at`. A release that was never Current shows stats but no burndown.
- A warning badge marks releases whose target date contradicts the manual order. The board never reorders releases by date.
- Past lists only RELEASED releases. There is no DROPPED state.
- Only OPEN releases accept assignments. A locked Next refuses writes.

## Decisions for this story
1. **No sidebar sub-items.** No other sidebar entry has sub-items, so Releases is a plain entry like Cycles. The sidebar Current/Next drop targets (epic decision 16) move into the Releases view. The epic and xpo-974eba specs are updated to match.
2. **Name and go-key.** The feature stays "Releases": the merged backend, `xpo release`, the MCP tool and the RELEASED state all use the word, and an abandoned release is deleted, so everything in Past did ship. Go-key `g v`; `g r` stays with Drafts. Releases is always visible (no `requires`).
3. **One column, with the app's top bar pattern.** The view scrolls as one column of rows (at most `7xl` wide). The top bar follows the Labels and My Issues views: the "Releases" heading on the left, the issue search in the center, and on the right the `FilterButton` (key `f`) and a primary "New Release" button that opens the New release dialog (label, goal, target date; all optional). The view was first split into releases (left) and unplanned issues (right); one column reads as a single timeline, gives rows the full width and has one scroll. Dragging an issue up from No release scrolls the view (dnd-kit auto-scroll).
4. **The column, top to bottom:**
   - **Saved open releases in reverse order:** the last Upcoming release first, then Next, then Current. Placeholders get no row, so a Current/Next badge only appears on a release that exists. With no saved release, the column starts with No release; "New Release" creates the first one.
   - **No release** (decision 9), directly under Current: the work still to plan sits next to the release being worked on, which keeps drags short.
   - **Past releases**, newest first, at the bottom: the 3 most recent, then a "Show N older releases" toggle for the rest.
5. **Release rows.** One line, all `text-sm`, in shared grid columns so rows read as a table: leading icon, title, warning, date, progress, expand.
   - **Leading icon:** the drag handle on open releases, a checkmark on past releases, an inbox on No release.
   - **Title:** the name in medium weight (`#4` in the secondary tone until it has a label), then " – " and the goal in the muted tone, then the slot badge. Only the text truncates. Clicking the title opens the release.
   - **Slot badges:** Current is a solid accent pill with white text (the loudest); Next is the accent tint with accent text.
   - **Date:** the target date, in the warning colour once it has passed with open scope; for a past release, its release date.
   - **Progress:** the epic progress ring with done/total in a pill. A click opens a popover card: counts by status, done/total, the target date with a time bar (`started_at` or `created_at` to the target date, filled to today, warning colour when overdue), and the burndown ("Starts when this release becomes Current" before that).
   - **Past rows** are faded (60% opacity, full on hover), with no handle and no drops.
6. **Reordering.** Saved open releases drag by their handle (`@dnd-kit/core`) and drop on each other; a line shows where the release lands. The column shows the order upside down, so a release dragged upward moves later in the order. A drop calls `POST /api/releases/reorder` with `before`/`after` the target, then the view refetches (the backend sets `started_at` when a release becomes Current).
7. **Issues in releases.** Every release row expands to list its issues (open first, then done, dimmed).
   - **Open releases:** the issue rows drag into another open release or onto No release; dropping on the issue's own release does nothing. Epics move alone; the "Also move its N open children?" prompt comes with xpo-974eba.
   - **Past releases:** the list shows what the release delivered. Its rows are read-only: they open the issue and show the hover card, but do not drag. Issues archived with `xpo archive` are no longer in the issue list, so they don't show; the release's stats still come from its snapshot.
8. **Compact issue rows in the backlog's style.** Every issue list (open and past releases, No release) is flat and shows only the issues in it, as Jira's version page and Linear's lists do. A row keeps the backlog's order and styles but only what matters for sorting work into releases: `CopyableId`, status icon, the parent as a muted "Parent ›" breadcrumb, the title, and the sub-issue progress pill (ring and done/total of all direct children). Priority, labels and the work indicators (branch, chats, attachments) stay in the backlog; the hover card shows priority and assignee. The title gets the space: the breadcrumb takes at most 40% of the row and shrinks first. The breadcrumb's tooltip names the parent and where it is: "Parent: upe-000006 Cat Mood Detection · in 1.2" or "· no release"; a parent not in the issue list (e.g. archived) gets no breadcrumb. The row is a `div role="button"` because `CopyableId` is a button. A parent never shows as a stand-in in a list it is not in. (`BacklogIssueRow` itself is not reused: it is wired to the backlog's tree, popovers and keyboard focus.)
9. **No release.** A row like the release rows: an inbox icon, "No release – Open issues in no release", the count of those issues in the progress column, and the expand chevron. It starts expanded and lists open issues (not DONE, CANCELED or DUPLICATE) with no release, in the same rows as the releases; at most 100 render, and the search finds the rest. The whole row, expanded or not, is the drop target that removes an issue's release. The name is not "Unassigned", which would read as "no assignee".
10. **Search and filter narrow the issue lists.** The search (`SearchInput` in the TopBar center; `/` focuses it) matches issue titles and IDs, ignoring case. The filter is the app's `FilterButton`/`FilterMenu` (status, labels, assignee, priority, epic, worked by), stored per view with `useViewState` and `FILTERS_CODEC` as in My Issues; active filters show as `FilterChips` in the status bar. Both apply to every row's issue list (open and shown past releases, No release). While either is active, every row with a match opens and lists only its matches; rows without a match stay collapsed, and open ones stay drop targets. Release rows themselves are never hidden. Counts (the No release count, release progress) and the parent breadcrumbs and child progress still use all issues. Clearing both returns the rows to how the user left them.
11. **Issue hover cards.** Issue rows show the app's `IssuePreview` card after 250 ms, as issue chips do. It opens beside the row and closes when a drag starts. Rows highlight on hover.
12. **One `DndContext`** holds both kinds of drag; each draggable carries `kind: "release" | "issue"`, and a collision filter lets each kind see only its own targets.
13. **Release refusals reach the user.** `POST /api/draft` UPDATE answered every `UpdateIssue` error with a generic 500. Release refusals (RELEASED target, locked Next, unknown ref) are now a typed `exponential.ReleaseRefusal`, answered with 400 and the server's message; the view shows it in the "Couldn't save" toast.
14. **Routes use the number**: `#/releases/7`. The progress API accepts `#7`; placeholders are not stored, so their progress is not fetched.
15. **Detail stub:** back link, the release title, and the popover's breakdown card. xpo-e6ff99 replaces it.
16. **Order warning (client):** walk the saved open releases in order. A release is flagged when its target date is earlier than the target date of a release placed before it; that earlier-placed release is flagged too. Releases without a target date are skipped. The badge tooltip names the conflicting releases.
17. **Release queries refetch on the issues SSE topic** (the server notifies `TopicIssues` on release writes): `queryKeys.releases` joins `SSE_INVALIDATED_KEYS`.

## Flow
1. Add a `releases` view to `web/src/views.ts`: route `releases`, go-key `v`, section `main`, after Cycles. Parse and format `#/releases/<number>`.
2. Add the "Releases" entry and icon (lucide `Package`) to `web/src/components/Layout/Sidebar.tsx`, and route it in `ViewRouter.tsx`.
3. Add `release_id`/`release` to the web `Issue` type. Add the release types, API client calls (list, progress, create, reorder), query keys (`releases`, `releaseProgress(ref)`) and hooks.
4. Create `web/src/components/Releases/`: `Releases.tsx` (view, top bar, column, No release row, issue rows), `ReleaseRow.tsx` (row, title, badges, progress popover), `NewReleaseDialog.tsx`, `ReleaseDetail.tsx`, and a pure `releases-utils.ts` (order warning, time bar, reorder neighbours, issues per release, issue search, no-release issues, parent context, child progress, drop → `release_id`).
5. Return release refusals from `POST /api/draft` as 400 with the message, with a Go test.
6. Write tests: utils, route parse/format, query keys, rows, summary card and column (order, search, filters).
7. Run `make test`.

## Edge cases
- **MEDIUM:** A reorder moves a release into Current. The backend sets `started_at`. The view refetches.
- **MEDIUM:** A reorder or an issue drop fails (e.g. the release was completed meanwhile). Show the error; the refetch restores the view.
- **LOW:** No release is saved yet. The column has no open release rows; "New Release" creates the first, which becomes Current.
- **LOW:** An epic is in 1.2 and its children have no release. 1.2 lists the epic with its child progress; No release lists the children, each with a "Cat Mood Detection ›" breadcrumb whose tooltip says "in 1.2".
- **LOW:** A filter for DONE issues. Open releases with done issues and past releases open and list them; No release (open issues only) stays collapsed.
- **LOW:** A long parent title next to a long issue title. The breadcrumb truncates first, at 40% of the row; the title keeps the rest.
- **LOW:** A past release's done issues were archived. Its list shows only the issues still in the issue list (or "No issues yet." if none); its stats are unchanged.
- **LOW:** No release holds hundreds of issues. 100 render; the search narrows it. It can be collapsed.
- **LOW:** The search or filter matches nothing. No row opens, and No release stays collapsed.
- **LOW:** No release has a target date. No dates, time bars or warnings show.
- **LOW:** `#/releases/99` names no release. The stub shows "Release #99 not found" with the back link.

## Acceptance criteria
- [ ] The sidebar shows "Releases" (no sub-items); `g v` opens it.
- [ ] The top bar shows the heading, the issue search, the filter button and a "New Release" button that creates a release.
- [ ] One column shows Upcoming (last first), Next, Current, No release, then the 3 latest past releases with "Show N older". Placeholders have no row.
- [ ] Rows show the title (label or `#N` – goal), the Current/Next badge, the date and the progress pill in aligned columns; the pill opens the breakdown.
- [ ] Saved open releases can be reordered by drag. The order changes Current and Next.
- [ ] Open releases expand to list their issues. Issues drag from No release or an open release into another open release, or onto No release. Past releases accept no drops.
- [ ] Past releases expand to list the issues they delivered, read-only.
- [ ] No release starts expanded and lists open issues in no release, with its count.
- [ ] Issue lists are flat, with compact rows in the backlog's style (ID, status, parent breadcrumb, title, child progress); the breadcrumb says where the parent is.
- [ ] The search and the filter narrow the issue lists of every row; rows with matches open and list only their matches; active filters show as chips.
- [ ] A refused drop shows the server's message.
- [ ] A warning badge shows when target dates contradict the order.
- [ ] Stats count leaf issues only. A release that was never Current has no burndown.
- [ ] Clicking a release opens `#/releases/<number>`.
- [ ] `make test` passes.
