# Walkthrough: one issue hover card, everywhere

## What changed and why

The issue started narrow: once release rows show assignee, estimate and labels (xpo-df162c), the release row's hover card would mostly repeat the row. The first version replaced those fields with label/value rows (Priority, Activity, Last change, Work, Blocked by). In review that failed: the rows were too cramped ("Last change" showed little more than a name), and with no header you couldn't tell which issue a card belonged to.

A tophat page compared five layouts. The user picked the simplest one and then asked to make it the only hover card in the app, so an issue looks the same wherever it is hovered. Two spacing passes followed: first the gaps around the divider shrank, then the divider went away.

## The card (`web/src/components/IssueRef/IssuePreview.tsx`)

`IssuePreviewCard` has three parts:

1. **Header.** Status icon and ID on one line, the assignee's avatar in the top-right corner (name in its tooltip, nothing when unassigned), then the title in `text-sm`, left-aligned and clamped to two lines.
2. **Excerpt.** The description rendered with `IssueMarkdown` in a box capped at six lines. A layout effect compares `scrollHeight` with `clientHeight` and adds the existing `fade-bottom` mask only when the text is cut off, so short descriptions don't fade for no reason. The card is `pointer-events-none`, so it can't scroll: a click on the anchor opens the issue.
3. **Footer.** Icons with counts, muted, spread with `justify-between`, 8px below the excerpt (the same gap as title to excerpt). It has no divider: the divider made the space below the excerpt feel looser than the space above it. Each item appears only when it has something to count and carries a `title`/`aria-label` ("3 comments", "Blocked by 1 issue").

The `full` / `details` variants and the `variant` prop are gone. The three callers render the same card: `useIssueLink` (issue links and chips in Markdown and chat, and the chat title link), `Relations.tsx` (relation chips) and `ReleaseIssues.tsx` (release rows).

`footerItems` is a plain function, not a component, because the card calls `useIssueList()` itself and passes the counts in. An earlier draft called a `Footer` component as a function to find out whether it rendered anything; that would have broken the rules of hooks.

## Counting (`issue-preview-utils.ts`)

`previewCounts(issue, issues)` returns what the footer shows. It reuses existing rules instead of redefining them:

- sub-issues: `childProgress` from `releases-utils`, the same done/total the release and backlog rows show;
- blockers: `unresolvedBlockers` from `relation-utils`, the start gate's rule (only DONE resolves a blocker);
- spec and walkthrough: `artifact_type` `spec` / `walkthrough`; every other artifact counts as an attachment;
- comments and commits: `issue.comments` and `branch_stats.commits`.

Everything comes from the issue list payload, which already carries comments, artifacts, links and branch stats. The card fetches nothing. The first version fetched the issue's history for "Last change"; that and the matching `issueHistoryQuery` refactor were removed with the design change.

## Styling (`index.css`)

`.prose-exponential.prose-preview` sets xs text on the xs line height, so "six lines" is a fixed `calc(6 * var(--text-xs--line-height))` box. Its heading rule uses `:where(h1, h2, h3, h4)` to keep the specificity low enough that `.prose-exponential > :first-child { margin-top: 0 }`, which comes later in the file, still removes the top margin of a leading heading.

## What a future reader should know

- The issue-link card used to show the status and priority as text. It now shows the status as an icon only and no priority. That was accepted with the change to one card.
- The demo data has no branches, so the commit count was checked through tests and a made-up issue on the tophat page, not on a real board.

## Acceptance criteria

- [x] The card shows the status icon, ID, the assignee's avatar top-right, and the title (left-aligned, up to 2 lines). Evidence: test "shows the ID, title and assignee avatar…" (avatar `title` present, name not rendered as text) and "has no avatar when unassigned"; user's screenshots on the demo board.
- [x] It shows a Markdown excerpt, cut at 6 lines with a fade when longer; no excerpt for an empty description. Evidence: tests "shows the description as a clamped Markdown excerpt" (`<strong>` rendered) and "has no excerpt without a description"; user's screenshot of a two-paragraph excerpt.
- [x] Below the excerpt (no divider), it shows icons with counts for comments, attachments, spec, walkthrough, sub-issues, commits and open blockers, each only when present; no footer without any. Evidence: test "shows a footer item for each thing the issue has" checks all seven `aria-label`s; "leaves out footer items with nothing to count"; "has no footer when there is nothing to count"; `previewCounts` unit tests (attachments vs spec/walkthrough, done/total sub-issues, DONE blockers excluded); the rich tophat card, which the user picked.
- [x] It no longer shows the assignee's name, priority, estimate or labels. Evidence: the first card test asserts no "High", no "Frontend" and no `>Ada Lovelace<`.
- [x] There is one card: no `variant` prop; issue links/chips, the chat title link, relation chips and release rows all show it. Evidence: `IssuePreviewVariant` and `variant` removed; `tsc` builds all three callers; the user checked it on the demo board.
- [x] The `IssueRow` doc comment in `ReleaseIssues.tsx` describes the new card.
- [x] Tests cover the header, excerpt and each footer item; `make test` passes (1548 frontend tests, lint, Go tests).
