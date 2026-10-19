# Walkthrough: Releases — list view and sidebar entry

## What was built
The web home for releases. A **Releases** entry in the sidebar (`g v`) opens a one-column view at `#/releases`. It has three parts:
- **Releases:** the open releases, with the future on top.
- **No release:** the open issues that are in no release yet.
- **Past releases:** below No release.

You reorder releases by dragging them, and sort work into them by dragging issues between rows. A small backend change makes refused release assignments explain themselves.

The design changed a lot during review (cards → rows, two panes → one column, group headers → flat lists). The spec records the final design and the reasons for it. This walkthrough describes the code as it is now.

## How the pieces fit

### Routing and data (`web/src/views.ts`, `App.tsx`, `api/*`)
- **Route:** `releases` is a view like `cycles`. `Route` gains `releaseNumber`, so `#/releases/7` opens release 7.
  - **Why the number:** it is the stable ID, and it also names placeholders, which have no `rel-` ID.
  - **Plumbing:** `AppNav` exposes `releaseNumber`/`setReleaseNumber` the same way it does `cycleId`.
- **Go-key:** `g v`. `g r` was already taken by Drafts.
- **API client:** `fetchReleases`, `fetchReleaseProgress`, `createRelease` and `reorderRelease`, with the types in `api/types.ts`. `Issue` gains `release_id` and `release`, the display name.
- **Query keys:** `releases` and `releaseProgress(ref)`. `releases` is in `SSE_INVALIDATED_KEYS`. The server notifies the issues topic on every release write, so any change from the CLI, MCP or another tab refetches the board.
- **Mutations:** `useCreateRelease` and `useReorderRelease` live in `mutations.ts`. They invalidate both releases and issues: a create or reorder can save a placeholder or move `started_at`. Issue drops reuse the existing optimistic `useUpdateIssue` with a `release_id` patch.

### The view (`web/src/components/Releases/`)
- **`Releases.tsx` — `Releases`:** loads the board. When `releaseNumber` is set, it renders the detail stub; otherwise it renders `ReleasesBoard`.
- **`ReleasesBoard`:** owns the top bar (search, `FilterButton`, "New Release"), the filters (stored with `useViewState` + `FILTERS_CODEC`, shown as `FilterChips` in the status bar), and the mutations.
- **`ReleaseStack`:** the testable core. It takes plain props and callbacks.
  - **Rows it renders:** the saved open releases in reverse order (`savedOrder` filters placeholders out), then `NoReleaseRow`, then `PastReleases`. Past shows 3 rows plus "Show N older".
  - **Shared state:** it holds the expansion set and the drag state.
- **`ReleaseRow.tsx`:** the release row and its parts.
  - **`ReleaseRowBody`:** one row on a shared `RELEASE_ROW_GRID`: leading icon, title, warning, date, progress, expand. Every row lines up as a table, including `NoReleaseRow`, which uses the same grid.
  - **`ReleaseTitle`:** the name, " – goal" and the Current/Next `SlotBadge`. Current is a solid accent pill; Next is the accent tint.
  - **`ReleaseProgress`:** the epic progress ring (`SubProgress`) in an `indicatorPill`. It opens `ReleaseSummaryCard` in a `Popover`: counts by status, done/total, the target date with its `TimeBar`, and the `Burndown`. The summary card is also the body of the detail stub.
  - **Past rows:** 60% opacity, a checkmark instead of the drag handle.
- **`releases-utils.ts`:** all decision logic, as pure functions with tests:
  - `orderWarnings`: every pair where the later release has the earlier target date. Both releases get the badge.
  - `reorderMove`: the before/after neighbour for a drop.
  - `timeBar`: elapsed fraction and whether the release is overdue.
  - `issuesInRelease` and `noReleaseIssues`: the issues of a row, matching the search.
  - `matchesIssueSearch`, `parentContext` and `childProgress`.
  - `releaseDropRef` and `acceptsIssueDrop`: what a drop sends, and whether a row accepts it.

### Drag and drop
- **One `DndContext`, two kinds of drag:** each draggable's data says `kind: "release" | "issue"`.
- **`releaseCollision`:** filters the droppables by kind.
  - **Releases** only see other saved release rows, and fall back to `closestCenter` when the pointer is between rows.
  - **Issues** see every release row and No release, using `pointerWithin` only.
- **Release drags:** a line above or below the target shows where the release lands, not a sliding sortable list. Rows differ in height once expanded, and sortable's animation looked wrong with them. The column shows the order upside down, so a release dragged upward moves *later* in the order: `highlightFor` maps `after` to the line above the target.
- **Issue drops:** send `release_id` as one of three values:
  - the target's `rel-` ID
  - `@current`/`@next` for a placeholder slot (no longer reachable in this UI, but `releaseDropRef` keeps the rule)
  - `""` for No release

  Past releases and a locked Next refuse drops (`acceptsIssueDrop`).

### Issue rows
`IssueRow`/`DraggableIssueRow` (in `Releases.tsx`) are compact versions of the backlog row, in the same order and styles: `CopyableId`, status, a muted "Parent ›" breadcrumb, the title, and the sub-issue progress pill.
- **Left out:** priority, labels and the work indicators, which crowded out the titles at half width during review.
- **The breadcrumb's tooltip** says where the parent is ("in 1.2", "no release"). The lists are deliberately flat (epic decision 11 was revised): a parent never appears as a stand-in in a list it is not in. That was the confusing part of the group-header design.
- **Why not a `button`:** the row is a `div role="button"`, because `CopyableId` is a button and buttons can't nest.
- **Hover card:** after 250 ms the row shows `IssuePreview` (the same delay as issue chips). It closes on pointer-down, so it never blocks a drag.
- **Past releases** render plain `IssueRow`s, so what a release delivered is read-only.

