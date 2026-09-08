# Fix uneven hover/selection backgrounds in filter menu items

## What

Scrollbar thumbs inside menus/popovers are nearly invisible in dark mode, making it look like item hover backgrounds don't extend full width.

## Root Cause

Global scrollbar thumb used `--color-border-default` (#282828), which is only 6 hex steps from `--color-surface-3` (#222222) — the menu/popover background. The scrollbar occupies 6px of layout space but is invisible, creating the illusion of uneven item widths.

## Fix

Change global scrollbar thumb color from `--color-border-default` to `--color-border-overlay` (#323232) in `index.css`. This token is semantically intended for popover/modal edges and provides enough contrast to be visible against dark surfaces without being jarring on lighter ones.

The comprehensive fix (`scrollbar-gutter: stable` for consistent item widths in scrollable menus) is deferred to xpo-5d1c38 (Popover primitive) and xpo-de8d06 (Menu system) where it can be applied once to all menus.

## Acceptance Criteria

- [x] Scrollbar thumbs are visible inside menus and popovers in dark mode
- [x] Scrollbar thumbs are not overly bright on non-menu surfaces
- [x] `make lint` passes
