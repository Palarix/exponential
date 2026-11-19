# Walkthrough: eslint-plugin-react-hooks 7.1

## What was built
The project now uses `eslint-plugin-react-hooks` `^7.1.1`. Version 7.1 reports 30 errors that 7.0 did not. Each fix is a real code change, with no `eslint-disable`. The app looks and behaves as before. Tests and a markup comparison prove it (see **How we proved nothing changed**).

The new rules come from the React Compiler's analysis. The compiler is not in the build, but the lint rules use its model of a component. Three rules fired:

- `refs`: render code reads or writes `ref.current`. Render code also passes a function that touches refs to a call that may run it during render.
- `set-state-in-effect`: an effect calls `setState` synchronously, which costs an extra render.
- `preserve-manual-memoization`: the compiler cannot prove a `useMemo` or `useCallback` dependency is stable.

## The fixes, file by file

### `keyboard/hooks.ts`
`useKeyboardShortcuts` keeps the latest registration in `registrationRef`. The registry reads that ref when a key arrives. The hook wrote the ref during render; a `useLayoutEffect` with no deps now writes it. Layout effects run in the commit, before the browser can deliver the next keydown. So a handler always sees the values of the last committed render.

This hook guarantee simplifies the Backlog and App fixes below. A keyboard handler does not need to be stable, because the registry always calls the newest one.

### `App.tsx`
The `g`-key shortcuts were built with `.map`, and each entry had a `run: () => navigate(target)` closure. `navigate` writes refs, and `.map` runs its callback during render, so the `refs` rule fired. Two other forms still fired in a probe: a module-level helper, and `nav.navigate`.

The fix splits the shortcuts in two:
- `GO_SHORTCUTS` is a module-level list of plain data (`id`, `key`, `label`, `group`).
- A second registration, scope `global.go`, uses `useKeyboardHandler` with one handler: `navigate(VIEW_BY_GO_KEY[e.key])`. On a key sequence, the registry passes the last key's event, so `e.key` is the key after `g`.

`c` (create issue) stays alone on `global.navigation`, because `g c` (Cycles) uses the same final key. Both registrations keep `enabled: !showPalette && !showNewIssue`. `global.go` registers after `global.navigation`, so the help overlay lists the shortcuts in the old order.

### `ui/Menu.tsx`, `ui/SubMenu.tsx`, `ui/menu-owner.ts`, `Agent/IssueChats.tsx`
`Menu` clones its children and used to inject handlers as props (`_onClick`, `_onMouseEnter`, `_onSubMenuClose`, `_onFlyoutEnter`). Those handlers read refs, and `Children.map` runs during render, so the `refs` rule fired.

Now `Menu` provides `MenuOwnerContext` (`menu-owner.ts`) with `ownerId` and the handlers. `cloneElement` passes only plain values: `_menuIndex`, `_focused`, `_subMenuOpen`, `_ownerId`. `MenuItem`, `SubMenu` and `ChatRow` call `useMenuOwner(_ownerId)` to get the handlers.

`useMenuOwner` returns the context only when its `ownerId` equals the row's `_ownerId`. This keeps one old rule: only a Menu's direct children act on that Menu. Before, a `MenuItem` nested deeper got no injected props and ran its own `onClick`. A plain context would reach such an item, for example one in a submenu's custom panel. The item would then call the outer menu's `onItemClick(0)`. The `ownerId` check prevents that. A flyout's nested `<Menu>` provides its own context, so its rows bind to the nested menu.

`menu-owner.ts` is a separate file because `react-refresh/only-export-components` does not allow a component file to export a hook.

### `CommandPalette/CommandPalette.tsx`
Two effects reset state: one on open (empty query, first row), and one on each query change (first row).
- `CommandPalette` now renders `<PaletteBody>` only while it is open. Each open mounts a new body, so the state starts empty. The focus `setTimeout` stays in a mount effect, which no longer calls `setState`.
- The input's `onChange` sets the query and resets `selectedIndex` to 0 in the same event.

