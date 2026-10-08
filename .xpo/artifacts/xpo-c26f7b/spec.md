# Extract `ViewContainer` shell wrapping TopBar + scrollable content

## What
Add `web/src/components/ui/ViewContainer.tsx`. It renders the standard view shell:

```
<div outer: h-full flex flex-col [className]>
  {topBar}
  {header}                          ← optional bars between TopBar and content
  <div content: flex-1 min-h-0 scroll-stable | overflow-hidden [contentClassName]>   ← full width: scrollbar at the right edge
    <div inner: mx-auto w-full max-w-* [innerClassName]>   ← only when maxWidth is set
      {children}
    </div>
  </div>
</div>
```

Then migrate every TopBar-over-content view to it.

Two standards become structural instead of conventions each view has to remember:
1. **The scrollbar sits at the right edge of the view.** The scroll container is always full width, and width-constrained views center their content *inside* it, using the pattern IssueDetail established in xpo-a8be39.
2. **Stable scroll gutter.** Every view's scroll container uses one semantic `scroll-stable` utility, not Tailwind overflow classes that a global CSS rule overrides.

## Why
This is part of epic xpo-570a07 and a prerequisite for xpo-de3808 (Layout → ViewRouter → View). The router refactor wants each view to have one uniform root. Today twelve shells hand-roll the same column with drift: class order, `relative` on some, padding on the scroll div in others.

## Findings
1. **Empty states outside the scroll div.** Backlog, MyIssues, Dependencies, DepGraph and Timeline render their empty state as a direct flex child.
2. **Bars between the TopBar and the content.** IssueDetail has `{banner}` (used by Inbox). MergeView has its merge bar and tabs bar.
3. **Non-scrolling content.**
   - Board is `overflow-hidden p-3`, holding columns.
   - DepGraph is an `overflow-hidden` pan/zoom canvas.
   - MergeView is `overflow-hidden`, and each tab scrolls itself.
4. **Refs and handlers.**
   - Content refs: Backlog `listRef` (plus `onClickCapture`/`onMouseLeave`), IssueDetail `scrollRef`, DepGraph `containerRef` (plus mouse and wheel handlers).
   - Outer ref: Board `boardRef`, used as a popover anchor.
5. **Inbox is a horizontal split.** Its TopBar sits inside a `w-80` left panel, and the right panel embeds IssueDetail.
6. **`PendingEventsPanel.tsx` is dead code.** It renders a TopBar, but nothing imports it.
7. **Overlays are portals or fixed-position.** ContextMenu, Modal and the MergeView dialog can stay anywhere in `children`, so no overlay slot is needed.
8. **The global scroll override is wider than intended.**
   - xpo-0b2c6c added `scrollbar-gutter: stable` to `.overflow-y-auto`, `.overflow-auto` and `.overflow-y-scroll`.
   - `e578645` (xpo-af0b09, an unrelated MergeView change) replaced that with `.overflow-y-auto, .overflow-y-scroll { overflow-y: scroll; }`.
   - So `overflow-y-auto` means "always scroll" app-wide, including Menu, Modal, CommandPalette, KeyboardHelp, NewIssueModal, PropertySidebar, BoardColumn and the MergeView tab panes.
   - It also dropped `.overflow-auto` entirely, so the MergeView diff panes have jumped when their scrollbar appears ever since.
9. **Width-constrained views already mostly follow the right-edge pattern.** The scroll div is full width with a centered `max-w-*` inner:
   - Cycles, Dashboard, Labels and Timeline use `max-w-7xl`.
   - IssueDetail uses `max-w-[76rem]` with `flex min-h-full` for its sticky sidebar.
   - Each one spells this out by hand.

## Design

### `scroll-stable` utility (`index.css`)
```css
@utility scroll-stable {
  overflow-y: scroll;
  scrollbar-gutter: stable both-edges;
}
```
- **`overflow-y: scroll`, not `auto`.**
  - Safari renders our styled (`::-webkit-scrollbar`) scrollbars as classic (space-taking) in either macOS scrollbar setting, but applies the gutter, `both-edges` included, only while a scrollbar exists.
  - With `auto`, Safari content jumped inward on both sides when the list started overflowing (tophat finding).
  - Forcing the track makes the scrollbar always exist, so the inset is constant. The track is transparent, so nothing shows when the content doesn't overflow.
  - Chrome 155 reserves the same 22px (11px per side) either way, as probed in the dev browser.
  - This is very likely what `e578645` was working around with its global `overflow-y: scroll`.
