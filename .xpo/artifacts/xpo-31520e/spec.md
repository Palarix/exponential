# Spec: Start Work disabled while the issue is in BACKLOG

## What

The sidebar shows Start Work disabled for a BACKLOG issue. Its tooltip says why: "Move the issue to Planned to start work." If the issue also has unresolved blockers, the tooltip lists both reasons.

## Why

`start` refuses a BACKLOG issue (`start.go`: "move it to PLANNED before starting"). The board still offers Start Work for it, so clicking it only produces an error toast.

## How

Frontend only, in `PropertySidebar.tsx`. It reuses the xpo-35943d pattern.

- `workAction` is unchanged. BACKLOG and PLANNED both show "Start Work".
- `inBacklog = workAction === "Start Work" && issue.status === "BACKLOG"`.
- `startDisabled = inBacklog || blockers.length > 0`. The button is natively `disabled` when `starting || startDisabled`.
- When `startDisabled`, a `<span className="block cursor-not-allowed">` wraps the button and triggers the Radix Tooltip, the same as for blockers.
- Tooltip content, a new `StartWorkBlockedReasons` component:
  - BACKLOG row first: a BACKLOG status icon and "Move the issue to Planned to start work".
  - Then the existing `BlockerList`, when there are blockers.
- Blockers are still computed while in BACKLOG, so both reasons show together.
- Continue Work is unchanged.

## Acceptance criteria

- [ ] A BACKLOG issue shows Start Work disabled, and clicking it does nothing (no request, no toast).
- [ ] The tooltip on the disabled button says "Move the issue to Planned to start work".
- [ ] A BACKLOG issue with an unresolved blocker shows both reasons in the tooltip.
- [ ] A PLANNED issue with no blockers still has Start Work enabled.
- [ ] Moving the issue to Planned enables the button with no reload.
- [ ] PropertySidebar tests cover BACKLOG, BACKLOG with a blocker, and PLANNED.

## Decisions

- The BACKLOG row has no Move to Planned button. The status picker sits just below the button (user's call).
