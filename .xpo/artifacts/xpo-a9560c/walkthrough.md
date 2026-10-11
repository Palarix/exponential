# Walkthrough: Agent-on-behalf-of attribution with a dashed underline

## The rule
When an agent acted for a principal, the UI shows **the principal's name with a dashed underline**, and hovering or focusing it reveals the agent ("Claude Code for Nicolas Bettenburg"). When the principal acted on their own, the name is plain. This rule applies everywhere, and there is no other attribution style.

## How we got here
The first version applied the underline only where a hover already existed (system events). The spelled-out "Claude Code on behalf of Name" comment headers were kept, which was a deliberate xpo-35fc16 choice. In review, the mix of styles read as inconsistent on both the Dashboard and the issue Activity, so the user made it one app-wide rule. That retires the spelled-out headers and extends the rule to the Inbox and the sidebar assignee.

## The shared piece
- `web/src/components/ui/actor-name-utils.ts`: `actorNameDisplay(name, via?, className?, { focusable })` is the single source for label, tooltip, sr-only text, classes and focusability. It's a pure function, so it's testable under vitest (the repo has no DOM renderer).
  - Underline: `underline decoration-dashed decoration-1 underline-offset-[3px]` in the muted text token, so it works in both themes. There's also a focus-visible ring and `cursor-help`.
  - Tooltip: the existing `agentTooltip`. sr-only text: " (via Claude Code)", so assistive tech doesn't depend on hover.
  - `name` and `via` can be raw identities or already-formatted names, because `shortName` and `agentDisplayName` are idempotent.
- `web/src/components/ui/ActorName.tsx` renders that inside the existing Radix `Tooltip`. `tabIndex=0` lets keyboard focus open the tooltip. `focusable={false}` is for names inside a control that already takes focus.

## Surfaces
- **Issue Activity timeline** (`ActivityTimeline.tsx`): system rows and comment headers. The tooltip moved from avatar + name to the name only, so the underline is the hover target.
- **Merge view** comments (`MergeView.tsx`).
- **Dashboard feed** and **Timeline** (`ActivityFeed.tsx`, `Timeline.tsx`): the inline tooltip span is replaced by `ActorName`. COMMENT rows now use the same `name` element as every other event type.
- **Inbox banner** (`inbox-utils.ts`, `Inbox.tsx`): it used to push the raw agent ("claude-code/… (You)") into the summary strings. `buildChangeSummary` now returns `{ actor, parts }`, and `summaryActor()` gives `{ principal, via, isYou }`. The banner renders the actor with `ActorName`, then " (You)", then the parts.
- **Sidebar assignee** (`PropertySidebar.tsx`): `ActorName` with `assignee_via` and `focusable={false}`, because it sits inside the picker `Button` and a nested tab stop would be wrong. The button's accessible name picks up the sr-only text.
- `commentAuthor()` now returns `{ identity, name, via? }` (the same as `activityActor`). `agentCommentAuthor` and `agentLabel` are removed.

## Deliberately excluded
- Dashboard assignee distribution and workload rows aggregate many issues per person, so there is no single via.
- Artifact list `updated_by`: artifacts don't store the principal. Follow-up: xpo-61ff1e.

## Related bug found in review
Agent merges showed no underline because MCP `merge` recorded MERGE and DONE as the human. That was a data bug, fixed in xpo-40ab84 (missing `syncLocal()`).

## Acceptance criteria
- [x] Every scoped surface renders on-behalf-of principals with a dashed underline, and the agent is revealed on hover, focus and sr-only text. Verified by the user in the UI ("lgtm").
- [x] Direct principal actions render without an underline or tooltip (`actorNameDisplay` direct-case tests)
- [x] No spelled-out "X on behalf of Y" remains in the web UI (grep of `web/src` is clean)
- [x] The Inbox banner shows the principal, with "(You)" for the viewer (`summaryActor` / `buildChangeSummary` tests)
- [x] The sidebar assignee is underlined when `assignee_via` is set
- [x] Implemented through the single shared `ActorName` component
- [x] Frontend tests cover the on-behalf and direct cases (`ActorName.test.ts`, `format.test.ts` `commentAuthor`, `inbox-utils.test.ts`)
- [x] `make test` passes (552 frontend tests plus Go) and `make frontend` builds
