# CountBadge — Extract shared issue-count component

## What

Extract a `CountBadge` UI primitive from the duplicated `text-xs tabular-nums` issue-count spans found in 5 views.

## Why

Five views repeat identical markup with minor syntax variation (ternary, template literal, inline concat). A single component eliminates the duplication and makes the pattern easy to reuse in future views.

## Where (duplication sites)

| # | File | Line | Count expression |
|---|------|------|------------------|
| 1 | `Board/Board.tsx` | 500 | `issues.filter(…).length` via IIFE + template literal |
| 2 | `MyIssues/MyIssues.tsx` | 119 | `filtered.length` + inline concat |
| 3 | `Dashboard/Dashboard.tsx` | 217 | `issues.length` + inline concat |
| 4 | `Backlog/Backlog.tsx` | 1695 | `filteredIssues.length` + multi-line JSX |
| 5 | `Labels/Labels.tsx` | 262 | `label.count` + ternary with full word |

Note: the issue description lists 3 instances (Board, My Issues, Dashboard). Investigation found 2 more (Backlog, Labels). All 5 will be migrated.

## How

### 1. Create `web/src/components/ui/CountBadge.tsx`

```tsx
interface CountBadgeProps {
  count: number;
  label?: string;   // singular form — default "issue"
  short?: boolean;   // true → show count only, omit label
  rounded?: boolean; // true → pill/badge style
}

export default function CountBadge({
  count,
  label = "issue",
  short = false,
  rounded = false,
}: CountBadgeProps) {
  return (
    <span
      className={`text-xs tabular-nums ${
        rounded
          ? "inline-flex items-center justify-center min-w-[1.25rem] px-1.5 py-0.5 rounded-full bg-[var(--color-hover-surface)] text-[var(--color-text-muted)]"
          : "text-[var(--color-text-muted)]"
      }`}
    >
      {short ? count : `${count} ${label}${count !== 1 ? "s" : ""}`}
    </span>
  );
}
```

Design notes:
- `label` is the **singular** form; the component appends "s" for plural. Matches all 5 current usages.
- `short` renders just the number — useful for column headers and compact contexts (BoardColumn, BacklogGroupHeader, etc. already show bare counts with `tabular-nums`).
- `rounded` renders a pill with subtle background — useful for badge/tag-style counts. Uses `--color-hover-surface` for the background to stay theme-aware and visually light.
- `short` and `rounded` can combine (pill with just a number).
- All existing migration sites use the defaults (`short=false`, `rounded=false`) so the rendered HTML is identical.

### 2. Export from barrel

Add to `web/src/components/ui/index.ts`:
```ts
export { default as CountBadge } from './CountBadge';
```

### 3. Migrate each site

Replace each `<span>` with `<CountBadge count={…} />`. All 5 use the default label "issue" and the default (non-short, non-rounded) style, so no extra props needed.

**Board.tsx** — the IIFE computing the filtered count stays; only the span wrapper changes:
```tsx
<CountBadge count={issues.filter(i => !hiddenColumns.has(i.status)).length} />
```

**MyIssues.tsx, Dashboard.tsx, Backlog.tsx, Labels.tsx** — straightforward replacement passing the existing count variable.

## Acceptance criteria

- [ ] `CountBadge` component created in `web/src/components/ui/`
- [ ] Props: `count` (required), `label` (optional, default "issue"), `short` (optional), `rounded` (optional)
- [ ] All 5 instances migrated (Board, MyIssues, Dashboard, Backlog, Labels)
- [ ] Singular/plural handled correctly (1 issue, 0 issues, 2 issues)
- [ ] `short` mode renders count only, no label text
- [ ] `rounded` mode renders pill/badge style with background
- [ ] No visual regression on migrated sites — identical rendered HTML
- [ ] `make test` passes
