## What was built

The Dependencies view (table + graph) showed every `relates_to` link twice. Reviewing the fix surfaced two related causes of "double arrows". All three are fixed in `web/src/components/Dependencies/useDepGraph.ts`, which feeds the table rows, the ELK graph edges and `computeStats`.

## How it fits together

### 1. Symmetric links deduplicate as undirected pairs

Since xpo-9d6609 the API returns derived inverses: if A stores `relates_to B`, B's `dependencies` also carries a derived `B relates_to A` row. `collectEdges` used a directional dedup key (`src->tgt:kind`). That works for asymmetric kinds, because `INVERSE_KINDS` canonicalises `blocks` into `blocked_by` with a swap, so both sides produce the same key. But `relates_to` is its own inverse, so the two rows got different keys.

A new `SYMMETRIC_KINDS` map (`relates_to`, legacy `related` → `relates_to`) switches those kinds to a key built from the sorted id pair. A derived symmetric row is flipped back to the stored direction (`isInverse = !!dep.derived`), so the surviving edge always points the way the link was created, whichever issue is iterated first. Derived rows are deliberately not dropped outright: the stored side might not be in the loaded issue list.

### 2. Redundant relates_to is hidden (user decision during review)

Some pairs genuinely hold two stored links, e.g. xpo-09f337 depends_on xpo-e9a2eb plus a relates_to between them. After collecting, `collectEdges` drops any symmetric edge whose unordered pair also has a directed edge. The stronger link already implies "related". Stored data is untouched. xpo-c93b11 will prevent new multi-link pairs from being created in the first place.

### 3. Cycle detection only uses blocker edges

`detectCycles` built its adjacency from every edge, so "A depends_on B" + "B relates_to A" registered as a cycle. Cycle edges are left out of the ELK layout and drawn as straight yellow dashed lines, and the cards get ⚠. That is what made the pairs so loud in the screenshot. It now only follows `blocked_by`/`depends_on` edges, matching `computeStats` and `computeCriticalPath`. All three now share a module-level `BLOCKER_KINDS` constant instead of local copies. `detectCycles` is exported for testing.

## Non-obvious points

- `computeStats` was never inflated: it only counts blocker kinds, and a test pins that.
- The graph's `edgeById` lookup in `useDepGraph` still keys ELK edges by `src->tgt` without kind. Two edges between the same pair would collide. After this change that only happens for two different directed kinds on one pair, which xpo-c93b11 will forbid.

## Acceptance criteria

- [x] A single `relates_to` link appears once in the Dependencies table: `collectEdges` returns one edge for stored+derived rows (test "emits one relates_to edge for the stored row and its derived inverse"). The live board data for xpo-f01ce3's component yields one edge per pair.
- [x] The graph draws one edge for it: the graph is built from the same `collectEdges` output. Confirmed by the user in the running board.
- [x] Unit test in `useDepGraph.test.ts` covers an API payload with both the stored and the derived relates_to rows: 3 tests (either iteration order, legacy `related` + `relates_to`).
- [x] Asymmetric kinds still dedupe correctly: the existing "emits one edge when the API returns a link and its derived inverse" and "deduplicates edges" tests pass unchanged.
- [x] A pair with both a blocking link and a relates_to link shows only the blocking edge/row: tests "hides a relates_to link when the same pair has a blocking link" and "…stored in the same direction as a blocks link". Live data: xpo-09f337/xpo-e9a2eb → depends_on only, xpo-e669d4/xpo-e60c26 → blocked_by only.
- [x] relates_to edges never make issues show up as a cycle: `detectCycles` tests "ignores relates_to edges when looking for cycles" and "does not treat a ring of relates_to edges as a cycle"; a true blocker cycle is still flagged.

`make test` passes (lint + 562 frontend tests + Go suite).
