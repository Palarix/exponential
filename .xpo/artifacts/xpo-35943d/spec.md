# Spec: Start Work disabled while blockers are unresolved

## What

A `blocked_by` blocker is resolved only when it is DONE. Until then, the sidebar shows Start Work disabled with a tooltip that names each unresolved blocker, and `start` refuses the issue. CANCELED and DUPLICATE blockers still block, and the tooltip offers Remove link for them.

## Why

xpo-a18d46 made CANCELED and DUPLICATE blockers unblock their dependents. That hides a stale link: the blocker no longer means anything, or it points at the wrong issue. The board also offers Start Work for issues `start` will refuse.

## How

### Backend

- `model.BlockerResolved(status) bool`: true only for DONE.
- `update.go` start gate: uses `BlockerResolved` instead of `IsTerminal`. Deleted or missing blockers are still ignored.
- Error wording:
  - Open blocker: `cannot start issue X: blocked by incomplete issue Y` (unchanged).
  - CANCELED/DUPLICATE: `cannot start issue X: blocked by Y, which is canceled; remove the link or point it at another issue`.
- `drive.go` `hasUnresolvedBlockers`: `blocked_by` uses `BlockerResolved`, so drive never picks an issue that `start` refuses. `depends_on` keeps `IsTerminal`.

### Frontend

- `unresolvedBlockers(issue, issues)` in `relation-utils.ts` returns the target issue of each `blocked_by` link whose target exists, isn't deleted, and isn't DONE. Missing targets are skipped, the same as the backend.
- `ui/Tooltip`: `content` widens from `string` to `ReactNode`.
- PropertySidebar: when `workAction === "Start Work"` and there are unresolved blockers, the button is natively `disabled`. It is wrapped in a `<span>` that is the Radix Tooltip trigger, because a disabled button fires no pointer events.
- Tooltip content: one row per blocker, with its status icon and "Blocked by xpo-…", plus " (canceled)" or " (duplicate)" where that applies. CANCELED and DUPLICATE rows get a **Remove link** button, which calls `saveUpdate({ dependencies: removeLink(deps, id, "blocked_by") })`. Radix keeps the tooltip open while the pointer is over it, so the button can be clicked. The button enables once the issue's links update, with no reload.
- Keyboard users can't focus a disabled button, so they can't open the tooltip. The Relations card's ✕ still removes the link from the keyboard.
- Continue Work is unchanged.

## Acceptance criteria

- [ ] An issue with an open, CANCELED or DUPLICATE blocker shows Start Work disabled. Its tooltip names each blocker and its status.
- [ ] The tooltip's Remove link clears a CANCELED or DUPLICATE blocker link, and Start Work enables without a reload once no unresolved blockers remain.
- [ ] Informational links (`relates_to`, `depends_on`, …) don't disable the button.
- [ ] `start` refuses an issue with a CANCELED or DUPLICATE blocker, and the error says to remove the link.
- [ ] drive skips issues with CANCELED or DUPLICATE `blocked_by` blockers.
- [ ] Go and PropertySidebar tests cover open, done, canceled and duplicate blockers.

## Decisions

- Drive also counts only DONE for `blocked_by` (confirmed by user).
- The tooltip is a Radix Tooltip around a span that wraps the disabled button, not a click popover (user's call).
