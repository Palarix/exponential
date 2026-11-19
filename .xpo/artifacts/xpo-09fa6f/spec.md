# Upgrade eslint-plugin-react-hooks to 7.1 and fix its new lint errors

## What
Bump `eslint-plugin-react-hooks` from `~7.0.1` to `^7.1.1` and fix every error it reports with real code changes. No `eslint-disable` comments. **The app must look and behave exactly as it does today.**

Under 7.1.1 `bun run lint` reports 30 errors. That's the 20 the description lists plus 10 `preserve-manual-memoization` errors in `Backlog.tsx` (`transientCollapsed`, `toggleNode`, `expandAllNodes`, `collapseAllNodes`). The React Compiler is not in the build, so those 10 are lint-only findings, but the rule is in the recommended set.

## How, per site

| Site | Fix |
|---|---|
| `keyboard/hooks.ts:14` | Update `registrationRef.current` in a `useLayoutEffect` with no deps. The registry already reads the latest registration through the ref, and layout effects run before the browser can deliver the next keydown. |
| `Backlog.tsx` "latest refs" (`rowsRef` … `collapseAllNodesRef`) | Delete them. `handleKeyboard` becomes a plain function that closes over current `rows`, `focusedIndex`, `toggleGroup`, and so on. That works because `useKeyboardHandler` → `useKeyboardShortcuts` already calls the newest handler through its own ref, so the handler doesn't need to be stable. |
| `Backlog.tsx` `issuesRef` | Delete it: it's written and never read. |
| `Backlog.tsx` `dragBatchRef` (render reads at row props and `DragOverlayCard`; the batch is the dragged row plus its children in the same status, from `getDragGroup`) | Make it state: `const [dragBatch, setDragBatch] = useState<string[]>([])`. Set it in `handleDndStart` next to `setActiveId`, and clear it in `resetDropState`. |
| `Backlog.tsx` `isDescendant` recursion | Make it a module-level `isDescendant(issues, parentId, childId)`, and pass `issues` in at the call sites. |
| `Backlog.tsx` tab effect (expanded groups, hierarchy mode, show-empty-groups loaded from localStorage) | Module-level `loadExpandedGroups`, `loadHierarchyMode` and `loadShowEmptyGroups` read a tab's settings. The three states start from them, and a `loadedTab` state reloads them during render when `activeTab` changes (React's "adjust state when a prop changes" pattern). Tab switches render the right groups on the first pass, with no extra frame. This also removes the existing `exhaustive-deps` disable. |
| `Backlog.tsx` `preserve-manual-memoization` | Memoize `tabFilters` (`useMemo` over `filtersForScope`). A fresh object each render made the compiler treat `filterKey` and `filtering`, which derive from it, as values that could still change. |
| `CommandPalette.tsx:41` (reset on open) | Split it in two. `CommandPalette` returns `null` when closed and otherwise renders `<PaletteBody>`, which mounts fresh on each open (empty query, index 0). The focus `setTimeout` stays in an effect, but it no longer calls setState. |
| `CommandPalette.tsx:128` (reset index on query change) | Reset `selectedIndex` to 0 in the input's `onChange` and remove the effect. |
| `ui/Menu.tsx:403` (`Children.map` clones in ref-reading handlers) | The Menu provides a `MenuOwnerContext` (`ui/menu-owner.ts`) with `ownerId` and the handlers `onItemMouseEnter`, `onItemClick`, `closeSubMenu` and `onFlyoutEnter`. `cloneElement` passes only plain values: `_menuIndex`, `_focused`, `_subMenuOpen`, `_ownerId`. `MenuItem`, `SubMenu` and `ChatRow` (`Agent/IssueChats.tsx`) get the handlers with `useMenuOwner(_ownerId)`, which returns the context only when its `ownerId` matches. So only a Menu's direct children act on it, as with the cloned props before: a `MenuItem` nested deeper (say, in a submenu's custom panel) still runs its own `onClick`. A flyout's nested `<Menu>` provides its own context for its rows. |
| `App.tsx:147` (`.map` creates closures over `navigate`, which writes refs) | Split the go-to shortcuts into their own `useKeyboardHandler` registration (scope `global.go`). The shortcut list (`GO_SHORTCUTS`, module level) holds only data, and one handler runs `navigate(VIEW_BY_GO_KEY[e.key])`. `c` (create issue) stays on `global.navigation` because `g c` (Cycles) shares the key. Both keep `enabled: !showPalette && !showNewIssue`, and `global.go` registers after `global.navigation`, so the help overlay lists the shortcuts in the same order. Probes showed that moving the `.map` into a helper, or reading `nav.navigate` instead, is still flagged. |

## Out of scope
- The `eslint-disable-line` in `Dashboard.tsx:5`. It isn't a react-hooks rule.

## How we keep behavior unchanged
Characterization first, refactor second. Use the existing jsdom harness (`src/test-dom`, `*.dom.test.tsx`).

1. **Write tests against today's code, before any refactor.** They must pass on the old code and stay green, unchanged, after each fix:
   - **Keyboard hook** (`keyboard/hooks.dom.test.tsx`): a re-rendered component's shortcut runs its newest `run`.
   - **App shortcuts** (`App.dom.test.tsx`, the real `App` with views and data stubbed): each `g`+key goes to its view, `c` opens a new issue, neither fires while the palette or new-issue dialog is open, and a snapshot of the help overlay's shortcut list, in order.
   - **Menu** (`ui/Menu.dom.test.tsx`): hover, click, arrow keys, Enter, type-ahead, keepOpen and disabled rows, opening a submenu, items inside a submenu acting on their own menu, and a custom row (`ChatRow`, exported for the test). The existing `SubMenu.dom.test.tsx` and `MenuSeparators.test.tsx` stay green.
   - **Command palette** (`CommandPalette.dom.test.tsx`): opening gives an empty query, reopening resets it, typing selects the first item, and arrows/Enter/Esc/hover work.
   - **Backlog** (`Backlog.dom.test.tsx`): j/k/arrows/Enter/`{`/`}`/s/l/e/p/a/r on real rows, and tab switching restores each tab's expanded groups, hierarchy mode and show-empty-groups from localStorage.
2. **Temporary markup check:** before the refactor, save the rendered HTML of the Backlog (each tab, a focused row, an open popover), an open menu with an open submenu, and the open palette. After the refactor, check it is byte-identical. This is a scratch test, deleted before commit.
3. **One area at a time:** keyboard hook, App, Menu, CommandPalette, Backlog, with lint for the area and the full test suite green after each. (Separate commits per area weren't possible: the pre-commit hook lints the whole tree against 7.1.1, so no commit passes until every area is fixed.)

Left to check by hand: backlog drag and drop, including dragging a parent with its children and the count badge on the drag card (dnd-kit pointer sensors don't run in jsdom), and a quick look at backlog tab switching.

## Acceptance criteria
- [x] `eslint-plugin-react-hooks` on `^7.1.1`; `bun run lint` passes with no new eslint-disable comments, and the Backlog `exhaustive-deps` disable is removed.
- [x] Characterization tests written against the old code pass before and after the refactor, unchanged.
- [x] Rendered HTML of the Backlog tabs, a menu with a submenu, and the palette is identical before and after.
- [x] `make test` passes.
- [x] Checked in a browser: backlog drag and drop (a plain row, and a parent with its children and the count badge) and tab switching.
