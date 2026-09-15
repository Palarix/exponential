# Spec: Composable Menu system (Menu, MenuItem, SubMenu, MenuDivider, MenuLabel)

## What

Build a composable menu component system that encapsulates keyboard navigation, ARIA semantics, and sub-menu flyout positioning. Refactor ContextMenu and FilterMenu to compose from these primitives.

## Why

Menu-like UI is scattered across 5+ components, each rolling its own keyboard handling, focus management, and sub-menu positioning. This duplication makes every new menu a maintenance burden and leaves some menus (e.g. InlineDropdown) without keyboard support entirely.

## Prior art

- **Popover primitive** (xpo-5d1c38) — anchor-ref positioning with `computePopoverPosition()` in `popover-utils.ts`. Currently supports `bottom-start | bottom-center | bottom-end`. Sub-menus need a `right-start` placement (with left-flip).
- **Keyboard registry** (xpo-caf557) — centralized `keydown` with priority levels. Menus register at `overlay` priority.

## Component API

### Menu

Container that renders a `role="menu"` list, manages roving `focusIndex`, and handles keyboard navigation.

```tsx
interface MenuProps {
  children: ReactNode;
  onClose?: () => void;
  className?: string;
  autoFocus?: boolean;       // auto-focus first item on mount (default: true)
  "aria-label"?: string;
}
```

Rendered markup: `<div role="menu" aria-label={...} tabIndex={-1}>` wrapping the children.

**Keyboard behavior:**
- `ArrowDown` — move focus to next item (wraps to first)
- `ArrowUp` — move focus to previous item (wraps to last)
- `Home` — focus first item
- `End` — focus last item
- `Enter` — activate focused item (click handler or open sub-menu)
- `Escape` — call `onClose`
- Single-character shortcuts — if a MenuItem has a matching `shortcut` prop, activate it

**Focus management:** roving tabindex pattern. The `Menu` tracks `focusIndex` as state. Each `MenuItem` receives `tabIndex={0}` when focused, `tabIndex={-1}` otherwise. `MenuDivider`, `MenuLabel`, and disabled items are skipped. On mount (if `autoFocus`), focus moves to the first enabled item.

### MenuItem

A single menu item. Behavior adapts based on props:

```tsx
interface MenuItemProps {
  label: string;
  icon?: ReactNode;
  shortcut?: string;        // single-char keyboard shortcut hint
  disabled?: boolean;
  checked?: boolean;         // when defined: role="menuitemcheckbox" + aria-checked + checkmark icon
  active?: boolean;          // subtle background highlight for "current" state (visual only)
  destructive?: boolean;     // red text/icon styling
  onClick?: () => void;
  suffix?: ReactNode;        // custom trailing content (after shortcut hint)
}
```

**Behavioral modes based on props:**

| Prop | ARIA role | Visual | Close on click? |
|------|-----------|--------|-----------------|
| (default) | `menuitem` | Standard | Yes (calls `onClose`) |
| `checked` defined | `menuitemcheckbox` | Checkmark when true | No (stays open for toggles) |
| `active` | `menuitem` | Subtle highlight bg | Yes |
| `destructive` | `menuitem` | Red text + icon color | Yes |

When `checked` is defined (even if `false`), the item renders as a checkbox-style toggle:
- `role="menuitemcheckbox"` with `aria-checked={checked}`
- Checkmark icon visible when `checked === true`
- Click does NOT auto-close the menu (toggle pattern)

When `checked` is `undefined`, the item is a regular action:
- `role="menuitem"`
- Click calls `onClick` then `onClose` (auto-close)

`active` is purely visual — a subtle background highlight indicating the "current" option (e.g. current sort order). It doesn't change ARIA behavior.

`destructive` applies red/danger styling to text and icon. Orthogonal to other props.

ARIA: `aria-disabled="true"` when disabled (not HTML `disabled`, so screen readers still announce it).

### SubMenu

