# Walkthrough Tab in the Merge View

## What

Add a "Walkthrough" tab to `MergeView.tsx` that renders the issue's `walkthrough.md` content as Markdown. When a walkthrough exists, it becomes the default active tab.

## Why

The walkthrough is the durable implementation record. Currently it's only visible in the IssueDetail view — reviewers opening the merge view to review changes before merging never see it unless they close the merge view and switch tabs. Surfacing it in the merge view makes it the natural pre-merge reading.

## How

### File: `web/src/components/IssueDetail/MergeView.tsx`

1. **Extend the `Tab` type** (line 21): Add `"walkthrough"` to the union:
   ```typescript
   type Tab = "files" | "commits" | "conversation" | "walkthrough";
   ```

2. **Detect walkthrough presence**: Check `issue.artifacts` for a walkthrough artifact (same pattern as `IssueDetail.tsx` line 356):
   ```typescript
   const hasWalkthrough = issue.artifacts?.some((a) => a.artifact_type === "walkthrough");
   ```

3. **Default tab**: Change the initial `activeTab` state (line 29) to `hasWalkthrough ? "walkthrough" : "files"`.

4. **Fetch walkthrough content**: Add a `walkthroughContent` state variable. In the existing `useEffect` data-fetch block (lines 55-72), conditionally fetch walkthrough content using the existing `fetchArtifactContent(issue.id, "walkthrough.md")` from `client.ts`.

5. **Tab header**: Add a "Walkthrough" tab button in the tab bar (lines 238-251). No count badge — walkthroughs are singular. Only render when `hasWalkthrough` is true.

6. **Tab content**: Add a `WalkthroughTab` case in the content rendering (lines 253-267). Render with `react-markdown` using `remark-gfm` and `remark-breaks` plugins, wrapped in a `prose-exponential` div — identical to the pattern in `IssueDetail.tsx` lines 439-455.

### No backend changes required

The `GET /api/issues/{id}/artifacts/{filename}` endpoint and `fetchArtifactContent` client function already exist and work for walkthroughs.

## Acceptance Criteria

- [ ] Walkthrough tab appears in the merge view when the issue has a walkthrough artifact
- [ ] Tab renders walkthrough markdown with full `prose-exponential` styling
- [ ] Walkthrough tab is the default active tab when present
- [ ] Tab is hidden when no walkthrough exists
- [ ] Default tab falls back to "Files changed" when no walkthrough exists
- [ ] Existing merge view behavior unchanged for issues without walkthroughs
