# Spec: Extract `IconButton` component for toolbar actions

## What

Extract a shared `IconButton` component into `web/src/components/ui/IconButton.tsx` that replaces the duplicated `w-7 h-7` bordered toolbar button pattern. Export it from the UI barrel (`index.ts`).

## Why

Six instances across five views duplicate the same markup with minor variations (text color, active indicator, tooltip). Consolidating into a component ensures visual consistency and reduces future drift.

## Current State

Six instances of the pattern:

| # | File | Line | Tooltip | Active dot | Text color variant |
|---|------|------|---------|-----------|--------------------|
| 1 | `Backlog.tsx` | ~1507 | "Filter" | `hasActiveFilters(filters)` | muted → secondary |
| 2 | `Backlog.tsx` | ~1541 | "View options" | none | muted → secondary |
| 3 | `Board.tsx` | ~466 | none | none | secondary → primary |
| 4 | `Inbox.tsx` | ~198 | "Filter" | `hasActiveFilters(filters)` | muted → secondary |
| 5 | `MyIssues.tsx` | ~139 | "Filter" | `hasActiveFilters(filters)` | muted → secondary |
| 6 | `Timeline.tsx` | ~365 | "Filter" | `!allEnabled` | muted → secondary |

All share:
- `flex items-center justify-center w-7 h-7 rounded-[var(--radius-md)] bg-[var(--color-surface-1)] border border-[var(--color-border-default)] transition-colors`
- Active dot: `absolute -top-0.5 -right-0.5 w-2 h-2 rounded-full bg-[var(--color-accent-primary)]`

## API

```tsx
interface IconButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  icon: ReactNode;
  active?: boolean;
  tooltip?: string;
}
```

- `icon` — the SVG or Lucide icon element to render inside
- `active` — when true, renders the indicator dot; also adds `relative` to the button
- `tooltip` — when provided, wraps the button in `<Tooltip content={tooltip}>`
- All other native button props (`onClick`, `ref`, etc.) pass through
- Uses `forwardRef` so callers can attach refs for popover anchoring

## Flow

1. Create `web/src/components/ui/IconButton.tsx` with `forwardRef`, Tooltip integration, active dot, and `cn()` class merging.
2. Export from `web/src/components/ui/index.ts`.
3. Write tests in `web/src/components/ui/IconButton.test.ts`.
4. Migrate all 6 instances — each site replaces ~10-15 lines of button markup with a single `<IconButton>` call.
5. Run `make test` to verify no regressions.

## Decisions

- **Default text color is `text-secondary hover:text-primary`** (Board's variant) — for a small icon-only button without a text label, the stronger contrast makes the hover state noticeably visible. The 5 instances currently using `text-muted hover:text-secondary` will be upgraded to the higher-contrast default. This is a deliberate visual improvement, not preserving drift.
- **Tooltip handled inside the component** — avoids wrapping boilerplate at every call site. When `tooltip` is falsy, no Tooltip is rendered (matches existing `Tooltip` behavior which already short-circuits on empty content).
- **`relative` class applied conditionally when `active` is true** — only needed for the absolute-positioned dot. This matches existing behavior where buttons without an active dot don't have `relative`.
- **6 instances, not 5** — Timeline.tsx has the same pattern; including it in scope.

## Acceptance Criteria

- [ ] `IconButton` component created at `web/src/components/ui/IconButton.tsx`
- [ ] Exported from UI barrel
- [ ] All 6 instances migrated to `IconButton`
- [ ] Active indicator dot renders when `active` is true
- [ ] Tooltip shows on hover when `tooltip` is provided
- [ ] Ref forwarding works (popover anchoring unchanged)
- [ ] Tests cover: default render, active dot, tooltip, className override, ref forwarding
- [ ] `make test` passes
- [ ] No visual regression