- **`both-edges`**: with classic scrollbars, the browser reserves an equal gutter on the left. Full-bleed row backgrounds (Backlog group headers, hover rows, menu highlights) are therefore inset symmetrically instead of flush-left and cut off on the right. Centered `max-w-*` columns also stay truly centered. Overlay scrollbars reserve nothing on either side.
- **Horizontal scrolling still works.** Only `overflow-y` is set, and CSS computes the other axis's `visible` to `auto`. Two-axis containers (the MergeView diff panes) can therefore use it in place of `overflow-auto`.
- **Naming**: phase 1 shipped this as `view-scroll` with a one-sided gutter. Phase 2 renamed it once it was used outside views and switched it to `both-edges` after tophat feedback.
- This is a Tailwind v4 `@utility`, so variants work.
- The broad `.overflow-y-auto, .overflow-y-scroll { overflow-y: scroll }` override is removed in **phase 2** of this issue (see Delivery), after a checkpoint commit. Removing it changes every non-view scroll container (finding 8), so it gets its own tophat.
- `ViewContainer` doesn't emit `overflow-y-auto`, so view content is already off the override after phase 1.

### API
```tsx
interface ViewContainerProps {
  topBar: ReactNode;
  header?: ReactNode;               // between TopBar and content (banners, tab bars)
  scroll?: boolean;                 // default true → scroll-stable; false → overflow-hidden
  maxWidth?: "7xl" | "detail";      // centered inner column: max-w-7xl | max-w-[76rem]; omit = full width
  className?: string;               // outer extras (e.g. MergeView bg), merged via cn()
  contentClassName?: string;        // scroll-container extras (padding, cursor)
  innerClassName?: string;          // inner-column extras (py-4, flex min-h-full); only with maxWidth
  contentRef?: Ref<HTMLDivElement>;
  contentProps?: Omit<HTMLAttributes<HTMLDivElement>, "className" | "children">;
  ref?: Ref<HTMLDivElement>;        // outer element (React 19 ref-as-prop)
}
```

Notes on the API:
- The `scroll` prop is now a boolean, replacing the issue's `'auto' | 'scroll' | 'hidden'`. `scroll-stable` always forces the track (see above), so there's nothing left to choose between `auto` and `scroll`.
- Class computation lives in pure helpers, `viewContentClass(scroll)` and `viewInnerClass(maxWidth)`, which are unit-tested.
- **Empty states:** while empty, a view passes `scroll={false}` and renders its empty state as `children`. This is the same box as today: a `flex-1` region with no gutter, holding the `h-full` EmptyState. It needs no extra slot.

## Migration scope (12 shells)
| View | Width | Notes |
|---|---|---|
| Backlog | full | `contentRef={listRef}`, `contentProps` handlers, `contentClassName="relative"`, `scroll={false}` when empty. `DndContext` is hoisted to wrap the container; ContextMenu and Modal sit outside the content div, because React events bubble through portals. |
| Board | full | `ref={boardRef}`, `scroll={false}`, `contentClassName="p-3"`. `DndContext` stays in children (no content handlers). |
| Cycles (CyclesTimeline) | 7xl | `innerClassName="pb-4"` |
| Dashboard | 7xl | `innerClassName="space-y-3 py-3"` |
| Dependencies (TableView) | full | `scroll={false}` when empty. The `px-5 py-3` padding moves to the list wrapper, and the local EmptyState uses `h-full`. |
| DepGraph | full | `scroll={false}`. The ref and handlers stay on its own `h-full` canvas div, so the empty state gets no pan/zoom handlers. The local empty state uses `h-full`. |
| IssueDetail | detail | `header={banner}`, `contentRef={scrollRef}`, `innerClassName="flex min-h-full"` |
| MergeView | full | `header` = merge bar + tabs bar, `scroll={false}`, `className="bg-[var(--color-surface)]"` |
| Labels | 7xl | `innerClassName="py-4"` |
| MyIssues | full | `scroll={false}` when empty |
| PendingChanges | full | plain |
| Timeline | 7xl | all three HeaderBar returns share one `headerBar` element; `innerClassName="py-2"`; `scroll={false}` when empty |

**Out of scope:**
- Inbox (horizontal split).
- Early returns with no TopBar.
- CycleDetail.

**Dropped:** outer `relative` on Backlog and IssueDetail. Nothing absolutely positioned depends on it.

## Delivery: two commits on this branch
**Phase 1: ViewContainer** (checkpoint commit `9a5e277`)
- Add the scroll utility and `ViewContainer`, and migrate the 12 shells.
- Delete the unused `PendingEventsPanel.tsx` (nothing imports it).

