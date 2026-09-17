# Walkthrough: Rewrite pickers as Menu compositions

## Summary

Replaced four near-identical picker components (StatusPicker, PriorityPicker, EstimatePicker, LabelPicker) with compositions of the existing Menu + MenuItem system. Added two primitives to Menu (`MenuFilter`, `bare` prop) and registered digit keys for shortcut matching. Net result: -233 lines, zero new components, and the pickers now share keyboard navigation, focus management, and selection indication with every other menu in the app.

## What was built

### Menu system additions (Menu.tsx)

**`MenuFilter`** — a non-navigable Menu child that renders a filter input with a divider below it. Menu recognizes it like `MenuDivider` and `MenuLabel`: arrow keys skip it, but Menu's existing ArrowUp-past-top logic (`querySelector("input")`) focuses it when the user navigates above the first item. Auto-focuses via `requestAnimationFrame` on mount.

**`bare` prop** — when true, Menu strips its visual chrome (background, border, shadow, padding, rounded corners) and keeps only `outline-none` + keyboard/focus behavior. This exists because pickers render inside PopoverPanel or SubMenu flyouts that already provide chrome — without `bare`, you'd get double borders and double backgrounds.

**Input-focus bypass** — when a focused input is detected inside the menu container, the keyboard handler only processes ArrowDown (enter the item list) and Escape (close). All other keys pass through for typing. Without this, Menu's handler would `preventDefault()` on Space and character keys, blocking filter input.

**Digit key registration** — `useKeyboardHandler` previously only registered `a-z` as character shortcuts. Number keys (`0-9`) were silently ignored because they never reached the handler. Extended the registration to `a-z0-9` so MenuItem `shortcut="1"` through `shortcut="9"` actually work.

### Picker rewrites

**StatusPicker, PriorityPicker, EstimatePicker** — each reduced from ~97 lines to ~20 lines. They render a bare `<Menu>` with `<MenuItem>` children. Each item uses `checked` for the current-value indicator (MenuItem's built-in CheckIcon) and `shortcut` for number-key quick-select. The filter input was removed entirely — these are short, fully-visible lists (5-7 items) where Menu's shortcut system replaces the old input-driven number-key handling.

The onClick on each item explicitly calls both `onSelect(value)` and `onClose()`. This is necessary because MenuItem treats items with `checked` as checkboxes and skips auto-close — correct for multi-select (LabelPicker) but wrong for single-select pickers that should dismiss after selection.

**LabelPicker** — reduced from 263 to ~150 lines. Composes `<Menu bare autoFocus={false}>` + `<MenuFilter>` + `<MenuItem>` children. Everything unique to LabelPicker stays: context consumers (`LabelColorsContext`, `HideDefaultLabelsContext`, `DefaultLabelsContext`), label ordering (defaults first), exclude filtering, `canCreate` logic, the color picker flow, and `handleCreateLabel`. The manual keyboard handler, focus state, and input ref were all removed — Menu handles them.

Each label row is a `<MenuItem>` with a colored dot icon (matching the label's configured color via `labelColor()`) and `checked` for selection state. The previous checkbox/radio indicators were replaced by MenuItem's standard CheckIcon — consistent with all other pickers. `LabelBadge` was not used in the icon slot because it includes its own text, which would duplicate MenuItem's `label` text.

The divider between default and user labels uses `<MenuDivider>`. The "Create" option is a plain `<MenuItem>`. The `singleSelect` prop is still accepted in the interface for API compatibility but no longer affects rendering.

## Architecture decisions

**Pickers are Menu compositions, not a separate `Picker` component.** During spec discussion we identified that a generic Picker would reimplement what Menu already provides — focus, keyboard nav, checked state, shortcuts. The only missing pieces were a filter input primitive and chrome-free mode, both general-purpose additions to Menu.

**No filter on simple pickers.** Status (7 items), Priority (5), and Estimate (6) are fully visible. Menu's shortcut system provides the same quick-select as the old number-key input handler, without the UI weight of a filter field.

**`bare` as opt-in, not default.** Standalone menus (ContextMenu) need built-in chrome. Pickers opt into `bare` because their container already provides it.

## Acceptance criteria

- [x] `MenuFilter` component added to Menu.tsx, non-navigable, with auto-focus
- [x] `bare` prop on Menu strips chrome (bg, border, shadow, padding, rounded)
- [x] `StatusPicker` rewritten as bare Menu + MenuItems with `checked` and `shortcut`
- [x] `PriorityPicker` rewritten the same way
- [x] `EstimatePicker` rewritten the same way
- [x] `LabelPicker` rewritten composing Menu + MenuFilter + MenuItems with `checked`
- [x] Duplicate `CheckIcon` definitions removed (MenuItem already has one)
- [x] All keyboard behavior preserved: arrow nav, number keys, Enter/Space, Escape — verified by user tophat
- [x] Pickers work inside Popover, SubMenu `renderPanel`, and NewIssueModal
- [x] No visual or behavioral regression (user confirmed "lgtm")
- [x] `make test` passes (lint + 357 frontend tests + all Go packages)