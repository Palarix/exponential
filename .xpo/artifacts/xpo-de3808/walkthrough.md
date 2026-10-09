# Walkthrough: App → Layout → Sidebar + ViewRouter → View

This is the capstone of epic xpo-570a07. Before it, `App.tsx` was a 543-line god component:
- It fetched every piece of server data.
- It held the tab, filter and sort state of three different views.
- It rendered all nine views inline through a conditional chain.
- It passed each view 10+ props.

After it, App is 188 lines and owns only global UI concerns: the route, overlays, global shortcuts and providers. Everything else moved to where it belongs.

```
main.tsx: QueryClientProvider
└── App                 route, overlays, global shortcuts, nav-order snapshot, providers
    └── Layout          chrome + status-bar portal slot
        ├── Sidebar     registry-driven nav, ProjectSelector, user
        └── ViewRouter  loading / offline / IssueDetail / not-found / active view
            └── <View>  no props; reads hooks, owns its local state
```

Three issues were fixed along the way: xpo-0c0556 and xpo-787e7b (both planned) and xpo-1657f1 (found in the browser).

---

## 1. Server data: TanStack Query (`api/queries.ts`)
The user proposed TanStack Query instead of the hand-rolled `useAppData()` context the first draft of the spec described. We scoped it to the data App owned: issues, config, inbox and user. Follow-ups xpo-ec0480 (per-view fetches) and xpo-6e757d (mutations, removing `onRefresh`) are filed.

What it replaced:

| Before (App) | After |
|---|---|
| `fetchData` + `useState(issues)` + a `lastJsonRef` JSON-compare | `useIssues()`. Structural sharing keeps references stable when nothing changed. |
| `patchIssue` doing `setIssues(prev => …)` | `usePatchIssue()` → `setQueryData` |
| `onRefresh={fetchData}` | `useRefreshIssues()` → `invalidateQueries`. It returns a promise, so `await refresh()` in NewIssueModal still waits for the refetch. |
| eight `useState`s filled from `fetchConfig` | `useConfig()` → `deriveAppConfig(raw)` |
| `setConfigLabels` | `useSetConfigLabels()` → `setQueryData(config)` |
| inbox items, last read and unread state + `fetchInboxData` | `useInbox()`, `useMarkInboxRead()` |
| `fetchUser` called separately in Layout, Inbox and MyIssues | `useUser()`, one cached query |
| SSE handler calling `fetchData()` + `fetchInboxData()` | `useInvalidateOnServerEvent()`. The `inbox` prefix also covers `["inbox", "status"]`. |

**The client defaults matter.** `createQueryClient()` sets:
- `retry: false`
- `refetchOnWindowFocus: false`
- `staleTime: Infinity`

SSE is this app's freshness signal, so TanStack's own polling heuristics would only add requests. More importantly, the default `retry: 3` with backoff would delay the "Server Offline" screen by several seconds. Keep these defaults unless a follow-up deliberately changes them.

The derivation logic (`deriveAppConfig`, `effectiveLabelColors`, `withConfigLabels`, `patchIssueList`) lives in a pure, tested `query-utils.ts`. `queries.ts` is only thin hook wiring.

## 2. One registry for views (`views.ts`)
View metadata used to be spread across:
- two `View` union types (App and Layout)
- `VIEW_LABELS`, `VIEW_ROUTES`, `ROUTE_VIEWS` and `GO_TARGETS`
- a hand-written NavItem list

Now `VIEWS` is one array in sidebar order. Each entry has `id`, `route`, `title` (document title and "Go to …" labels), `navLabel` (sidebar text), `goKey`, `section` and an optional `requires: "cycles"`.

- `title` and `navLabel` are separate because they really do differ ("Dashboard" vs "Overview", "Inbox" vs "Notifications").
- `parseRoute`/`formatRoute` replace `parseHash`/`setHash`. They are pure and tested, including the rule that unknown routes fall back to dashboard and a round trip for every view.
- Icons stay in `Sidebar.tsx` so the registry has no JSX.
- The command palette deliberately keeps its own curated six-view list, now typed as `ViewId`.

## 3. Layout, Sidebar and the status-bar slot
- **`Sidebar.tsx`** is lifted out of Layout essentially verbatim: header buttons, `ProjectSelector`, the collapse state and the user footer. The nav is now `VIEWS.filter(section, requires).map(NavItem)`, with the divider between the `main` and `personal` sections.
- **`Layout.tsx`** is just chrome: the sidebar, `<main>` and the status bar.
- **The status bar used to receive `statusBarLeft`**, which App computed from three views' filter state. That is exactly the coupling this story removes. Layout now renders an empty `flex-1 min-w-0` div, which is the same box as before, and publishes it through `StatusBarSlotContext`. Views render `<StatusBarSlot><FilterChips/></StatusBarSlot>`, a `createPortal` into that div.
- In MyIssues and Backlog the slot is a **sibling** of `ViewContainer`, in a fragment, not inside its content div. Portals don't escape React event bubbling, and chip clicks shouldn't reach the list's handlers.

