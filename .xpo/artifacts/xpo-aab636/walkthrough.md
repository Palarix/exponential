# Walkthrough: Lighter, class-based dimming in the Dependencies view

## What was built and why

The Dependencies graph dimmed anything outside the focus or hover set to opacity 0.15 (edges) and 0.3 (nodes). On our near-black surfaces that made edge labels and issue ids nearly invisible. The values were also hard-coded inline on every element, plus a separate 0.6 for DONE rows in the table, so there was nowhere to tune them.

Dimming is now done with CSS classes backed by three variables, and it is lighter.

## How the pieces fit

1. **Variables** (`web/src/index.css`, a `:root` block just before BASE STYLES):
   - `--dep-dimmed-node-opacity: 0.55`
   - `--dep-dimmed-edge-opacity: 0.4`
   - `--dep-done-opacity: 0.7`

   They go in plain `:root` and not in `@theme`, because they aren't colors or Tailwind tokens.
2. **Classes** (DEPENDENCIES VIEW section at the end of `index.css`):
   - `.dep-done` uses the done opacity.
   - `.dep-node.dep-dimmed` and `.dep-edge.dep-dimmed` use the dimmed opacities.
   - `.dep-edge` has a 150ms opacity transition (`--duration-normal`).
3. **Class helpers** (`dep-graph-utils.ts`): `depNodeClass({ dimmed, done })` and `depEdgeClass(dimmed)` return the class strings. They're pure functions, so we can unit test them without component-testing setup, which the web app doesn't have.
4. **Components:**
   - `EdgePath`'s `<g>` takes `depEdgeClass(dimmed)`.
   - `GraphNodeCard` adds `depNodeClass(...)` to its className and drops `opacity` from `style`.
   - `IssueCell` in `Dependencies.tsx` adds `dep-done` to DONE titles.

## Non-obvious decisions

- **Dimmed beats done.** `.dep-node.dep-dimmed` (two classes) is more specific than `.dep-done` (one class), so a DONE node that is also dimmed shows the dimmed value. The two fades don't multiply. This matches the old ternary.
- **Node transitions stay in Tailwind.** Rules in `index.css` are unlayered, so they win over Tailwind's `@layer utilities`. A `.dep-node { transition: opacity … }` rule would replace `transition-all duration-150` and drop the border/box-shadow fade on hover. So the node keeps `transition-all`, which already covers opacity, and only the SVG edge group, which has no Tailwind transition, gets a CSS one.
- **Dark only.** The app has a single dark theme today. A future light theme only needs to override the three variables in its own selector.

## Acceptance criteria

- [x] No inline `opacity` on graph nodes/edges or table rows; dimming/done state comes from CSS classes. Evidence: a grep for `opacity` in `components/Dependencies/*.tsx` finds only the empty-state icon's `opacity-30`, which is out of scope.
- [x] Dim/done opacity values are defined once, as CSS variables in `index.css`
- [x] Dimmed nodes and edges, including their labels, are still readable (dark theme). Evidence: user review ("lgtm").
- [x] Hover/focus highlighting still clearly stands out. Evidence: user review.
- [x] Unit tests cover the class helpers (6 new cases in `dep-graph-utils.test.ts`), and `make test` passes
