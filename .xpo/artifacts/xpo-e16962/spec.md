# Spec: Extract `FilterButton` component (IconButton + FilterMenu wiring)

## What

Extract a `FilterButton` component that bundles the filter IconButton, filter menu visibility state, anchor ref, `f` keyboard shortcut, and FilterMenu rendering into a single reusable component. Migrate Backlog, My Issues, and Inbox.

## Why

All three views duplicate ~25 lines of identical wiring: `useState(showFilterMenu)`, `useRef(filterBtnRef)`, the filter icon SVG, `hasActiveFilters` check, and `<FilterMenu>` rendering. The `f` keyboard shortcut is also duplicated in each view's keyboard handler. Additionally, the views maintain manual keyboard guards (`showFilterMenuRef`, `keyboardNavigationEnabled={!showFilterMenu}`) that are redundant — the keyboard system's `blocksLowerPriorities` mechanism already blocks lower-priority shortcuts when FilterMenu's overlay-priority handler is active.

## Current State

Each view wires:
1. `const [showFilterMenu, setShowFilterMenu] = useState(false)`
2. `const filterBtnRef = useRef<HTMLButtonElement>(null)`
3. `f` key handler in the view's keyboard handler to toggle the menu
4. `<div className="relative">` + `<IconButton>` with filter SVG + `{showFilterMenu && <FilterMenu ...>}`
5. Backlog and MyIssues maintain a `showFilterMenuRef` to guard keyboard handling — **redundant** because FilterMenu registers at overlay priority with `blocksLowerPriorities=true` (the default for overlay), which already prevents all control/view/global handlers from firing

## Why no `onOpenChange`

The keyboard system's dispatcher computes a `blockingPriority` from all enabled entries with `blocksLowerPriorities=true`. When FilterMenu is mounted at overlay priority (3), all lower priorities are filtered out before matching:
- `control` (SearchInput `/`, Tabs `[`/`]`, FilterButton `f`): BLOCKED
- `view` (arrow keys, letter shortcuts): BLOCKED
- `global`: BLOCKED

FilterMenu's own overlay handler handles Escape to close. No state needs to leak to the parent.

## API

```tsx
interface FilterButtonProps {
  issues: Issue[];
  filters: BacklogFilters;
  onFiltersChange: (filters: BacklogFilters) => void;
}
```

FilterButton is fully self-contained — it internally owns:
- `showFilterMenu` state
- `filterBtnRef` ref
- The filter icon SVG
- `hasActiveFilters` check → IconButton `active` prop
- `<FilterMenu>` conditional rendering
- `f` keyboard shortcut registration (at `control` priority)

## Flow

### 1. Create `FilterButton.tsx` at `web/src/components/ui/FilterButton.tsx`

- Uses `IconButton` for the button with the filter SVG icon
- `active={hasActiveFilters(filters)}`
- `tooltip="Filter"`
- Manages `showFilterMenu` state internally
- Creates its own `filterBtnRef` for FilterMenu anchoring
- Registers `f` shortcut via `useKeyboardShortcuts` at `control` priority
- Renders `<FilterMenu issues={issues} filters={filters} onChange={onFiltersChange} anchorRef={filterBtnRef} onClose={...} />` when open

### 2. Export from barrel (`web/src/components/ui/index.ts`)

### 3. Migrate Backlog, MyIssues, Inbox

For each view:
- Remove `showFilterMenu` state, `filterBtnRef` ref, `showFilterMenuRef` (if present)
- Remove `f` key handling from the view's keyboard handler
- Remove `f` from the view's shortcuts array
- Remove `showFilterMenu` from keyboard navigation guards (redundant — blocksLowerPriorities handles it)
- Replace the `<div className="relative">` + IconButton + FilterMenu block with `<FilterButton issues={...} filters={filters} onFiltersChange={onFiltersChange} />`

### 4. `make test`

## Decisions

- **`f` shortcut at `control` priority** — follows the pattern established by SearchInput (`/`) and Tabs (`[`/`]`). The help overlay folds control-priority shortcuts into the active view's section.
- **Fully self-contained, no `onOpenChange`** — FilterMenu's overlay-priority registration with `blocksLowerPriorities` already suppresses all lower-priority handlers when the menu is open. The legacy `showFilterMenuRef` guards are redundant and will be removed.
- **No separate test file** — FilterButton is a compositional component (wiring, not computation). `hasActiveFilters` is already tested in `filters.ts`.
- **FilterMenu import from `../Backlog/FilterMenu`** — FilterMenu is specific to the BacklogFilters type. Moving it to `ui/` is a future concern.

## Acceptance Criteria

- [ ] `FilterButton` component created at `web/src/components/ui/FilterButton.tsx`
- [ ] Exported from UI barrel
- [ ] Backlog, My Issues, and Inbox migrated to `FilterButton`
- [ ] `f` keyboard shortcut works in all three views
- [ ] Active indicator shows when filters are non-default
- [ ] Legacy `showFilterMenuRef` guards removed from views
- [ ] `make test` passes
- [ ] No visual regression
