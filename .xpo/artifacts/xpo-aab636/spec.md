# Spec: Lighter, class-based dimming in the Dependencies view

## What

Move the Dependencies view's dim/done opacity out of inline styles into CSS classes backed by CSS custom properties, and make the dimming lighter so dimmed nodes, edges and labels stay readable.

## Why

`DepGraph.tsx` hard-codes `opacity` on every edge (`0.15`) and node (`0.3` dimmed, `0.55` done), and `Dependencies.tsx` hard-codes `0.6` on done row titles. At 0.15/0.3 against the near-black surfaces, edge labels and issue ids practically disappear. The values are also scattered, which makes them impossible to tune in one place.

## How

1. **Tokens** in `web/src/index.css`, in a new `:root` block next to the theme tokens (these are not colors, so they go in plain `:root` instead of `@theme`):
   - `--dep-dimmed-node-opacity: 0.55`
   - `--dep-dimmed-edge-opacity: 0.4`
   - `--dep-done-opacity: 0.7`
2. **Classes** in `index.css`:
   - `.dep-edge`: `transition: opacity var(--duration-normal)` (150ms)
   - `.dep-done` (node and table title): `opacity: var(--dep-done-opacity)`
   - `.dep-node.dep-dimmed`: `opacity: var(--dep-dimmed-node-opacity)` (more specific, so dimmed wins over done)
   - `.dep-edge.dep-dimmed`: `opacity: var(--dep-dimmed-edge-opacity)`
3. **Pure class helpers** in `dep-graph-utils.ts`: `depNodeClass({ dimmed, done })` and `depEdgeClass(dimmed)`, unit tested first (TDD).
4. **Components**: the `EdgePath` `<g>` gets `className={depEdgeClass(dimmed)}` (no `opacity` attribute or inline transition). `GraphNodeCard` drops `opacity` from `style` and prepends `depNodeClass(...)` to its className. It keeps Tailwind's `transition-all duration-150`, which also animates opacity. The `IssueCell` title drops its inline `opacity` and adds `dep-done` when DONE.

## Decisions

- **Themes:** the app only has the dark theme (`color-scheme: dark`, single `@theme` block), and the user confirmed tuning for dark. A future light theme can override the three variables in its own selector.
- **Dimmed + done:** dimmed wins (no multiplication), matching current behavior.
- **Node transition stays in Tailwind:** `index.css` rules are unlayered and beat Tailwind's `@layer utilities`, so a `.dep-node { transition: opacity }` rule would override `transition-all` and drop the border/box-shadow fade on hover. Only the SVG edge `<g>` (which has no Tailwind transition) gets the CSS transition.
- **Focus contrast:** non-dimmed elements stay at full opacity. Highlighted edges keep their thicker stroke and focus/hover nodes keep their accent border, so 1.0 against 0.55/0.4 still reads clearly.

## Acceptance criteria

- [ ] No inline `opacity` on graph nodes/edges or table rows; dimming/done state comes from CSS classes
- [ ] Dim/done opacity values are defined once (CSS variables in `index.css`)
- [ ] Dimmed nodes and edges, including their labels, are still readable (dark theme; only theme today)
- [ ] Hover/focus highlighting still clearly stands out
- [ ] Unit tests cover the class helpers; `make test` passes
