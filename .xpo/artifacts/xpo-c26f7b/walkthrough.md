# Walkthrough: Extract `ViewContainer` shell + stable scrolling

The work landed in two commits on the branch:
- **Phase 1 (`9a5e277`):** the `ViewContainer` component and the migration of 12 view shells.
- **Phase 2 (`f0f908c`):** removing a global CSS override that had quietly changed how Tailwind's `overflow-y-auto` behaves, and replacing it with an explicit `scroll-stable` utility. Tophat findings in Chrome and Safari shaped it into its final form.

Bug xpo-164038 was found and fixed along the way.

---

## Phase 1: `ViewContainer`

### The problem
Twelve views each hand-rolled the same column: an outer `h-full flex flex-col`, a `TopBar`, then a `flex-1 overflow-y-auto` content div. Each copy had drifted:
- class order
- `relative` on some
- padding on the scroll div in others
- empty states rendered *outside* the scroll div in five views

xpo-de3808 (Layout → ViewRouter → View) needs every view to have a uniform root, so this had to be consolidated first.

### The component (`web/src/components/ui/ViewContainer.tsx`)
```
outer  h-full flex flex-col [className]           ← ref
  topBar
  header?                                          ← banners / tab bars (IssueDetail banner, MergeView merge+tabs bars)
  content  flex-1 min-h-0 scroll-stable | overflow-hidden [contentClassName]   ← contentRef, contentProps
    inner?  mx-auto w-full max-w-7xl | max-w-[76rem] [innerClassName]   ← only with maxWidth
      children
```

Props:
- **`scroll` is a boolean.**
  - `true` (the default) emits the `scroll-stable` utility.
  - `false` emits `overflow-hidden`. Canvases (Board, DepGraph, MergeView) use it, and so do empty states.
- **`maxWidth` makes the right-edge scrollbar a structural rule.** The scroll container is always full width, and constrained views center an inner column *inside* it. IssueDetail first established this pattern in xpo-a8be39; now Cycles, Dashboard, Labels and Timeline (`7xl`) and IssueDetail (`detail`) get it from one prop instead of spelling it out.
- **`contentRef` / `contentProps` exist for Backlog.** Its scroll container is also its DOM-query root and carries click/mouse handlers.
- **The class strings come from pure helpers** in `view-container-utils.ts` (`viewContentClass`, `viewInnerClass`) and are unit-tested. This follows the `search-input-utils` pattern, which also keeps fast-refresh lint happy.

### Empty states
Five views rendered their empty state as a direct flex child so it wouldn't get a scrollbar gutter. Rather than add an `empty` slot, those views pass `scroll={!isEmpty}`. While empty, the content div is `overflow-hidden`, which gives the same box as before with no gutter.

Two local empty states, in Dependencies and DepGraph, relied on `flex-1` to fill the column. Inside the content div that no longer stretches, so they became `h-full`, matching `ui/EmptyState`.

Dependencies' `px-5 py-3` padding moved from the scroll div into the list wrapper, so it doesn't pad the empty state.

### Per-view notes worth knowing
- **Backlog** is the one non-trivial migration.
  - `DndContext` is hoisted to wrap the `ViewContainer`. It renders no layout DOM: only dnd-kit's `HiddenText` (`display:none`) and a fixed 1px `LiveRegion`. So keeping it always mounted is harmless; that was checked during review.
  - `ContextMenu` and `Modal` are rendered **outside** the content div, in a fragment. Both portal to `document.body`, but React synthetic events still bubble through the React tree. Inside the content div, a click in the context menu would hit the list's `onClickCapture` drag guard.
  - A side benefit: the sub-issue Modal no longer unmounts if the filtered list goes empty while it's open.
- **Board** doesn't hoist `DndContext`. Its content div has no handlers, so the provider, overlays and popovers just live in `children`. `ref={boardRef}` keeps the popover anchor on the outer element.
- **DepGraph** keeps its ref and pan/zoom handlers on its own `h-full` canvas div inside a `scroll={false}` container. That way the empty state never receives wheel or mouse handlers.
- **Timeline** has three returns: no events, no matches, and the list. They now share one `headerBar` element, and all three use `ViewContainer`.
- **IssueDetail** passes `header={banner}` (used by Inbox) and `innerClassName="flex min-h-full"`. The inner column has to establish the flex row whose sticky `PropertySidebar` relies on `min-h-full`.
- **Dropped** outer `relative` on Backlog and IssueDetail. Nothing absolutely positioned depended on it; IssueDetail's tab underline has its own positioned parent.
- **Deleted** `PendingEventsPanel.tsx`. It rendered a TopBar, but nothing imported it.

### Bug fixed: xpo-164038
Backlog attached a native `contextmenu` listener to `listRef.current` in a `useEffect(..., [])`, but the list div only rendered when the filtered list was non-empty:
- If Backlog mounted while the list was empty, no listener was ever attached.
- If the list emptied and came back (a search with no matches that is then cleared), React mounted a new div while the listener stayed on the detached old one.

`ViewContainer` makes the content div (`contentRef={listRef}`) permanent, so the mount-time listener is always on the live element.

---

## Phase 2: the scroll override and `scroll-stable`

### History that explains the mess
- **xpo-0b2c6c** added `scrollbar-gutter: stable` to `.overflow-y-auto`, `.overflow-auto` and `.overflow-y-scroll`.
- **A day later, `e578645`** (xpo-af0b09, an unrelated MergeView change) replaced that with `.overflow-y-auto, .overflow-y-scroll { overflow-y: scroll }`. It also dropped `.overflow-auto` from the selector.
- **The effect:** every `overflow-y-auto` in the app silently meant "always scroll", including menus, modals, the command palette and pickers.
- **And:** the two-axis MergeView diff panes have had no stable gutter since then.

