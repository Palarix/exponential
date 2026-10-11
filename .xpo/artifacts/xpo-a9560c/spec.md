# Spec: Render agent-on-behalf-of-user author names with a dashed underline

## Rule (app-wide)
When an **agent acted on behalf of a principal**, show the **principal's name with a dashed underline**. Hovering or focusing it reveals the agent ("Claude Code for Nicolas Bettenburg"). When the principal **acted on their own**, show the plain name with no underline and no tooltip.

There are no other attribution styles: spelled-out "Claude Code on behalf of Name" headers are removed.

## Why
Since xpo-35fc16 the display is principal-first, so "I did this" and "my agent did this for me" looked identical. The spelled-out comment headers then made the app inconsistent: some agent actions named the agent, others hid it. One subtle cue, used everywhere, fixes both.

## Scope
1. Issue detail **Activity timeline**: system rows *and* comment headers (`IssueDetail/ActivityTimeline.tsx`)
2. **Merge view** comment headers (`IssueDetail/MergeView.tsx`)
3. Dashboard **Activity feed**: all rows, including COMMENT (`Dashboard/ActivityFeed.tsx`)
4. **Timeline** view: all rows, including COMMENT (`Timeline/Timeline.tsx`)
5. **Issue sidebar assignee** (`IssueDetail/PropertySidebar.tsx`): `assignee` with `assignee_via` is rendered through `ActorName`. The name sits inside the picker `Button`, so `ActorName focusable={false}` avoids a nested focusable element; the button itself is focusable and the sr-only text becomes part of its accessible name. Dashboard assignee/workload rows are excluded: they aggregate many issues per person, so there is no single via.
6. **Inbox** change-summary banner (`Inbox/inbox-utils.ts`, `Inbox.tsx`). Today it shows the raw agent ("claude-code/2.1.294 (You)"). It changes to the principal (the existing "(You)" marker is kept for the viewer's own principal) rendered through `ActorName`.

## How
- Shared `ActorName` component (`ui/ActorName.tsx`) with its logic in `ui/actor-name-utils.ts` (`actorNameDisplay`):
  - `via` absent: plain span, no tooltip, not focusable.
  - `via` present: dashed underline (`underline decoration-dashed decoration-1 underline-offset-[3px]`, muted token, works in light and dark), tooltip `agentTooltip(via, name)`, `tabIndex=0` with a focus-visible ring, and sr-only " (via Claude Code)".
  - `focusable` prop (default true) for use inside interactive controls.
  - Accepts a raw identity or an already-formatted display name for both `name` and `via`.
- Comments: `commentAuthor()` returns `{ identity, name, via? }` (principal-first, matching `activityActor`). Comment headers render `<ActorName name via />` inside their existing `Text` styling. `agentCommentAuthor` is removed together with its tests.
- Feed and Timeline COMMENT rows reuse the same `name` element as every other event type.
- Inbox: `buildChangeSummary` returns `{ actor?: { principal, via, isYou }, parts: string[] }` instead of putting the actor into the string list. The banner renders the actor with `ActorName` (followed by " (You)" where it applies), then the parts joined with " · ".
- Activity timeline: the tooltip sits on the name only. The avatar is unchanged.
- Tooltip wording: "Claude Code for Name" (confirmed).

## Acceptance criteria
- [ ] Every scoped surface renders on-behalf-of principals with a dashed underline, and the agent is revealed on hover and focus (and via sr-only text)
- [ ] Direct principal actions render without an underline and without a tooltip
- [ ] No spelled-out "X on behalf of Y" remains in the web UI
- [ ] The Inbox banner shows the principal (with "(You)" when it is the viewer), not the raw agent
- [ ] The sidebar assignee is underlined when `assignee_via` is set
- [ ] Implemented through the single shared `ActorName` component
- [ ] Frontend tests cover the on-behalf and direct cases (`actorNameDisplay`, `commentAuthor`, `buildChangeSummary`)
- [ ] `make test` passes

## Out of scope
- Artifact list `updated_by`: artifacts don't store the principal. Follow-up: xpo-61ff1e.

## Related
- xpo-40ab84: MCP `merge` recorded MERGE and DONE as the human with no `on_behalf_of`, so agent merges showed no underline. That was a data bug, fixed separately.
