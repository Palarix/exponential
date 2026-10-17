# Walkthrough: Start Work disabled while the issue is in BACKLOG

## What changed

`start` refuses a BACKLOG issue (`internal/exponential/start.go`: "move it to PLANNED before starting"). The sidebar still offered Start Work for one, so clicking it only produced an error toast. The button is now disabled in BACKLOG, and its tooltip says why. This is a frontend-only change in `web/src/components/IssueDetail/PropertySidebar.tsx`.

## How it fits together

- `workAction` is unchanged: BACKLOG and PLANNED both show "Start Work". The button stays visible so the user can see the next step, but it can't be used.
- Two new values sit next to the existing `blockers` memo:
  - `inBacklog` is true when the action is Start Work and the status is BACKLOG.
  - `startDisabled` is `inBacklog || blockers.length > 0`.
- The button is natively `disabled` when `starting || startDisabled`.
- When `startDisabled`, the button gets the wrapper from xpo-35943d: a `<span className="block cursor-not-allowed">` that triggers the Radix Tooltip. The span is needed because a disabled button fires no pointer events.
- The tooltip renders a new `StartWorkBlockedReasons` component:
  - The BACKLOG row comes first: a BACKLOG status icon and "Move the issue to Planned to start work".
  - Then `BlockerList`, unchanged, when there are blockers.
- Blockers are still computed in BACKLOG, so an issue with both problems shows both reasons at once. The user doesn't fix one only to hit the other.

## Decisions

- The BACKLOG row has no "Move to Planned" button, unlike Remove link on stale blockers. The status picker sits just below the button (user's call).
- xpo-e627e1 (no git repo) stays a separate issue. Its main fix is in the backend `StartWork`, and the board needs a new server signal for it. Its UI hint can be added as another row in `StartWorkBlockedReasons`.

## Acceptance criteria

- [x] A BACKLOG issue shows Start Work disabled, and clicking it does nothing. Evidence: test "disables Start Work while the issue is in BACKLOG" checks `disabled=""` and the `cursor-not-allowed` wrapper. A disabled button fires no click, so no request or toast.
- [x] The tooltip on the disabled button says "Move the issue to Planned to start work". Evidence: test "says to move a BACKLOG issue to Planned".
- [x] A BACKLOG issue with an unresolved blocker shows both reasons. Evidence: test "lists both reasons, BACKLOG first…".
- [x] A PLANNED issue with no blockers still has Start Work enabled. Evidence: test "enables Start Work for a PLANNED issue without blockers". The existing blocker tests still pass.
- [x] Moving the issue to Planned enables the button with no reload. Evidence: `inBacklog` is derived from `issue.status` on every render, so a status update re-enables the button. There is no dedicated test for the transition, and it was not checked on a live board before merge.
- [x] PropertySidebar tests cover BACKLOG, BACKLOG with a blocker, and PLANNED. Evidence: `PropertySidebar.test.tsx`; `make test` passes.