The Safari finding (below) shows why that change was probably made. It was never written down, which is how a global rule ended up changing Tailwind semantics.

### The utility (`web/src/index.css`)
```css
@utility scroll-stable {
  overflow-y: scroll;
  scrollbar-gutter: stable both-edges;
}
```
Each part of this exists because of a tophat finding:

1. **`scrollbar-gutter: stable`:** content mustn't shift when it starts or stops overflowing.
2. **`both-edges`** (finding: a gap next to the Backlog group headers).
   - With a one-sided gutter, full-bleed row backgrounds ran flush-left but stopped at the scrollbar track on the right, leaving a visible gap. Linear avoids this because its rows are inset from both edges.
   - `both-edges` mirrors the gutter on the left, so backgrounds are inset symmetrically. Centered `max-w-*` columns also become truly centered, where before they sat half a gutter left of centre.
   - The user chose one utility everywhere over a separate both-edges variant: about 11px of extra inset in narrow panes was worth the consistency.
3. **`overflow-y: scroll` instead of `auto`** (finding: Safari jumps).
   - Safari renders our styled (`::-webkit-scrollbar`) scrollbars as classic, space-taking scrollbars in either macOS scrollbar setting, but applies the gutter only while a scrollbar exists.
   - With `auto`, content was flush while it fit, then jumped inward on both sides once it overflowed.
   - Forcing the track means the scrollbar always exists, so the inset is constant. The track is transparent, so nothing shows when content fits.
   - Chrome 155 measured identically either way: 22px reserved, with the header at 260/1844 across collapse and expand.

Only `overflow-y` is set. CSS computes the other axis's `visible` to `auto`, so two-axis containers keep horizontal scrolling. That's why the MergeView diff panes could switch from `overflow-auto` to `scroll-stable` (finding: diffs jumped when you collapsed all and then expanded one).

### Which containers opt in
**Rule:** `scroll-stable` if the container's content changes while it's visible (panes, filter-as-you-type lists, collapsible content, drop targets). Otherwise plain `overflow-y-auto`, which now really means auto.

| `scroll-stable` | plain `overflow-y-auto` |
|---|---|
| ViewContainer content; Cycles detail panes; Inbox list; MergeView walkthrough, commits, files tree, conversation and both diff panes; PropertySidebar and its parent, assignee and relation pickers; BoardColumn; Modal body; NewIssueModal pickers; CommandPalette; Menu with `maxHeight` | sub-issue confirm lists (Backlog, PropertySidebar); Timeline commit popover and its file list; MergeView conflict files; Dashboard distribution; KeyboardHelp |

`Menu` previously used `scrollbar-gutter-both scrollbar-thin`. Neither class exists (Tailwind v4 has no such utilities and no scrollbar plugin is installed), so its gutter had only ever come from the global override. It now uses `scroll-stable`. Thin scrollbars already come from the global `scrollbar-width: thin`.

### Backlog row styling
With the symmetric inset, full-bleed separator lines looked out of place.
- **Lines:** the group header, issue rows, the inline new-issue row and the "+ N more" button now use `border-b border-transparent`. The border keeps its width, so row heights are unchanged.
- **Rounding:** all four get `rounded-[var(--radius-md)]`, so header surfaces, hover and focus backgrounds, and the inset keyboard focus ring draw as rounded shapes.

---

## Acceptance criteria
- [x] `scroll-stable` utility added to `index.css`. Evidence: `@utility scroll-stable { overflow-y: scroll; scrollbar-gutter: stable both-edges }`.
- [x] `ViewContainer` exists in `ui/` and is exported from `ui/index.ts`.
- [x] `viewContentClass` / `viewInnerClass` are unit-tested. Evidence: 6 tests in `ViewContainer.test.ts`, written first; the rename to `scroll-stable` was also test-first.
- [x] The 12 shells are migrated, and there are no hand-rolled TopBar column shells left outside Inbox. Evidence: every TopBar consumer except `Inbox.tsx` renders `ViewContainer`.
- [x] Every migrated view's scrollbar sits at the right edge, and constrained views center inside the full-width container. Evidence: `maxWidth` on Cycles, Dashboard, Labels, Timeline and IssueDetail; tested by the user.
- [x] Empty states show no gutter. Evidence: `scroll={!isEmpty}` in Backlog, MyIssues, Dependencies and Timeline, and `scroll={false}` in DepGraph; tested by the user.
- [x] `contentRef` works. Evidence: Backlog `listRef` (focus scroll and context menu, including after a no-match search), IssueDetail `scrollRef`, and DepGraph `containerRef` on its canvas; tested by the user.
- [x] Board popovers anchor through the outer `ref`. Evidence: `ref={boardRef}`; tested by the user.
- [x] IssueDetail's sticky sidebar still works. Evidence: `innerClassName="flex min-h-full"`; tested by the user.
- [x] No visual regression. Evidence: the user tophatted phase 1 and approved it.
- [x] `PendingEventsPanel.tsx` deleted.
- [x] `make test` passes (phase 1). Evidence: lint, 435 frontend tests and Go are green; `tsc -b` is clean.
- [x] Phase 2: the global override is deleted, opt-in is explicit, static lists reserve no gutter, and backgrounds are inset symmetrically. Evidence: the rule is removed from `index.css` and the opt-in table is above.
- [x] Phase 2: no layout shift in Chrome or Safari. Evidence: Chrome probe (22px reserved and header at 260/1844 across states); the user confirmed in Safari, including MergeView diff collapse and expand.
- [x] Phase 2: Backlog rows have no separator lines (transparent borders) and rounded backgrounds. Tested by the user.
- [x] `make test` passes (phase 2). Evidence: 435 tests green.
