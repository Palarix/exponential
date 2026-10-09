# Spec: Refactor App into Layout → Sidebar + ViewRouter → View hierarchy

## What
Break up `web/src/App.tsx` (543 lines). Today it owns state for every view, renders every view inline, and passes 10+ props into each one. After this change:

- **App** owns only global UI concerns: the route, overlays and global shortcuts.
- **Server data** moves into TanStack Query: issues, config, inbox and user.
- **A `views` registry** is the single source of truth for view metadata.
- **`Sidebar`** is extracted from Layout and driven by that registry.
- **`ViewRouter`** maps the route to a view (or to IssueDetail).
- **Each view** owns its tab, filter and sort state, and reads data from query hooks instead of props.

Intended behavior changes (all agreed):
- IssueDetail prev/next follows the order of the view the issue was opened from.
- xpo-0c0556 and xpo-787e7b are fixed.

Otherwise there is no visual or behavioral regression.

## Why
This is the capstone of epic xpo-570a07. The leaf components (Tabs, SearchInput, IconButton, FilterButton, CountBadge, ViewTitle, ViewContainer) are extracted; this story removes the god component above them. TanStack Query replaces the hand-rolled data layer App currently provides:
- `fetchData`
- the `lastJsonRef` JSON dedupe
- `patchIssue`
- SSE refetch
- 10+ drilled props

## Current coupling the design must handle
1. **IssueDetail replaces the view.** When `selectedIssueId` is set, the active view **unmounts**, so view-local state must outlive the component. Today `backlogTab` lives in App and is not persisted.
2. **Prev/next order.**
   - It always comes from `backlogNavOrder`, reported by Backlog.
   - The fallback is `sortIssuesWithinGroups(issues, sortKey)`.
   - It applies even when the issue was opened from Board, MyIssues or Cycles.
3. **Status bar `FilterChips`** are computed in App from the Backlog, MyIssues and Inbox filters.
4. **Labels → Backlog** mutates `backlogFilters` and assigns an invalid `#/backlog` hash (xpo-787e7b).
5. **Route params** `selectedCycleId` and `depFocusId` live in the hash, so they are router state.
6. **View metadata is duplicated** across `View` (declared twice: App and Layout), `VIEW_LABELS`, `VIEW_ROUTES`, `ROUTE_VIEWS`, `GO_TARGETS` and Layout's NavItem list. Nav labels differ from title labels ("Overview" vs "Dashboard", "Notifications" vs "Inbox").
7. **Server data is fetched ad hoc.** `fetchUser` runs in Layout, Inbox and MyIssues. Issues and inbox are refetched on SSE through App.

## Design

### Target tree
```
QueryClientProvider
└── App                     route state, overlays, global shortcuts, nav-order, view-state store, label contexts
    └── Layout              chrome: sidebar column + main + status bar
        ├── Sidebar         registry-driven NavItem[] + header buttons, ProjectSelector, user
        ├── <main>
        │   └── ViewRouter  loading / error / IssueDetail / not-found / active view
        │       └── <View>  owns local state; ViewContainer → TopBar + content
        └── StatusBar       connection + version, and a portal slot views fill
```

### 1. TanStack Query core — `web/src/api/queries.ts` (new)
- **Dependency:** add `@tanstack/react-query` (v5) with `bun add`. `main.tsx` wraps the app in `QueryClientProvider`.
- **Client defaults preserve today's behavior:**
  - `retry: false` keeps "Server Offline" immediate, with no roughly 7s of retry backoff.
  - `refetchOnWindowFocus: false`, because SSE drives freshness.
  - `staleTime: Infinity`.
- **Query keys** live in one `queryKeys` object: `issues`, `config`, `inbox`, `inboxStatus`, `user`.
- **Hooks:**
  - `useIssues()`: `fetchIssues`. Structural sharing replaces `lastJsonRef`.
  - `useConfig()`: `fetchConfig`, derived through `deriveAppConfig` into `prefix`, `version`, `projectName`, `contributors`, `cyclesEnabled`, label colors and default labels. It stays best-effort: errors are ignored and defaults apply.
  - `useInbox()`: items and status, best-effort as today. `useMarkInboxRead()` calls `markInboxRead` and then `setQueryData(inboxStatus)`, with the same toasts.
  - `useUser()`: one cached query for the Sidebar, Inbox and MyIssues.
  - `usePatchIssue()`: `setQueryData(issues, …)`, replacing `patchIssue`.
  - `useRefreshIssues()`: `invalidateQueries(issues)`, which returns a promise so `await refresh()` still works. It replaces `fetchData`/`onRefresh`.
  - `useSetConfigLabels()`: `setQueryData(config, …)`, replacing `setConfigLabels`.