A menu item that opens a nested `Menu` flyout.

```tsx
interface SubMenuProps {
  label: string;
  icon?: ReactNode;
  shortcut?: string;
  disabled?: boolean;
  children: ReactNode;       // the sub-menu content
}
```

Rendered markup: trigger is a `<button role="menuitem" aria-haspopup="menu" aria-expanded={open}>` with a chevron-right indicator. The flyout is a nested `Menu` rendered via `createPortal`.

**Positioning:** Uses `computePopoverPosition()` extended with `right-start` placement. Prefers opening to the right of the parent menu. If that would overflow the viewport, flips to the left. Vertical position aligns to the trigger row, clamped to viewport.

**Keyboard behavior (within parent menu):**
- `ArrowRight` or `Enter` on a SubMenu trigger — opens the sub-menu, focuses first item
- `ArrowLeft` inside the sub-menu — closes sub-menu, returns focus to trigger in parent
- `Escape` inside the sub-menu — closes sub-menu, returns focus to trigger in parent

**Mouse behavior:**
- `mouseenter` on the trigger opens the sub-menu (with ~50ms delay to avoid flicker)
- Moving mouse into the sub-menu keeps it open
- Moving mouse to a different parent item closes the sub-menu

### MenuDivider

Visual separator, skipped by keyboard navigation.

Rendered markup: `<div role="separator" />` with a border style.

### MenuLabel

Non-interactive section heading (e.g. "Layout", "Columns", "Add Filter..."). Skipped by keyboard navigation.

```tsx
interface MenuLabelProps {
  children: ReactNode;
}
```

Rendered markup: `<div role="presentation" className="...uppercase tracking-wider...">`. Styled to match the existing heading pattern (small caps, muted color).

## Implementation plan

### Step 1: Extend popover-utils with right-start placement

Add `right-start` to the `Placement` type. Logic:
- Initial: `top = anchor.top`, `left = anchor.right + offset`
- Left-flip: if `left + width > viewport.width - PAD`, flip to `anchor.left - width - offset`
- Left-clamp: if flipped `left < PAD`, clamp to PAD
- Vertical clamp: standard top/bottom clamping

Write tests first (TDD).

### Step 2: Menu context + pure keyboard logic

Create `menu-utils.ts` with pure functions:
- `nextIndex(current, count, direction, skip)` — returns the next focusable index with wrapping, skipping non-navigable indices
- `matchShortcut(key, shortcuts)` — finds an item index by its shortcut character

Create `MenuContext` to pass `focusIndex`, `setFocusIndex`, `onClose`, and registration down from Menu to children.

Write tests for the pure functions first (TDD).

### Step 3: Menu, MenuItem, MenuDivider, MenuLabel components

Build `Menu.tsx` with:
- `useKeyboardHandler` at `overlay` priority for arrow/enter/escape/home/end/character handling
- Roving tabindex via MenuContext
- Auto-focus first item on mount

Build `MenuItem` with prop-driven behavior:
- `checked` defined → `role="menuitemcheckbox"`, `aria-checked`, no auto-close
- `checked` undefined → `role="menuitem"`, auto-close on click
- `active` → subtle background highlight
- `destructive` → red styling

Build `MenuDivider` and `MenuLabel` as presentational components.

### Step 4: SubMenu component

Build `SubMenu.tsx`:
- Renders a MenuItem trigger with `aria-haspopup="menu"` and chevron
- On open, renders a child `Menu` via `createPortal` with `right-start` positioning
- Arrow-right/Enter opens, arrow-left/Escape closes
- Mouse enter/leave with delay

### Step 5: Refactor ContextMenu

Decompose `ContextMenu.tsx` to use `Menu`/`MenuItem`/`SubMenu`/`MenuDivider`:
- The main menu becomes a `Menu` with `MenuItem` entries for each property + a `SubMenu` for each picker
- "Remove from parent" and "Delete" remain as `MenuItem`s (Delete gets `destructive`)
- Delete confirmation dialogs stay as they are (not menus)
- Business logic (API calls, patchIssue) stays in ContextMenu
- Remove all manual keyboard handling, positioning, and focus management