### Search and filters
- **Two lists:** `ReleaseStack` keeps all issues, and makes a `listed` set the filters let through.
- **Row lists** take `listed` plus the search query.
- **Counts, breadcrumbs and child progress** use all issues, so filtering never changes them.
- **`shownExpanded`:** while search or filters are active, a row with matches opens and a row without stays collapsed, but stays a drop target. When both are cleared, rows go back to the user's own expansion state.

### Backend: refusals reach the user (`internal/exponential/update.go`, `internal/server/handlers.go`)
- **Before:** `POST /api/draft` UPDATE turned every `UpdateIssue` error into "failed to update issue" with status 500.
- **Now:** `applyUpdate` wraps errors from `releaseWriter.assign` in a typed `ReleaseRefusal` (unknown ref, RELEASED target, locked Next). The handler answers those with 400 and the real message. The web app's "Couldn't save: …" toast shows it.
- **Other errors** still return 500. `TestHandleDraft_UpdateReleaseRefusalIs400` covers `@next` while locked and an unknown `#99`.

## Decisions made during review
- **No sidebar sub-items:** drag targets live in the view. The epic and xpo-974eba specs were updated.
- **Rows, not cards:** the number is dropped from row titles; a release without a label is named `#N`. Current is the louder badge.
- **One column, not two halves:** the earlier reason for the split was the filter line, and search now lives in the top bar. No release sits under Current, and Past at the bottom.
- **"New Release" in the top bar,** like Labels, instead of a dashed row. Placeholders get no row.
- **"No release",** not "Unassigned", which reads as "no assignee".
- **The name stays "Releases",** not "Versions": the backend, CLI and MCP already use "release".
- **Release chips** on backlog rows and cards were added to xpo-974eba's spec, which already owned the sidebar picker and the context menu.

## Acceptance criteria
- [x] The sidebar shows "Releases" (no sub-items); `g v` opens it. — `views.test.ts` (go keys, sidebar order, `#/releases/7` parse/format); Sidebar `VIEW_ICONS.releases`; tried on the demo board.
- [x] The top bar shows the heading, the issue search, the filter button and a "New Release" button that creates a release. — `ReleasesBoard` top bar; `NewReleaseDialog` → `POST /api/releases`; tried on the demo board.
- [x] One column shows Upcoming (last first), Next, Current, No release, then the 3 latest past releases with "Show N older". Placeholders have no row. — `Releases.test.tsx` "is one column…", "gives placeholders no row and no badge", "shows the three latest past releases and offers the rest".
- [x] Rows show the title (label or `#N` – goal), the Current/Next badge, the date and the progress pill in aligned columns; the pill opens the breakdown. — `ReleaseRow.test.tsx` (title order, `#N`, date, progress label, overdue colour); `RELEASE_ROW_GRID`; reviewed on the demo board.
- [x] Saved open releases can be reordered by drag. The order changes Current and Next. — `reorderMove` tests; `POST /api/releases/reorder`; tried on the demo board.
- [x] Open releases expand to list their issues. Issues drag from No release or an open release into another open release, or onto No release. Past releases accept no drops. — `acceptsIssueDrop`/`releaseDropRef` tests; "keeps the issues of open releases draggable"; tried on the demo board.
- [x] Past releases expand to list the issues they delivered, read-only. — "lists what a past release delivered, read-only…" (no draggable attributes on its rows).
- [x] No release starts expanded and lists open issues in no release, with its count. — "opens No release by default, with its open issues and their count"; `noReleaseIssues` tests.
- [x] Issue lists are flat, with compact rows in the backlog's style (ID, status, parent breadcrumb, title, child progress); the breadcrumb says where the parent is. — "lists each issue only where it is…", "shows a parent's child progress…", "keeps issue rows compact…"; `parentContext`/`childProgress` tests.
- [x] The search and the filter narrow the issue lists of every row; rows with matches open and list only their matches; active filters show as chips. — "opens the rows with search matches…", "narrows every list with the filters…"; `matchesIssueSearch` tests.
- [x] A refused drop shows the server's message. — `TestHandleDraft_UpdateReleaseRefusalIs400`; the error toast in `useWithErrorToast`.
- [x] A warning badge shows when target dates contradict the order. — `orderWarnings` tests; "shows the order warning with the conflicting releases".
- [x] Stats count leaf issues only. A release that was never Current has no burndown. — Counts come from the server (xpo-f89e55); "has no burndown for a release that was never Current".
- [x] Clicking a release opens `#/releases/<number>`. — Route tests; `ReleaseDetail` stub (shows "Release #N not found" for unknown numbers).
- [x] `make test` passes. — 1310 frontend tests and all Go tests, plus `go vet` and eslint.

## Notes for the next stories
- **xpo-e6ff99** replaces `ReleaseDetail.tsx`. It can reuse `ReleaseSummaryCard`, `ReleaseTitle` and the issue rows.
- **xpo-974eba** adds the "Also move its N open children?" prompt to issue drops in `ReleaseStack.onDragEnd`, and the release chip on backlog rows and cards. The chip should use the same `Package` icon as the sidebar entry.
