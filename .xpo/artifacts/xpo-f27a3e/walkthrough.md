# CountBadge — Implementation Walkthrough

## What was built

A `CountBadge` UI primitive that replaces 5 duplicated issue-count spans across the app. The component renders a `text-xs tabular-nums` span with automatic singular/plural handling, plus two optional display modes (`short` for count-only, `rounded` for pill style).

## Structure

Three new files in `web/src/components/ui/`:

- **`count-badge-utils.ts`** — two pure functions:
  - `formatCount(count, label, short)` — returns the display string (`"3 issues"`, `"1 issue"`, or `"3"` when short)
  - `countBadgeClass(rounded)` — returns the className string, adding pill styles when rounded
- **`CountBadge.tsx`** — the React component, composing the two utilities
- **`CountBadge.test.ts`** — 10 unit tests covering pluralisation, custom labels, short mode, and rounded class generation

Splitting logic into utilities follows the existing pattern (see `IconButton` / `icon-button-utils.ts`) and keeps tests pure without needing DOM rendering.

## Migration

All 5 sites used identical styling (`text-xs text-[var(--color-text-muted)] tabular-nums`) but each implemented singular/plural differently:

| Site | Before | After |
|------|--------|-------|
| `Board/Board.tsx` | IIFE + template literal | `<CountBadge count={issues.filter(…).length} />` |
| `MyIssues/MyIssues.tsx` | Inline string concat | `<CountBadge count={filtered.length} />` |
| `Dashboard/Dashboard.tsx` | Inline string concat | `<CountBadge count={issues.length} />` |
| `Backlog/Backlog.tsx` | Multi-line JSX fragments | `<CountBadge count={filteredIssues.length} />` |
| `Labels/Labels.tsx` | Ternary with full words | `<CountBadge count={label.count} />` |

The issue description listed 3 instances; investigation found Backlog and Labels as well. All 5 use default props so the rendered HTML is byte-identical to the original — no visual regression.

## Key decisions

- **`label` takes singular form, pluralised by appending "s"**: all current usages follow this pattern. If an irregular plural is needed later, the API can be extended without breaking existing call sites.
- **`short` and `rounded` are independent booleans**: they can combine (pill with just a number). Neither is used by the 5 migrated sites — they're available for future views like column headers or compact badges.
- **`--color-hover-surface` for rounded background**: stays theme-aware and visually light, consistent with other subtle badge backgrounds in the app.

## Acceptance criteria

- [x] `CountBadge` component created in `web/src/components/ui/`
- [x] Props: `count` (required), `label` (optional, default "issue"), `short` (optional), `rounded` (optional)
- [x] All 5 instances migrated (Board, MyIssues, Dashboard, Backlog, Labels)
- [x] Singular/plural handled correctly (1 issue, 0 issues, 2 issues) — verified by unit tests
- [x] `short` mode renders count only, no label text — verified by unit tests
- [x] `rounded` mode renders pill/badge style with background — verified by unit tests
- [x] No visual regression on migrated sites — identical rendered HTML
- [x] `make test` passes (314 frontend tests, 13 Go packages)