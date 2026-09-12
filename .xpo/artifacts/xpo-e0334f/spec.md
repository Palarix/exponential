# Text — Extract typography primitive

## What

Extract a `Text` UI primitive that captures the repeated `text-{size} text-[var(--color-...)]` patterns. `Heading` stays separate for semantic heading use cases.

## Why

Hundreds of sites repeat className combinations of size + color + weight + extras. A `Text` component eliminates the duplication and makes the typography system explicit.

## Scope

The raw grep surface is ~660 hits, but many are in button/input/interactive styling. This story targets **pure text display** in `<span>` and `<p>` elements — not interactive elements, not elements with complex conditional styling.

Priority order:
1. Create the component + utilities + tests
2. Swap `Heading` → `Text` for ActivityTimeline and MergeView author names
3. Migrate `text-[var(--color-text-muted)]` instances (most common — ~150 eligible)
4. Migrate `text-[var(--color-text-secondary)]` instances (~40 eligible)
5. Migrate standalone `text-[var(--color-text-primary)]` instances not covered by `Heading` (~30 eligible)
6. Migrate semantic color instances (success/error/warning/accent — ~20 eligible)
7. Migrate `font-mono` and `tabular-nums` combinations alongside color migrations

## API

```tsx
interface TextProps {
  children: ReactNode;
  as?: 'span' | 'p' | 'label';       // default 'span'
  color?: 'primary' | 'secondary' | 'muted' | 'success' | 'warning' | 'error' | 'accent';
                                       // default 'muted'
  size?: 'xs' | 'regular' | 'base' | 'lg';  // default 'regular' (= text-sm)
  weight?: 'regular' | 'medium' | 'semibold' | 'bold';  // default 'regular'
  mono?: boolean;
  tabular?: boolean;
  truncate?: boolean;
}
```

### Color map

| Prop value | CSS variable |
|-----------|-------------|
| `primary` | `--color-text-primary` |
| `secondary` | `--color-text-secondary` |
| `muted` | `--color-text-muted` |
| `success` | `--color-success` |
| `warning` | `--color-warning` |
| `error` | `--color-error` |
| `accent` | `--color-accent-primary` |

### Size map

| Prop value | Tailwind class |
|-----------|---------------|
| `xs` | `text-xs` |
| `regular` | `text-sm` |
| `base` | `text-base` |
| `lg` | `text-lg` |

### What NOT to migrate

- Buttons, links, inputs — interactive elements with hover/focus states
- Elements with conditional color (e.g. `style={{ color: condition ? X : Y }}`)
- Elements already covered by `Heading` or `CountBadge`
- Elements where `text-sm`/`text-xs` appears alongside layout-specific classes that make extraction awkward

## Acceptance criteria

- [ ] `Text` component created in `web/src/components/ui/`
- [ ] Props: `color`, `size`, `weight`, `mono`, `tabular`, `truncate`, `as`
- [ ] Default: `<span className="text-sm text-[var(--color-text-muted)]">`
- [ ] Swap `Heading` → `Text` for author names (ActivityTimeline, MergeView)
- [ ] Migrate muted text instances
- [ ] Migrate secondary text instances
- [ ] Migrate primary text instances (non-Heading)
- [ ] Migrate semantic color instances
- [ ] `Heading` remains separate
- [ ] `make test` passes
