# Walkthrough: Extract `FilterButton` component (IconButton + FilterMenu wiring)

## Summary

Extracted a self-contained `FilterButton` component that replaces ~25 lines of duplicated filter wiring in each of Backlog, MyIssues, and Inbox. Also fixed two pre-existing bugs in FilterMenu's keyboard handling and added scroll-to-focused for long option lists.

## New file

### `web/src/components/ui/FilterButton.tsx`

A 3-prop component that internally owns all filter button state:

```tsx
<FilterButton issues={filteredIssues} filters={filters} onFiltersChange={onFiltersChange} />
```

Internally manages:
- `showFilterMenu` state and `filterBtnRef` ref
- IconButton with the filter SVG icon and `hasActiveFilters` → active dot
- FilterMenu conditional rendering with anchor ref
- `f` keyboard shortcut via two `useKeyboardShortcuts` registrations

### The dual-registration pattern for `f`

The `f` shortcut needs to work both to open (when closed) and close (when open). But FilterMenu registers at `overlay` priority with `blocksLowerPriorities=true`, which suppresses all `control`-priority shortcuts. Solution:

- **When closed**: `control` priority, `enabled: !open` — opens the menu
- **When open**: `overlay` priority, `blocksLowerPriorities: false`, `enabled: open` — closes the menu without interfering with FilterMenu's own overlay shortcuts

## Migrations

Each view replaced ~20-30 lines with a single `<FilterButton>` element:

| View | Removed |
|------|---------|
| Backlog | `showFilterMenu` state, `showFilterMenuRef` + sync, `filterBtnRef`, `f` handler, `showFilterMenuRef` guard, `f` shortcut entry, `showFilterMenu` from keyboard guards, 21-line JSX block |
| MyIssues | `showFilterMenu` state, `showFilterMenuRef` + useEffect sync, `filterBtnRef`, `f`+Escape handler, `f` shortcut entry, `showFilterMenu` from Tabs guard, 21-line JSX block |
| Inbox | `showFilterMenu` state, `filterBtnRef`, `f` handler, `f` shortcut entry, 20-line JSX block |

### Legacy keyboard guards removed

Backlog and MyIssues had manual guards (`showFilterMenuRef.current` in keyboard handlers, `!showFilterMenu` in `keyboardNavigationEnabled` props) to suppress other shortcuts while the filter menu was open. These were redundant — FilterMenu's overlay-priority registration with `blocksLowerPriorities=true` already suppresses all control/view/global handlers via the keyboard system's dispatch logic.

## Bug fixes in FilterMenu

### ArrowLeft missing from keyboard shortcuts (xpo-35fc16)

FilterMenu's `useKeyboardHandler` registered `["Escape", "ArrowRight", "ArrowDown", "ArrowUp", "Enter", " "]` but not `"ArrowLeft"`. The handler code at line 335 checked for ArrowLeft to enter submenus, but the handler never fired for it since it wasn't in the shortcuts array. Added `"ArrowLeft"` to the array.

### Scroll-to-focused in long option lists

When navigating filter options with arrow keys in a submenu, the focused option didn't scroll into view when beyond the visible area of the `max-h-64 overflow-y-auto` container. Added a `useEffect` that calls `scrollIntoView({ block: "nearest" })` on the focused button whenever `effectiveIndex` changes.

## Acceptance Criteria

- [x] `FilterButton` component created at `web/src/components/ui/FilterButton.tsx`
- [x] Exported from UI barrel
- [x] Backlog, My Issues, and Inbox migrated to `FilterButton`
- [x] `f` keyboard shortcut works in all three views (open and close)
- [x] Active indicator shows when filters are non-default
- [x] Legacy `showFilterMenuRef` guards removed from views
- [x] Arrow key submenu navigation works (ArrowLeft fix)
- [x] Long option lists scroll to focused item
- [x] `make test` passes (304 tests)
- [x] No visual regression — confirmed by user
