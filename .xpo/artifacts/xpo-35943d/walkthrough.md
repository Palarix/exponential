# Walkthrough: Start Work disabled while blockers are unresolved

## What changed and why

xpo-a18d46 let CANCELED and DUPLICATE blockers unblock their dependents. That hid stale links: a blocker that was canceled or marked duplicate no longer means anything, or it points at the wrong issue. The board also offered Start Work for issues that `start` would refuse.

Now a `blocked_by` blocker counts as resolved only when it is DONE. The board, `start` and drive all follow that rule.

## Backend

- `model.BlockerResolved(status)` (`internal/model/types.go`) is the single definition: it returns true only for DONE. It is separate from `IsTerminal`, which still drives parent auto-close and cycle rollover.
- The start gate (`internal/exponential/update.go`) skips deleted or missing blockers, as before, and also blockers that `BlockerResolved` accepts. Any other blocker is refused, with one of two errors:
  - Open blocker: the old "blocked by incomplete issue Y" message.
  - CANCELED or DUPLICATE blocker: "blocked by Y, which is canceled; remove the link or point it at another issue."

  `StartWork` goes through `UpdateIssue`, so this one gate covers MCP, the CLI and the agent.
- `hasUnresolvedBlockers` in `drive.go` uses `BlockerResolved` for `blocked_by`, so drive never picks an issue that `start` would refuse. `depends_on` keeps `IsTerminal`, because `start` doesn't gate on it.

## Frontend

- `unresolvedBlockers(issue, issues)` (`relation-utils.ts`) returns the target issue of each `blocked_by` link that isn't DONE. `issue.dependencies` already includes derived links, so a link stored on the other issue counts too. Missing targets are skipped, as the backend does.
- `PropertySidebar` computes blockers only while the action is Start Work, so Continue Work is unaffected. While blockers remain, the button is natively `disabled` and gets `disabled:pointer-events-none`. A `<span>` around it is the Radix Tooltip trigger, because a disabled button fires no pointer events. The span gets `cursor-not-allowed`.
- `ui/Tooltip`'s `content` widened from `string` to `ReactNode`, so the tooltip can hold the `BlockerList`.
- `BlockerList` is exported for tests. It renders one row per blocker: status icon, "Blocked by xpo-…", and "(canceled)" or "(duplicate)" where that applies. Those stale rows get a **Remove link** button, which calls `saveUpdate` with `removeLink(deps, id, "blocked_by")`. Radix keeps a tooltip open while the pointer moves onto its content, so the button can be clicked. Once the issue's links update, the button re-renders enabled, with no reload.

## Decisions

- Drive uses DONE only for `blocked_by` (confirmed by the user).
- We use a Radix Tooltip around a wrapping span instead of a click popover (the user's call). Trade-off: keyboard users can't focus a disabled button, so they can't open the tooltip. The ✕ on the Relations card chip still removes the link from the keyboard.

## Acceptance criteria

- [x] An issue with an open, CANCELED or DUPLICATE blocker shows Start Work disabled, and its tooltip names each blocker and its status. Evidence: the `PropertySidebar Start Work blockers` tests (PLANNED, DOING, CANCELED, DUPLICATE) and the `BlockerList` tests.
- [x] The tooltip's Remove link clears a CANCELED or DUPLICATE blocker link, and Start Work enables without a reload. Evidence: Remove link renders only on stale rows (`BlockerList` test); `removeLink` is covered in `relation-utils.test.ts`; the "enables Start Work once every blocker is DONE" test. The user tested it in the board.
- [x] Informational links don't disable the button. Evidence: the "doesn't disable Start Work for informational links" test and the `unresolvedBlockers` "ignores informational links" test.
- [x] `start` refuses an issue with a CANCELED or DUPLICATE blocker, and the error says to remove the link. Evidence: `TestUpdateIssue_BlockerCanceledOrDuplicate_StillBlocks`.
- [x] Drive skips issues with CANCELED or DUPLICATE `blocked_by` blockers. Evidence: `TestHasUnresolvedBlockers_TerminalBlockers`.
- [x] Go and PropertySidebar tests cover open, done, canceled and duplicate blockers. `make test` and `go test ./...` pass.