### `Backlog/Backlog.tsx`
- **Keyboard refs:** ten "latest value" refs (`rowsRef`, `focusedIndexRef`, …) existed only so that `handleKeyboard` could be a stable `useCallback`. The registry always calls the newest handler (see `keyboard/hooks.ts`). So `handleKeyboard` is now a plain function that reads `rows`, `focusedIndex` and the toggles directly. Most of the line count in the diff is this function's re-indent.
- **`issuesRef`:** removed. The code wrote it and never read it.
- **Drag batch:** `dragBatchRef` became state (`dragBatch`). Render reads it for the row fade and the count badge on `DragOverlayCard`. The batch is the dragged row plus its children in the same status (`getDragGroup`). `handleDndStart` sets it next to `setActiveId`, so both changes land in one render. `resetDropState` clears it.
- **`isDescendant`:** it was a `useCallback` that called itself, and the compiler flags reading a variable before its declaration. It is now a module-level function that takes `issues`.
- **Per-tab view settings:** an effect loaded each tab's expanded groups, hierarchy mode and show-empty-groups from localStorage. It ran after the render, so a tab switch painted one frame with the previous tab's settings. Module-level `loadExpandedGroups`, `loadHierarchyMode` and `loadShowEmptyGroups` now read the settings. The three states start from them. A `loadedTab` state reloads them during render when `activeTab` changes ("adjust state when a prop changes"). This removes the effect and its `exhaustive-deps` disable.
- **Memoization:** `tabFilters` is now a `useMemo` over `filtersForScope`. As a new object on each render, it made the compiler treat `filtering` and `filterKey` as values that could still change. That one change fixed all ten `preserve-manual-memoization` errors on `transientCollapsed`, `toggleNode`, `expandAllNodes` and `collapseAllNodes`.

A note for the next reader: the compiler skips a component after its first hard error. So `preserve-manual-memoization` errors can hide until the other errors in the file are gone. During this work, an ordering mistake (`setShowEmptyGroups` used before its declaration) hid them briefly.

## How we proved nothing changed
1. **Characterization tests first.** These tests were written and run against the unchanged code, and all passed. They pass unchanged after the refactor:
   - `keyboard/hooks.dom.test.tsx`: a shortcut runs the closure of the latest render, after new props and after its own state update.
   - `App.dom.test.tsx`: mounts the real `App` with views and data stubbed. It checks every `g`-key, `c`, ⌘K/Ctrl+K, and that no shortcut fires while the palette or the new-issue form is open. A snapshot holds the help overlay's shortcut list, in order.
   - `ui/Menu.dom.test.tsx`: click, hover, arrows, Home/End, Enter, type-ahead, Escape, keepOpen and disabled rows, submenus, and a `ChatRow` inside a Menu. `ChatRow` is exported for this test.
   - `CommandPalette/CommandPalette.dom.test.tsx`: closed, open, arrows, hover, query reset, Enter, Escape, and reopen.
   - `Backlog/Backlog.dom.test.tsx`: a seeded query cache and stubbed mutations. It checks j/k/arrows, Enter, `{`/`}`, the s/l/e/p/a/r popovers, and per-tab settings from localStorage. A row fingerprint shows group headers, issue ids, focus and the flat-mode breadcrumb.
2. **Byte-identical markup.** Before the refactor, a scratch test saved the full `document.body.innerHTML` of each Backlog tab, a focused row and an open popover. It also saved a menu with an open submenu, and the palette open and typed into. After the refactor, the HTML was byte-identical. The scratch test was then deleted.
3. **Browser check.** The user checked a parent drag (its children move with it, and the count is correct) and per-tab collapse and nested/flat.

The pre-commit hook lints the whole tree, so per-area commits were not possible. Instead, the full suite ran after each area.

## Found on the way
- xpo-3e25ae: in the demo board, Backlog drag and drop drops rows in the wrong position. A `main` build shows the same bug, and the user has not seen it on a real board. It also notes that `SubIssuesTable` sorts with `localeCompare`, unlike the Backlog's byte order.

## Out of scope
- The `eslint-disable-line` in `Dashboard.tsx:5` is not a react-hooks rule and stays.

## Acceptance criteria
- [x] `eslint-plugin-react-hooks` on `^7.1.1`; `bun run lint` passes with no new eslint-disable comments, and the Backlog `exhaustive-deps` disable is removed. Evidence: `bun run lint` is clean, and the only `eslint-disable` left in `web/src` is `Dashboard.tsx:5`.
- [x] Characterization tests written against the old code pass before and after the refactor, unchanged. Evidence: 45 new tests were green on the old code, then green after each area.
- [x] Rendered HTML of the Backlog tabs, a menu with a submenu, and the palette is identical before and after. Evidence: the snapshot files matched the old-code backups byte for byte (`cmp`).
- [x] `make test` passes. Evidence: exit 0, with 122 frontend test files and 1777 tests.
- [x] Checked in a browser: backlog drag and drop (a plain row, and a parent with its children and the count badge) and tab switching. Evidence: user tophat on the dev board. The drop-position problem also occurs on `main` and is filed as xpo-3e25ae.
