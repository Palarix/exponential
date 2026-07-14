# CMD+K Search Not Matching on Issue Titles Correctly

## Problem

The command palette uses a fuzzy subsequence matcher (`fuzzyMatch`) for issue title search. Typing "trail" matches any title where t, r, a, i, l appear in order anywhere — producing 61 false positives out of 300 issues. Since results are capped at 12, the actual trail-related issues never appear.

## Root Cause

`fuzzyMatch` is a character-by-character subsequence matcher. It's appropriate for short, fixed-vocabulary items (navigation commands, action labels) but far too loose for issue titles which are long, natural-language strings.

## Fix

Use case-insensitive substring matching (`includes`) for issue titles and IDs. Keep fuzzy matching for actions and navigation labels (short, predictable strings).

Change line 99 in `CommandPalette.tsx`:

```typescript
// Before:
issues.filter(i => fuzzyMatch(i.title, query) || i.id.includes(query.toLowerCase()) || i.labels?.some(l => fuzzyMatch(l, query)))

// After:
issues.filter(i => i.title.toLowerCase().includes(query.toLowerCase()) || i.id.includes(query.toLowerCase()) || i.labels?.some(l => l.toLowerCase().includes(query.toLowerCase())))
```

## Acceptance Criteria

- [ ] Typing "trail" in CMD+K shows only issues with "trail" in their title, ID, or labels.
- [ ] Navigation items and actions still use fuzzy matching (e.g. "bkl" matches "Backlog").
- [ ] Results are relevant and not flooded with false positives.