- **SSE:** on each event, `useSSE` invalidates `issues` and `inbox` (which covers `inboxStatus`), the same set as today.
- **Out of scope:**
  - The remaining fetches (cycles, metrics, activity, timeline, instances, IssueDetail data) move in xpo-ec0480.
  - Mutations and `onRefresh` removal move in xpo-6e757d.
  - Leaf components keep their `onRefresh`/`patchIssue`/`contributors` props. View roots supply them from the hooks.

### 2. `views` registry — `web/src/views.ts` (new, pure)
```ts
export type ViewId = 'dashboard' | 'inbox' | 'backlog' | 'board' | 'cycles' | 'dependencies' | 'labels' | 'my-issues' | 'timeline';
interface ViewDef { id: ViewId; route: string; title: string; navLabel: string; goKey: string; section: 'main' | 'personal'; requires?: 'cycles' }
export const VIEWS: readonly ViewDef[]  // sidebar order
export type Route = { view: ViewId; issueId: string | null; cycleId: string | null; depFocusId: string | null };
export function parseRoute(hash: string): Route
export function formatRoute(route: Route): string
```
- Derived maps replace `GO_TARGETS`, `VIEW_LABELS`, `VIEW_ROUTES` and `ROUTE_VIEWS`.
- Icons stay in the Sidebar (`Record<ViewId, ReactNode>`), so this file has no JSX.
- **Test-first:**
  - every hash form in `parseHash` today, including unknown routes → dashboard
  - `formatRoute` round-trips
  - invariants: unique ids, routes and go keys

### 3. `Sidebar` — `components/Layout/Sidebar.tsx` (new)
- Moves out of Layout: `NavItem`, `ProjectSelector`, the collapse state, the header buttons (search, new issue, collapse) and the user footer (now via `useUser()`).
- Renders `VIEWS`, filtered by `requires` and split by `section` with the existing divider.
- Props: `activeView`, `onViewChange`, `onSearch`, `onNewIssue`, `cyclesEnabled`, `inboxUnread`.
- Layout keeps the main column and the status bar.

### 4. `useAppNav()` — `web/src/app/contexts.ts` + `web/src/app/hooks.ts` (new)
UI navigation state that isn't server data:
- `navigate(view, intent?)`: clears the issue, cycle and dep focus, clears the published nav order, and stores the one-shot `intent`.
- `openIssue(id)`
- `newIssue()`
- `cycleId` with `setCycleId`, and `depFocusId` with `setDepFocusId`
- `useIssueNavOrder(ids)`: a hook views call to publish their visible order.
- `useNavIntent(view, apply)`: applies the pending intent for that view once, after mount.
- `useIssueClick()`: a stable `onIssueClick` that calls `openIssue`.

### 5. Per-view prev/next order
- **Rule:** the active list view publishes its visible issue order through `useIssueNavOrder`. IssueDetail walks the order published by the view the issue was opened from.
- **Which views publish what:**
  | View | Order published |
  |---|---|
  | Backlog | the existing visible order (replaces `onNavigationOrderChange`) |
  | Board | columns left→right, cards top→bottom, skipping collapsed and hidden columns |
  | MyIssues | the visible list order for the active tab |
  | Cycles | the issue list order in cycle detail |
- **Fallback:** when no order is published, or the issue isn't in it (Dashboard, Timeline, Dependencies, a deep link on a fresh load), `ViewRouter` uses `sortIssuesWithinGroups(issues, sortKey)`, reading the sort through `useViewState(SORT_STATE_KEY, SORT_CODEC)`. This is today's fallback.
- Inbox embeds its own IssueDetail and is unaffected.

### 6. `useViewState` — colocated state that survives unmounts
- `useViewState<T>(key, initOrCodec)` is backed by an App-scoped in-memory `Map` (context).
  - **With an initializer:** the value survives open issue → back, but not a reload. This matches today's `backlogTab`.
  - **With a codec:** it also reads and writes `localStorage` under the **existing keys**, so there is no migration. Accesses are wrapped in try/catch.