## 4. Colocated state that outlives its view (`useViewState`)
**The core problem:** when an issue is selected, `ViewRouter` renders IssueDetail *instead of* the view, so the view unmounts. Plain `useState` in Backlog would lose the tab every time you opened an issue. App used to sidestep this by owning the state.

**The fix** is an App-scoped store (`createViewStateStore`, a `Map` + listeners) exposed through `ViewStateContext` and read with `useSyncExternalStore`:

```ts
useViewState(key, () => init)  // session-only: survives unmount, not reload
useViewState(key, CODEC)       // also persisted to localStorage under `key`
```

- **The codec owns the fallback** (`parse(null)`), so persisted state needs no separate initializer. Codecs sit next to the state they describe: `FILTERS_CODEC` in `Backlog/filters.ts`, `SORT_CODEC`/`SORT_STATE_KEY` in `utils/sort.ts`.
- **Keys are the existing localStorage keys**, so users keep their saved filters, sort and My Issues tab with no migration.
- **Keys can be dynamic.** Backlog uses `` `exponential-backlog-filters-${activeTab}` ``. When the tab changes, the hook simply reads a different key, so the right filters load. That is the whole fix for **xpo-0c0556**: App used to load the `all` key once and then save to per-tab keys.
- Storage access is wrapped in try/catch, so a private window or blocked storage degrades to memory-only.
- **Not chosen:** keeping hidden views mounted. It would also preserve scroll position, but hidden views would keep keyboard registrations, DnD sensors and native listeners live, which is too risky for a no-regression refactor.

## 5. Navigation (`app/contexts.ts`, `app/hooks.ts`)
`AppNavContext` carries UI navigation state that isn't server data: `view`, `navigate`, `openIssue`, `closeIssue`, `newIssue`, the `cycleId`/`depFocusId` route params and their setters, `publishNavOrder` and `takeIntent`.

Convenience hooks:
- `useIssueClick()`: a stable `(issue) => openIssue(issue.id)`
- `useIssueNavOrder(ids)`
- `useNavIntent(view, apply)`

### Prev/next follows the originating view
Before, IssueDetail always walked Backlog's order, even when the issue was opened from Board. The user chose per-view order.

- Backlog, Board, MyIssues and Cycles detail call `useIssueNavOrder(memoizedIds)`:
  - **Board:** visible columns, skipping collapsed ones, flattened. That includes cards behind "+N more", which are still in that column's order.
  - **Cycles:** the `DETAIL_STATUS_ORDER` groups, flattened.
- **The order is published into a ref, not state.** Board's containers change on every drag-over, and state-backed publishing would re-render App, Layout and Sidebar on each one.
- **App snapshots the ref into state at the moment an issue opens**, through `openIssue`, or a `hashchange` that adds an issue id, such as browser forward. The snapshot is exactly "the order of the view you clicked in". The view unmounts afterwards, and prev/next keeps working off the snapshot.
- **The ref is cleared** by `navigate()` and by a `hashchange` that changes the view, so a stale order from a previous view never leaks.
- **Fallback:** `resolveNavOrder(published, fallback, issueId)` (pure, tested) uses the published order only if it contains the issue. Otherwise it uses `sortIssuesWithinGroups(issues, sortKey)`, which is today's fallback. That covers deep links, Dashboard and Timeline. `ViewRouter`'s `IssueRoute` computes it and reads the sort through `useViewState(SORT_STATE_KEY, SORT_CODEC)`, so App never touches Backlog's sort.

### One-shot intents: Labels → Backlog (xpo-787e7b)
- **Before:** Labels' `onLabelClick` mutated App's `backlogFilters` and assigned `#/backlog`. That is not a route; it only worked because the `setHash` effect overwrote it before the queued `hashchange` read it.
- **Now:** Labels calls `navigate("backlog", { filters: { labels: [label] } })`, and `navigate` stores the intent in a ref map. Backlog's `useNavIntent("backlog", …)` takes it in an effect after mount and applies it through the normal persisted filter setter, so it behaves exactly like a filter the user picked (user decision).
- `takeIntent` deletes as it reads. StrictMode's double effect and later visits to Backlog therefore never reapply it.

