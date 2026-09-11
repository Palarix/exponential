# Walkthrough: Extract `SearchInput` component

## What was built

A shared controlled `SearchInput` component that encapsulates the TopBar search pattern: magnifying glass icon, `/`/`Esc` kbd hint, `Escape` clear-and-blur, and `/` focus shortcut registered at `control` priority through the keyboard registry. Backlog and Dependencies were migrated to use it.

A pre-existing TopBar centering bug (xpo-c078b7) was fixed — the center slot now uses equal-width flex columns instead of `flex-1` with `justify-center`, so the search input stays centered regardless of left/right content width changes.

## How the pieces fit together

### SearchInput (`web/src/components/ui/SearchInput.tsx`)

Accepts `value`, `onChange`, optional `placeholder`, and `keyboardNavigationEnabled`. Manages its own `inputRef` internally — no ref forwarding needed. Registers `/` at `control` priority via `useKeyboardShortcuts`, following the same pattern as `Tabs` (xpo-d47781). The help overlay folds it into the active view's section automatically.

The kbd hint logic lives in `search-input-utils.ts` — shows `/` when empty (or whitespace-only), `Esc` when populated.

### Consumer migration

**Backlog** — replaced 22 lines of inline search markup with `<SearchInput>`. Removed the `/` case from `handleKeyboard`, removed `/` from the shortcut metadata array, and removed `searchRef`.

**Dependencies** — replaced 22 lines of inline markup and the standalone `useKeyboardShortcuts` registration for `/` with `<SearchInput>`. Removed `searchRef` and the now-unused `useRef` import.

### TopBar centering fix (xpo-c078b7)

The old layout used `flex-1 justify-center` for the center slot — centering within remaining space, not the viewport. When right-side content changed width (issue count), the center shifted.

Fixed by giving left and right equal `flex-1` columns with `min-w-0`. The center sits between them at a fixed `max-w-md`. Equal flex shares keep it stable regardless of content width differences.

## Acceptance criteria

- [x] `SearchInput` renders the same markup as the current inline inputs
- [x] The kbd hint shows `/` when empty, `Esc` when populated
- [x] `Escape` clears the value and blurs the input
- [x] `/` focuses the input, registered at `control` priority — no document listener
- [x] `SearchInput` does not fire when disabled (`keyboardNavigationEnabled`)
- [x] Backlog and Dependencies render `SearchInput` in `TopBar.center`
- [x] Backlog's `/` handler removed from view-level keyboard registration
- [x] Dependencies' standalone `/` shortcut registration removed
- [x] The `/` shortcut appears under the active view's section in keyboard help
- [x] Tests, build, and lint pass

## Files changed

| File | Change |
|------|--------|
| `web/src/components/ui/SearchInput.tsx` | New — shared search input with keyboard registration |
| `web/src/components/ui/search-input-utils.ts` | New — kbd hint logic |
| `web/src/components/ui/SearchInput.test.ts` | New — 3 tests for hint logic |
| `web/src/components/ui/index.ts` | Added `SearchInput` export |
| `web/src/components/ui/TopBar.tsx` | Fixed centering with equal-width flex columns (xpo-c078b7) |
| `web/src/components/Backlog/Backlog.tsx` | Replaced inline search + removed `/` handler |
| `web/src/components/Dependencies/Dependencies.tsx` | Replaced inline search + removed standalone registration |