- **Test-first:** store and storage semantics live in a pure `view-state-utils.ts`.

### 7. `StatusBarSlot`
- Layout renders the status bar's left area (`flex-1 min-w-0`) as a portal target exposed through context.
- Views render `<StatusBarSlot><FilterChips …/></StatusBarSlot>`.
- An empty slot is laid out the same as today's spacer.

### 8. `ViewRouter` — `components/Layout/ViewRouter.tsx` (new)
- Holds the lazy imports and a `Record<ViewId, ComponentType>`. Views take **no props**. Backlog stays eager.
- Renders, in order:
  1. loading (`useIssues().isPending`)
  2. error ("Server Offline"; **Retry** calls `refetch()`)
  3. `IssueDetail`
  4. "Issue not found"
  5. the active view
- The blocks move here verbatim. The `ErrorBoundary` (`onReset` → refresh) and `Suspense` wrapping are unchanged.

### 9. Per-view state moves
| State | From | To | Persistence (keys unchanged) |
|---|---|---|---|
| `backlogTab` | App | Backlog | memory |
| `backlogFilters` | App | Backlog, **loaded per tab on switch** (xpo-0c0556) | `exponential-backlog-filters-${tab}` |
| `sortKey` | App | Backlog | `exponential-sort` |
| `backlogNavOrder` | App | Backlog → `useIssueNavOrder` | memory |
| `myIssuesTab` / `myIssuesFilters` | App | MyIssues | existing keys |
| `inboxFilters` | App | Inbox | `exponential-inbox-filters` |
| `selectedCycleId` / `depFocusId` | App | App (route) via `useAppNav` | hash |

**Labels → Backlog** (xpo-787e7b):
- Labels calls `navigate('backlog', { filters: { labels: [label] } })`.
- Backlog applies it with `useNavIntent('backlog', …)` on mount, through its normal, persisted filter setter on the active tab.
- The stray hash assignment is removed.

### 10. App after the refactor
App keeps:
- route state with the `hashchange` and `document.title` effects
- overlays (NewIssueModal, CommandPalette, KeyboardHelp)
- global shortcuts (palette, `c`, and `g`+key generated from `VIEWS`)
- `useSSE` → invalidation
- the providers: label contexts, which now come from `useConfig()`, AppNav and the view-state store

Target size: under 200 lines, with no view imports.

## Acceptance criteria
- [ ] `App.tsx` imports no view component and renders no view-specific content.
- [ ] App holds no server data in `useState`. Issues, config, inbox and user come from TanStack Query hooks; `fetchData` and `lastJsonRef` are gone.
- [ ] SSE events refresh issues and inbox, and the "Server Offline" screen appears immediately on failure, with a working Retry.
- [ ] Each view owns its tab, filter and sort state through `useViewState`. App has no `backlog*`, `myIssues*`, `inboxFilters` or `sortKey` state.
- [ ] `ViewRouter` maps `ViewId` → component through a record, not a conditional chain.
- [ ] Every view renders through `ViewContainer` → `TopBar` + content, including Inbox.
- [ ] `VIEWS` is the single source of view metadata for the Sidebar, go-shortcuts, routes and titles. The command palette keeps its curated subset, typed as `ViewId`.
- [ ] Filter chips appear in the status bar for Backlog, MyIssues and Inbox as before.
- [ ] Persisted state survives reload under the existing keys. Backlog's tab, filters and sort survive open issue → back.
- [ ] IssueDetail prev/next follows the originating view's visible order: Backlog, Board, MyIssues and Cycles detail. Other views and deep links use the default sorted order.
- [ ] xpo-0c0556: a Backlog tab switch loads that tab's filters.
- [ ] xpo-787e7b: a label click lands on `#/issues` with the label filter applied and persisted.
- [ ] Deep links work: `#/issues/<id>`, `#/cycles/<id>`, `#/dependencies/<id>` and every view route.
- [ ] Every keyboard shortcut still works: the global ones, `g`+key, view-local, `[`/`]`, `/`, and the help overlay listing.
- [ ] The new pure modules were unit-tested first: `views.ts`, `view-state-utils.ts`, nav-order resolution and the config/patch helpers in `query-utils.ts`.
- [ ] `make test` passes; the user tophats every view.

