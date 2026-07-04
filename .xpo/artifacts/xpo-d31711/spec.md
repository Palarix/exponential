## Problem

The global command dialog (command palette / search dialog) is too narrow compared to other dialogs in the application.

## Requirements

- Increase the width of the global command dialog to match the width of the "create new issue" dialog.
- Only the width should change; do not alter height, max-height, or other layout behavior of the command dialog.
- The dialog should remain centered and responsive (i.e., still work on smaller viewports).

## Acceptance Criteria

- [ ] The global command dialog renders at the same width as the create-new-issue dialog.
- [ ] The command dialog remains centered on screen.
- [ ] On viewports narrower than the new width, the dialog scales down gracefully (no horizontal overflow).
- [ ] No visual regressions in the create-new-issue dialog or other dialogs.