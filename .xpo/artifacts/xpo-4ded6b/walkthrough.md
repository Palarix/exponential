# Walkthrough: Registry-driven keyboard help

## Outcome

Keyboard Help is now a live view of the central keyboard registry rather than a separately maintained shortcut catalog. It shows global shortcuts together with registrations from the currently mounted view and controls, so its contents automatically follow application context.

## Registration lifecycle

`KeyboardHelp` remains mounted even when its dialog is closed. It registers `?` through `useKeyboardShortcuts`, making the help toggle behave like every other application shortcut.

When the dialog opens, the component also registers Escape at overlay priority. That binding is marked `showInHelp: false`, so it controls the dialog without documenting itself as an application command. Overlay priority prevents keys from leaking to underlying views.

Global navigation stays registered while Help is open. Consequently sequences such as `g` then `i` remain in the active metadata snapshot, but the open overlay prevents them from executing.

## Registry-driven presentation

The static shortcut arrays in `KeyboardHelp` were removed. `useActiveShortcuts()` supplies callback-free metadata containing each active shortcut's identifier, scope, label, group, key sequence, modifiers, and priority.

`keyboard-help-utils.ts` turns that metadata into display groups, puts Global first, combines aliases sharing a label into one row, and formats named keys, sequences, and platform modifiers.

Labels on Backlog, Board, and Issue Detail registrations were made user-facing because those registered labels are now the canonical help text. The command-palette entry displays the platform-appropriate Meta or Ctrl binding.

## Layout

Scope groups are stacked vertically at full width, keeping their context without assigning sparse groups to separate columns. Within each group, CSS multi-column layout balances entries responsively across one, two, or three columns. Multi-column flow gives the intended reading order: top-to-bottom within a column, then continuing to the right.

## Verification

- `make test`: passes, including 274 frontend tests and the backend suite
- `bun run build`: passes
- `bun run lint`: passes
- `git diff --check`: passes

Tests cover grouping, aliases, key formatting, scope fallback, active metadata publication, and the distinction between keeping underlying shortcuts visible while an overlay blocks their execution.