### Step 6: Refactor FilterMenu

Rebuild `FilterMenu` using `Menu`/`SubMenu`:
- Main dimension list becomes `Menu` with `SubMenu` per dimension
- The sub-menu flyout container comes from `SubMenu`; filter-specific content (search input + checkbox list) renders as `SubMenu`'s children
- FilterMenu's existing `SubMenu` component (the checkbox list with search) becomes the content inside the generic `SubMenu` flyout

### Step 7: Verify no regressions

- All existing keyboard shortcuts work identically
- Sub-menu positioning matches current behavior
- Visual styling unchanged
- `make test` passes

## Decisions

1. **Single MenuItem with behavioral props** — instead of separate MenuCheckboxItem/MenuRadioItem components, a single `MenuItem` adapts its ARIA role and close-on-click behavior based on `checked`. When `checked` is defined → checkbox toggle that stays open. When undefined → regular action that auto-closes. This keeps the component set small while covering all existing menu patterns (action items, checkbox toggles, active indicators, danger items).

2. **Wrapping vs. clamping arrow navigation** — Arrow keys wrap around (ArrowDown on last item goes to first). Matches common menu patterns (macOS, VS Code).

3. **Focus management approach** — Roving tabindex (one item has `tabIndex={0}`, rest have `tabIndex={-1}`). WAI-ARIA recommended pattern for menus — lets the browser handle focus natively and works with screen readers.

4. **Sub-menu open/close model** — Mouse-enter opens (with delay), arrow-right opens, arrow-left/Escape closes. No click-to-toggle on sub-menu triggers.

5. **Menu registers its own keyboard handler** — Each `Menu` instance registers via `useKeyboardHandler` at overlay priority. Nested menus get their own registration, which naturally takes precedence because the keyboard registry processes in LIFO order within the same priority level.

6. **ContextMenu keeps its portal** — The root menu is positioned at right-click coordinates. The Menu component doesn't handle portaling — ContextMenu wraps it in `createPortal`.

7. **FilterMenu sub-menu content remains custom** — Filter sub-menus contain search inputs and multi-select checkboxes, which aren't standard MenuItem patterns. SubMenu provides the flyout container; filter-specific content renders as children.

## Acceptance criteria

- [ ] `Menu`, `MenuItem`, `SubMenu`, `MenuDivider`, `MenuLabel` components in `ui/`
- [ ] `role="menu"`, `role="menuitem"`, `role="menuitemcheckbox"`, `role="separator"` ARIA roles
- [ ] `aria-haspopup="menu"` and `aria-expanded` on SubMenu triggers
- [ ] `aria-checked` on MenuItem when `checked` is defined
- [ ] `aria-disabled` on disabled items
- [ ] Roving tabindex (`tabIndex={0}` on focused item, `-1` on others)
- [ ] ArrowUp/ArrowDown navigation with wrapping
- [ ] Home/End to jump to first/last item
- [ ] Enter activates focused item
- [ ] Escape closes menu (or sub-menu first)
- [ ] ArrowRight opens sub-menu, ArrowLeft closes it
- [ ] Single-character shortcut activation
- [ ] `checked` prop: checkbox role, checkmark icon, no auto-close
- [ ] `active` prop: subtle background highlight
- [ ] `destructive` prop: red/danger styling
- [ ] SubMenu flyout with right-preference + left-flip + vertical clamp
- [ ] `popover-utils` extended with `right-start` placement
- [ ] Pure utility functions with unit tests (TDD)
- [ ] ContextMenu refactored to compose from Menu/MenuItem/SubMenu
- [ ] FilterMenu refactored to compose from Menu/SubMenu
- [ ] No behavioral or visual regression
- [ ] `make test` passes