## 6. `ViewRouter`
- **One record maps views to components:** `Record<ViewId, ComponentType>`. The views are lazy imports, except Backlog, which stays eager as before.
- **It renders, in order:**
  1. loading (`isPending`)
  2. Server Offline (`isError`; Retry → `refetch()`)
  3. `IssueRoute` (IssueDetail with prev/next, or "Issue not found")
  4. the active view
- The markup of the three state screens moved verbatim into local components.
- `ErrorBoundary` and `Suspense` wrap everything, as they used to wrap App's `renderContent()`.

## 7. App
What's left is easy to scan:
- route state with the `hashchange` and `document.title` effects
- `openIssue` and `navigate`
- the `nav` memo
- global shortcuts, where the `g` chords are generated from `VIEW_BY_GO_KEY`, so the help overlay now lists them in sidebar order
- the three overlays
- the provider stack: label contexts from `useConfig()`, the view-state store and AppNav

`hashchange` compares routes structurally (`sameRoute`). Without that, the echo of our own `formatRoute` write would set a new, identical object and cause an extra render.

## 8. Per-view migration
Each view root swapped its props for hooks, keeping the old local variable names (`onIssueClick`, `onRefresh`, `contributors`…) so the bodies didn't change. Leaf components (ContextMenu, pickers, PropertySidebar, IssueDetail) still take props; view roots feed them from hooks until xpo-6e757d.

- **Inbox** additionally moved its list panel onto `ViewContainer` (`className="w-80 xl:w-96 shrink-0 border-r …"`). It was the last hand-rolled TopBar shell.

## Bug found: xpo-1657f1
`ActivityTimeline` joined multi-field UPDATE fragments with keyed `" and "` separators, but the fragments themselves had no keys, so React logged a key warning. It surfaced in the console during browser verification. Each part is now wrapped in `<Fragment key={`part-${i}`}>`.

---

## Acceptance criteria
- [x] `App.tsx` imports no view component and renders no view-specific content. Evidence: its imports are Layout, ViewRouter and overlays only; 188 lines.
- [x] App holds no server data in `useState`. Evidence: issues, config, inbox and user come from `api/queries.ts` hooks; `fetchData` and `lastJsonRef` are deleted.
- [x] SSE refreshes issues and inbox; Server Offline appears immediately and Retry works. Evidence: the inbox badge went from 11 to 12 live in the browser; `retry: false`; Retry calls `refetch()`. The user tophatted.
- [x] Each view owns its tab, filter and sort state through `useViewState`. Evidence: App has no `backlog*`, `myIssues*`, `inboxFilters` or `sortKey`.
- [x] `ViewRouter` maps `ViewId` → component through a record. Evidence: `VIEW_COMPONENTS`.
- [x] Every view renders through `ViewContainer`, including Inbox. Evidence: Inbox's list panel migrated.
- [x] `VIEWS` is the single source for the Sidebar, go-shortcuts, routes and titles; the palette keeps its curated list typed as `ViewId`. Evidence: `views.ts` with 12 tests.
- [x] Filter chips appear in the status bar for Backlog, MyIssues and Inbox. Evidence: `StatusBarSlot`; the "Label: frontend" chip showed in the browser and was absent on other views.
- [x] Persisted state survives reload under the existing keys, and Backlog's tab survives open issue → back. Evidence: the Active tab was kept after opening xpo-de3808 and pressing Esc; `exponential-my-issues-tab` was written by `]`.
- [x] Prev/next follows the originating view, with a fallback. Evidence: Backlog Active 4/6 → `k` → 3/6; Board 4/429 with hidden columns skipped; a fresh deep link gave 62/493.
- [x] xpo-0c0556: switching tabs loads that tab's filters. Evidence: Completed showed no chips, Active showed its label chip.
- [x] xpo-787e7b: a label click lands on `#/issues` with the filter persisted. Evidence: the only `hashchange` was `#/issues`; `exponential-backlog-filters-active` was written.
- [x] Deep links work. Evidence: `#/issues/<id>`, `#/dependencies/<id>`, not-found, and every bare route rendered.
- [x] Keyboard shortcuts work. Evidence: `g`+key, `]`, ⌘K, `c`, `?` (lists all "Go to …"), `k` and Esc in IssueDetail. The user worked the backlog.
- [x] The new pure modules were tested first. Evidence: `views.test.ts`, `view-state-utils.test.ts`, `nav-order-utils.test.ts` and `query-utils.test.ts` (32 tests), all red before green.
- [x] `make test` passes and the user tophatted. Evidence: exit 0, 480 frontend tests plus Go; the user reported "nothing seems broken, I worked the backlog a bit and it's all good".