**Phase 2: remove the global override** (second commit after a separate tophat)
- **Audit**: every remaining `overflow-y-auto` / `overflow-y-scroll` / `overflow-auto` outside `ViewContainer`.
- **Rule**: a container gets `scroll-stable` if its content changes while it's visible (panes, filter-as-you-type lists, collapsible content, drop targets). Static content gets plain `overflow-y-auto`.
- **Opted in**:
  - Cycles detail panes
  - Inbox list
  - MergeView walkthrough, commits, files tree and conversation
  - **MergeView diff panes** in the Commits and Files tabs (`overflow-auto` → `scroll-stable`). Collapsing and expanding diffs toggles their overflow.
  - PropertySidebar and its parent, assignee and relation pickers
  - BoardColumn
  - Modal body
  - NewIssueModal pickers
  - CommandPalette
  - Menu with `maxHeight` (the context-menu Assignee panel, CyclePicker, and FilterMenu dimension submenus). This replaces the undefined `scrollbar-gutter-both` and `scrollbar-thin` classes.
- **Plain `overflow-y-auto`**:
  - Backlog and PropertySidebar sub-issue confirm lists
  - Timeline commit popover and its file list
  - MergeView conflict files
  - Dashboard distribution
  - KeyboardHelp
- Delete the `.overflow-y-auto, .overflow-y-scroll { overflow-y: scroll }` rule.
- **Backlog row styling** (tophat follow-on): with the symmetric gutter inset, full-bleed row separator lines look out of place.
  - `border-b` keeps its width but becomes `border-transparent`, so row heights are unchanged. This applies to the group header (`BacklogGroupHeader`), issue rows (`BacklogIssueRow`), the inline new-issue row, and the "+ N more" button.
  - The group header, issue rows and the inline new-issue row get `rounded-[var(--radius-md)]`, so their backgrounds (header surface, hover, focus ring) read as rounded pills. The "+ N more" button is rounded too, for its hover background.

## Acceptance criteria
- [ ] `scroll-stable` utility added to `index.css`
- [ ] `ViewContainer` exists in `ui/` and is exported from `ui/index.ts`
- [ ] `viewContentClass` / `viewInnerClass` are unit-tested
- [ ] The 12 shells are migrated, and there are no hand-rolled TopBar column shells left outside Inbox
- [ ] Every migrated view's scrollbar sits at the right edge of the view, and width-constrained views center inside the full-width scroll container
- [ ] Empty states show no scrollbar gutter (same as today)
- [ ] `contentRef` works: Backlog keyboard focus scroll and context menu, IssueDetail scroll-to-top on navigation, DepGraph fit-to-screen
- [ ] Board popovers still anchor correctly through the outer `ref`
- [ ] IssueDetail sticky sidebar still works (`flex min-h-full` inner)
- [ ] No visual regression (tophat each view)
- [ ] `PendingEventsPanel.tsx` deleted
- [ ] `make test` passes (phase 1)
- [ ] Phase 2: the global override is deleted; non-view scroll containers that need a stable gutter opt in explicitly; static lists no longer reserve a gutter; full-bleed row backgrounds are inset symmetrically (no right-side gap)
- [ ] Phase 2: tophat in **Chrome and Safari** shows no layout shift in the Backlog list (collapsing and expanding groups), PropertySidebar, MergeView tabs (including collapsing and expanding diffs in Files and Commits), Inbox, Cycles detail, modals or the command palette; diff panes still scroll horizontally
- [ ] Phase 2: Backlog rows and group headers have no separator lines (borders are transparent and row heights unchanged) and rounded backgrounds
- [ ] `make test` passes (phase 2)

## Decisions (confirmed by user)
1. The global override is removed in this issue, as a separate phase-2 commit with its own tophat.
2. Full-width views (Backlog, MyIssues, Dependencies, PendingChanges) stay full width.
3. Inbox is out of scope for now.
4. Hoisting `DndContext` in Backlog is acceptable; verify there are no side effects from it being always mounted.
5. Story points: 3.
6. `PendingEventsPanel.tsx` is deleted as part of this issue.
7. `scroll-stable` uses `scrollbar-gutter: stable both-edges` everywhere: one utility, symmetric insets. This was a tophat decision; the gap appeared next to the Backlog group headers.
8. The MergeView diff panes move to `scroll-stable`. This was a tophat finding: content jumped when collapsing all diffs and then expanding one.
9. Backlog row separator lines are removed (transparent border colour), and status headers and rows are rounded. This is part of the phase-2 commit; it was a tophat decision.
10. `scroll-stable` forces `overflow-y: scroll`, because Safari applies `scrollbar-gutter` only while a scrollbar exists. This was a tophat finding in Safari.
