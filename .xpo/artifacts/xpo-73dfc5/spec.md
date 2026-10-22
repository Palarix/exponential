# Issue hover card: header, excerpt, icon/count footer, everywhere

## What

`IssuePreviewCard` (`web/src/components/IssueRef/IssuePreview.tsx`) becomes one card: a header (status icon, ID, assignee avatar, title), a clamped Markdown excerpt of the description, and a footer of icons with counts. Priority, estimate and labels leave the card; the assignee stays only as an avatar in the corner.

It is the one issue hover card in the app. The `full` / `details` variants and the `variant` prop go away; every caller gets this card:
- issue links and chips through `useIssueLink` (`IssueRef` in Markdown and chat, the chat title's issue link),
- relation chips in the issue sidebar (`Relations.tsx`),
- release issue rows (`ReleaseIssues.tsx`).

## Why

With xpo-df162c the release row shows assignee, estimate and labels, so the old card would mostly repeat it. The first version of this issue used label/value rows (Priority, Activity, Last change, Work, Blocked by). In review the rows were too cramped to read ("Last change" showed little more than a name), and without a header you couldn't tell at a glance which issue the card was for. Tophat compared five layouts; the user picked the simplest: header, excerpt, and a footer of icons and counts. The user then asked for it to be the canonical card everywhere, so an issue looks the same wherever it is hovered.

## Layout

```
┌──────────────────────────────────────────┐
│ ◐ xpo-a1b2c3                          (A) │
│ Title, left-aligned, up to two lines     │
│ Description excerpt, Markdown, at most 6 │
│ lines, fading out at the bottom when it  │
│ is cut off…                              │
│ 💬 3  📎 1  📄  📖  ☰ 2/5  ⑂ 4  ⛔ 1        │
└──────────────────────────────────────────┘
```

- **Header**: status icon (12px) and ID (mono, muted) on one line, with the assignee's avatar (`sm`, name in its tooltip) in the top-right corner; no avatar when unassigned. Below it the title, `text-sm`, left-aligned with the card edge, clamped to 2 lines.
- **Excerpt**: `issue.description` rendered with `IssueMarkdown` in a `prose-preview` box (xs text on xs lines; headings at text size), capped at 6 lines with `overflow-hidden`. The `fade-bottom` mask is applied only when the content is taller than the box (measured in a layout effect). No scrolling: the card stays `pointer-events-none`; a click on the anchor opens the issue. No description: no excerpt.
- **Footer**: no divider (it broke the vertical rhythm in review); the footer sits 8px below the excerpt, the same gap as title to excerpt. Icons with counts, muted, spread evenly across the card width (`justify-between`), in this order. Each shows only when it has something to count; each has a `title`/`aria-label` naming it.
  | Icon | Shows | Source |
  |---|---|---|
  | `MessageSquare` | comments | `issue.comments.length` |
  | `Paperclip` | attachments | `issue.artifacts` other than spec and walkthrough |
  | `FileText` | has a spec (no count) | artifact_type `spec` |
  | `BookOpen` | has a walkthrough (no count) | artifact_type `walkthrough` |
  | `ListTree` | sub-issues done/total | `childProgress(issue, issues)` |
  | `GitCommitHorizontal` | commits | `branch_stats.commits` |
  | `OctagonMinus` (error color) | open blockers | `unresolvedBlockers(issue, issues)` |
  No footer items: no footer.
- The issue list for sub-issues and blockers comes from `useIssueList()`. Everything is in the list payload: no extra fetch.

## Out of scope

- Priority, activity, the last change, line counts, and the "updated X ago" time.
- Backend changes.

## Acceptance criteria

- [ ] The card shows the status icon, ID, the assignee's avatar top-right, and the title (left-aligned, up to 2 lines).
- [ ] It shows a Markdown excerpt, cut at 6 lines with a fade when longer; no excerpt for an empty description.
- [ ] Below the excerpt (no divider), it shows icons with counts for comments, attachments, spec, walkthrough, sub-issues, commits and open blockers, each only when present; no footer without any.
- [ ] It no longer shows the assignee's name, priority, estimate or labels.
- [ ] There is one card: no `variant` prop; issue links/chips, the chat title link, relation chips and release rows all show it.
- [ ] The `IssueRow` doc comment in `ReleaseIssues.tsx` describes the new card.
- [ ] Tests cover the header, excerpt and each footer item; `make test` passes.