## Flow
1. **Registry.** Test-first `views.ts`.
2. **Pure helpers.** Test-first `view-state-utils.ts`, `nav-order-utils.ts` and `query-utils.ts`.
3. **TanStack Query.** Add the dependency, the provider and `queries.ts` hooks.
4. **Infrastructure.** Add the `app/` contexts and hooks (AppNav, view state, nav order, intent) and `StatusBarSlot`.
5. **Sidebar and Layout.** Extract `Sidebar` from Layout, driven by `VIEWS`. Layout exposes the status-bar slot.
6. **Migrate views** to hooks and owned state, deleting their props:
   - Dashboard, Timeline, Dependencies, Labels (intent) and Cycles (nav order)
   - Board (nav order)
   - Inbox (plus ViewContainer and its filter chips)
   - MyIssues
   - Backlog (tab, filters, sort, nav order, intent, the xpo-0c0556 fix)
7. **ViewRouter and App.** Move the loading, error, IssueDetail and not-found blocks and the lazy imports into `ViewRouter`, and slim App down.
8. **Verify.** Run `make test`, then exercise every view, shortcut and deep link in the browser.

## Decisions
- **TanStack Query for core server data, not a hand-rolled `useAppData()` context.** It provides caching, dedupe, invalidation and structural sharing. The scope is limited to the data App owns today; the rest is in xpo-ec0480 and xpo-6e757d.
- **Conservative query defaults** (no retry, no refetch on focus, infinite stale time). They keep today's behavior because SSE is the freshness signal. They can be revisited in the follow-ups.
- **An in-memory store for colocated state, not keeping hidden views mounted.** Hidden views would keep their keyboard registrations, DnD sensors and listeners live.
- **Existing localStorage keys are reused verbatim.**
- **A portal slot for the status bar.** Lifting filter state back up would undo the colocation.
- **Per-view nav order through publication by the active view** (user decision). Order capture happens implicitly, so it needs no change at click sites.
- **The label-click filter persists like a manual filter** (user decision).

## Edge cases
- **HIGH:**
  - Backlog → open issue → back keeps the tab, filters and sort. The `useViewState` store covers this.
  - A deep link to `#/issues/<id>` on a fresh load has a working prev/next through the fallback order.
  - `retry: false` must hold, or the initial offline screen is delayed.
- **MEDIUM:**
  - A Board order that skips collapsed or hidden columns means prev/next can't reach those cards. That matches what's visible.
  - A nav intent is consumed once and never reapplied when you later return to Backlog.
  - A refetch error after a successful load shows "Server Offline", as today (`isError` with stale data).
- **LOW:**
  - Title and nav labels keep their differing text.
  - The status bar slot is empty for other views.
  - The cycles route still resolves when cycles are disabled, as today.

## Assumptions
- No view needs another view's local state beyond the cases above.
- Inbox can adopt `ViewContainer` as a pure shell swap.
- No React-rendering test infrastructure (testing-library or jsdom) will be added. Logic goes into pure, tested modules, and wiring is verified manually and with chrome-devtools.

## Implementation notes (decided during implementation)
- **`useViewState(key, initOrCodec)`:** pass an initializer for session-only state, or a codec for persisted state. The codec supplies the fallback, so there is no separate `init`. Codecs live next to their state: `FILTERS_CODEC` in `filters.ts`, `SORT_CODEC`/`SORT_STATE_KEY` in `utils/sort.ts`.
- **Nav order is published into a ref, not state.** App snapshots it into state when an issue opens (`openIssue`, or a `hashchange` that adds an issue id). Board republishes on every drag-over, and a state-backed order would re-render the whole tree during drags. The snapshot is exactly "the originating view's order".
  - The order is cleared by `navigate()` and by a `hashchange` that changes the view.
  - The fallback order is computed in `ViewRouter`'s `IssueRoute`, so App never touches the sort.
- **Board's order includes cards behind "+N more"** in a column, because they are part of that column's order.
- **Backlog stays eagerly imported**, as before. The other views stay lazy.
- **Help listing order for "Go to …" shortcuts** now follows sidebar order, since it is generated from `VIEWS`. Previously it followed the hand-written `GO_TARGETS` order.
- **`Loading`, `ServerOffline` and `IssueNotFound`** are local components in `ViewRouter.tsx`, with their markup unchanged.
- **Bug found and fixed:** xpo-1657f1. The ActivityTimeline UPDATE summary rendered unkeyed fragments, which logged a React key warning; it showed up while verifying in the browser.
