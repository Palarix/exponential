# Spec: Rewrite pickers as Menu compositions

## What

Rewrite `StatusPicker`, `PriorityPicker`, `EstimatePicker`, and `LabelPicker` as `Menu` + `MenuItem` compositions. Add two extensions to the Menu system:

1. **`MenuFilter`** — a non-navigable filter input child for Menu (LabelPicker now, FilterMenu submenus later)
2. **`bare` prop on Menu** — strips visual chrome so pickers inside PopoverPanel/SubMenu don't double up

No separate `Picker` component. No new props on MenuItem — its existing `checked` + CheckIcon handles all selection indication, including LabelPicker's multi-select.

## Why

The four picker components reimplement what Menu + MenuItem already provide: keyboard navigation, focus highlighting, checked state with CheckIcon, shortcut keys, Escape to close. Adding two small, general-purpose primitives eliminates the parallel implementation.

Part of the "Refactor WebUI to component tree" epic (xpo-570a07).

## Acceptance Criteria

- [ ] `MenuFilter` component added to Menu.tsx, non-navigable, with auto-focus
- [ ] `bare` prop on Menu strips chrome (bg, border, shadow, padding, rounded)
- [ ] `StatusPicker` rewritten as bare Menu + MenuItems with `checked` and `shortcut`
- [ ] `PriorityPicker` rewritten the same way
- [ ] `EstimatePicker` rewritten the same way
- [ ] `LabelPicker` rewritten composing Menu + MenuFilter + MenuItems with `checked`
- [ ] Duplicate `CheckIcon` definitions removed (MenuItem already has one)
- [ ] All keyboard behavior preserved: arrow nav, number keys, Enter/Space, Escape
- [ ] Pickers work inside Popover, SubMenu `renderPanel`, and NewIssueModal
- [ ] No visual or behavioral regression
- [ ] `make test` passes (lint + tests)

## Flow

### Step 1 — Add `MenuFilter` to Menu.tsx

A non-navigable Menu child that renders a filter input + divider:

```tsx
interface MenuFilterProps extends MenuItemInternalProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
}
```

- Menu recognizes `MenuFilter` the same way it recognizes `MenuDivider`/`MenuLabel`: non-navigable, skipped by arrow keys
- Renders: input (`px-2 py-1.5 text-sm`, consistent with Menu item sizing) + border-t divider
- Auto-focuses via `requestAnimationFrame` on mount
- Menu's existing ArrowUp-past-top already does `querySelector("input")` and focuses it — MenuFilter just provides the input to find
- Escape on the input: bubbles to Menu's handler

Export from Menu.tsx and barrel.

### Step 2 — Add `bare` prop to Menu

When `bare` is true, Menu omits: `p-2`, `bg-[...]`, `border`, `rounded-[...]`, `shadow-[...]`. Keeps: `min-w-64`, `outline-none`, keyboard handling, focus management, scroll behavior.

Pickers pass `bare` since PopoverPanel or SubMenu flyout already provides chrome.

### Step 3 — Rewrite StatusPicker

Bare Menu with checked MenuItems. No filter input (7 items). Number keys via `shortcut`:

```tsx
export default function StatusPicker({ current, onSelect, onClose }: StatusPickerProps) {
  return (
    <Menu onClose={onClose} bare>
      {STATUS_OPTIONS.map((opt, i) => (
        <MenuItem
          key={opt.value}
          label={opt.label}
          icon={<StatusIcon status={opt.value} size={14} />}
          checked={opt.value === current}
          shortcut={String(i + 1)}
          onClick={() => onSelect(opt.value)}
        />
      ))}
    </Menu>
  );
}
```

Delete: local CheckIcon, filter state, focusIndex, inputRef, keyboard handler, item rendering.

### Step 4 — Rewrite PriorityPicker

Same pattern with `PRIORITY_OPTIONS` and `PriorityIcon`.

### Step 5 — Rewrite EstimatePicker

Same pattern. Map `ESTIMATE_OPTIONS` (plain `number[]`) to MenuItems using `estimateLabel()`. No icon.

### Step 6 — Rewrite LabelPicker

LabelPicker keeps its own file but composes Menu internally:

- Context consumers, label ordering, `canCreate` logic, color picker flow, `exclude` filtering — all stay
- Replace manual keyboard handler and focus management with `<Menu bare onClose={onClose}>` + `<MenuFilter>` + `<MenuItem>`
- Each label row: `<MenuItem icon={<LabelBadge />} label={label} checked={isActive} onClick={() => onToggle(label)} />`
- Divider between default and user labels: `<MenuDivider />`
- "Create label" button: `<MenuItem>` at the bottom
- Color-picker sub-view (`creatingLabel` state): renders color picker instead of Menu when active
- LabelPicker's checkbox/radio indicators replaced by MenuItem's standard CheckIcon — same visual language as all other pickers

### Step 7 — Update barrel export

`web/src/components/ui/index.ts`: export `MenuFilter`. Existing picker exports unchanged.

### Step 8 — Test

Run `make test` to verify lint + tests pass.

## Decisions

1. **No `Picker` component** — Menu already provides everything. The pickers are just Menu configurations.

2. **No filter input on simple pickers** — Status (7), Priority (5), Estimate (6) are fully visible. Menu's shortcut system handles number-key quick-select.

3. **No `checkStyle` prop** — MenuItem's existing CheckIcon works for both single-select and multi-select. Checkbox/radio indicators were a visual affordance in LabelPicker that added inconsistency, not clarity. Check marks are the universal pattern (macOS menus, VS Code, etc.).

4. **`MenuFilter` as a general-purpose Menu primitive** — Not picker-specific. Useful anywhere a Menu needs filtering (LabelPicker, FilterMenu submenus). Same recognition pattern as MenuDivider/MenuLabel.

5. **`bare` prop on Menu** — Pickers render inside containers that already provide chrome. *Alternative: always strip chrome — rejected because standalone Menu (ContextMenu) needs it built in.*

6. **Wrapper components preserved** — Pickers remain as named exports for semantic imports and zero consumer-side changes.

## Edge Cases

- **MEDIUM**: `bare` Menu inside SubMenu flyout — SubMenu applies chrome for `renderPanel` mode. Verify padding/spacing matches current appearance.
- **LOW**: Menu keyboard scope — switching from `onKeyDown` on the input to Menu's document-level handler shouldn't conflict since pickers are already in overlay contexts.

## Assumptions

- Menu's `querySelector("input")` ArrowUp-to-input works with MenuFilter's input
- MenuItem's shortcut matching handles "1"-"9" the same way current pickers' `parseInt(e.key)` does
- LabelPicker's `LabelBadge` works in MenuItem's `icon` slot without layout issues

## Open Questions

None.