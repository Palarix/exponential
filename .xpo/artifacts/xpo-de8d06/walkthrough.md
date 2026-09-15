# Walkthrough: Extract composable Menu/MenuItem/SubMenu system

## What was built

A composable menu component system (`Menu`, `MenuItem`, `SubMenu`, `MenuDivider`, `MenuLabel`) that replaces bespoke menu implementations across the codebase with a single, keyboard-navigable, accessible primitive.

## New components

### Menu.tsx
The core container. Manages focus index, keyboard navigation (↑↓ Home/End Enter Space Escape, letter shortcuts), and submenu open/close state. Children register via `cloneElement` injection of internal props (`_menuIndex`, `_focused`, `_onClick`, etc.).

**Safe triangle**: When a submenu is open and the user moves diagonally toward it, hovering over other parent items doesn't immediately switch the submenu. Uses `pointInTriangle` geometry with a 300ms fallback timer. Only applies to non-SubMenu items — hovering another SubMenu trigger switches immediately.

**Auto-focus**: First navigable item highlighted on mount via `requestAnimationFrame`. A `focusViaKeyboard` ref gates whether the search input gets blurred — autoFocus highlights without stealing focus from search inputs, keyboard navigation blurs the input.

### SubMenu.tsx
Renders a `SubMenuTrigger` button and a portal flyout. Key behaviors:
- **Close delay**: 300ms timer on mouse leave (trigger or flyout), cancelled on re-enter. Timer is cancelled when `_subMenuOpen` becomes false to prevent stale timers from closing newly opened submenus.
- **Flyout alignment**: First navigable item in the flyout aligns horizontally with the trigger, clamped to viewport.
- **Two modes**: `children` (wrapped in `<Menu>` with keyboard nav) or `renderPanel` (raw content for custom pickers).

### menu-utils.ts
- `nextIndex`: Wrapping navigation with skip set. Returns -1 when no navigable item exists.
- `matchShortcut`: Case-insensitive single-character shortcut matching.
- `pointInTriangle`: Barycentric coordinate point-in-triangle test for safe triangle.

### PopoverPanel
New component for visual chrome (bg, border, shadow, padding). Popover itself is now positioning-only.

## Architecture decisions

### Popover is positioning-only
Popover previously carried visual chrome (surface-3 bg, border, shadow). This caused double borders when Menu (which has its own chrome) was nested inside. Popover was stripped to z-index + fixed positioning + click-outside + escape. PopoverPanel provides chrome for raw content. Menu provides its own chrome.

### MenuItem click routes through activateItem
Mouse clicks call `_onClick` (the Menu's `activateItem`) instead of the raw `onClick`. This ensures auto-close logic (close menu on non-checkbox click) works consistently for both mouse and keyboard.

### FilterMenu positioning uses queueMicrotask
Same fix as Popover (xpo-b5a70c): the anchor ref isn't attached when `useLayoutEffect` fires because React runs child layout effects before parent ref attachments. `queueMicrotask` defers positioning to after all refs are attached but before browser paint.

## Refactored consumers

### ContextMenu.tsx
Decomposed from ~470 lines to ~230. Business logic unchanged. Each property (Status, Priority, Assignee, Labels, Estimate, Cycle) is a `SubMenu` with `renderPanel`. Delete and Remove-from-parent are `MenuItem`s.

### FilterMenu.tsx
Rebuilt with Menu/MenuLabel/SubMenu. Each dimension (Status, Assignee, Priority, Labels, Epic) is a SubMenu with MenuItem children. Search inputs in MenuLabel. Positioning fixed.

### Backlog.tsx / Board.tsx view menus
Migrated from inline JSX + portal + manual click-outside to `Popover` + `Menu` composition. Layout/sort options use `MenuItem` with `active`/`checked` props.

### PropertySidebar, BacklogIssueRow, Board pickers, NewIssueModal, InlineDropdown
All raw-content Popovers wrapped in `PopoverPanel`.

## Visual design

Three-tier item hierarchy:
- **Default**: `text-secondary` (muted)
- **Hover/focused**: `hover-surface-3` bg + `text-primary`
- **Active**: `hover-surface-4` bg + `text-primary`
- **Checked**: blue checkmark, normal text

Compact sizing: `py-1.5` items, `py-1` labels, `my-0.5` dividers.

## Tests
- 6 new `right-start` placement tests in Popover.test.ts
- 15 new tests in Menu.test.ts (nextIndex, matchShortcut, pointInTriangle)
- All 357 frontend tests pass

## Acceptance criteria

- [x] Menu/MenuItem/SubMenu/MenuDivider components — implemented with full keyboard nav
- [x] ↑↓ arrow navigation, Enter to select, Escape to close — Menu keyboard handler
- [x] SubMenu flyout with right-preference + flip-left + vertical clamp — popover-utils right-start
- [x] Letter shortcut support — matchShortcut in Menu keyboard handler
- [x] ContextMenu refactored to compose from these primitives — SubMenu + renderPanel
- [x] FilterMenu refactored to compose from these primitives — SubMenu + MenuItem children
- [x] No behavioral or visual regression — tested in production build, all tests pass
