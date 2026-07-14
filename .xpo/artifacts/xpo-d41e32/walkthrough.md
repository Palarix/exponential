# Walkthrough: CMD+K Search Not Matching on Issue Titles Correctly

## What changed

One line in `web/src/components/CommandPalette/CommandPalette.tsx`, line 99.

### Before

```typescript
issues.filter(i => fuzzyMatch(i.title, query) || i.id.includes(query.toLowerCase()) || i.labels?.some(l => fuzzyMatch(l, query)))
```

`fuzzyMatch` is a subsequence matcher — it checks whether each character of the query appears in order anywhere in the text. For short command labels ("Backlog", "Create new issue") this works well. For issue titles (long natural-language strings), it's far too loose: "trail" matches any title containing t...r...a...i...l spread across different words, producing 61 false positives from 300 issues.

### After

```typescript
issues.filter(i => { const q = query.toLowerCase(); return i.title.toLowerCase().includes(q) || i.id.includes(q) || i.labels?.some(l => l.toLowerCase().includes(q)); })
```

Simple case-insensitive substring matching. "trail" now matches exactly the 5 issues that contain "trail" in their title.

### What was kept

The `fuzzyMatch` function is still used for actions (line 65) and navigation items (line 86). These have short, predictable labels where subsequence matching is valuable (e.g. "bkl" matches "Backlog").
