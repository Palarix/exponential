# Walkthrough: Walkthrough tab in the merge view

## What changed

Added a "Walkthrough" tab to `web/src/components/IssueDetail/MergeView.tsx` that renders the issue's `walkthrough.md` as Markdown. When a walkthrough artifact exists, it becomes the default active tab — the reviewer sees the narrative explanation before diving into diffs.

## How it works

The implementation reuses patterns already established in `IssueDetail.tsx`:

1. **Detection**: `hasWalkthrough` checks `issue.artifacts` for a `walkthrough` artifact type — same boolean used in the detail view.

2. **Data fetching**: Walkthrough content is fetched in the existing `Promise.all` data-load block via `fetchArtifactContent(issue.id, "walkthrough.md")`. When no walkthrough exists, the fetch is skipped (`Promise.resolve("")`).

3. **Tab rendering**: The walkthrough tab is rendered before the existing tabs (Commits, Files, Conversation) and only appears when `hasWalkthrough` is true. It uses a `BookOpen` icon instead of a count badge since walkthroughs are singular.

4. **Content rendering**: Uses `react-markdown` with `remark-gfm` and `remark-breaks` plugins, wrapped in a `prose-exponential` div — identical to how walkthroughs are rendered in the detail view.

5. **Default tab**: `useState<Tab>` initializes to `"walkthrough"` when present, falling back to `"files"` otherwise.

## Key decisions

- No backend changes required — the `GET /api/issues/{id}/artifacts/{filename}` endpoint and `fetchArtifactContent` client function already existed.
- Tab is placed first in the tab bar to reinforce that the walkthrough is the recommended starting point for review.
- No count badge on the walkthrough tab (unlike Commits/Files/Conversation) since there's always exactly one walkthrough.
