## What

A `relates_to` link appears twice in the Dependencies table and the graph draws two edges for it.

## Why

Since xpo-9d6609 the API returns derived inverses. For a symmetric kind, A→B (stored) and B→A (derived) both reach `collectEdges` in `web/src/components/Dependencies/useDepGraph.ts`, whose dedup key is directional, so both survive. That function feeds the table, the ELK graph and `computeStats`.

Found during review: two more causes of "double arrows" between a pair of cards.

1. **Redundant relates_to next to a blocking link.** Some pairs have both a blocking link (e.g. A depends_on B) and a separate relates_to link. These are two real stored links, so the view draws two edges.
2. **False cycles.** `detectCycles` treats every edge as directed, including relates_to. So "A depends_on B" + "B relates_to A" counts as a cycle. Cycle edges are left out of the ELK layout and drawn as straight yellow dashed lines, and the cards get the ⚠ badge.

## How

- In `collectEdges`, treat symmetric kinds (`relates_to`, legacy `related`) as undirected: the dedup key uses the sorted id pair and a canonical kind (`related` → `relates_to` for keying only).
- Orientation: a derived symmetric row is flipped back to the stored direction (derived B→A ⇒ A→B), so the edge's direction is the stored one no matter which issue is iterated first. Derived rows are not dropped outright, since the stored side may not be in the loaded issue list.
- **Hide redundant relates_to (user decision):** if a pair of issues also has a non-symmetric link (blocked_by, depends_on, duplicated_by, …), `collectEdges` drops the relates_to edge for that pair. This applies to the graph and the table (both use `collectEdges`). Stored data is unchanged.
- **Cycle detection only looks at blocker kinds** (`blocked_by`, `depends_on`), the same set the critical path and stats use. Relates_to and duplicates can't form a dependency cycle.
- `computeStats` only counts `blocked_by`/`depends_on`, so totals were not inflated; no change there. Add a test confirming it.

## Acceptance criteria

- [ ] A single `relates_to` link appears once in the Dependencies table
- [ ] The graph draws one edge for it
- [ ] Unit test in `useDepGraph.test.ts` covers an API payload that has both the stored and the derived relates_to rows
- [ ] Asymmetric kinds (blocks/blocked_by, etc.) still dedupe correctly
- [ ] A pair with both a blocking link and a relates_to link shows only the blocking edge/row
- [ ] relates_to edges never make issues show up as a cycle (no yellow dashes or ⚠ from them)
