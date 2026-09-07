## What

Replace the flat dependency table in the Dependencies view with a two-level interface:
a **filterable table** as the default view for scanning all dependencies, and a **graph
drill-down** for exploring the topology around a specific issue's dependency chain.

## Why

The primary question the Dependencies view answers is: "Which issues have dependencies,
and what do those dependency chains look like?" A scannable table answers the first part;
a focused graph answers the second. Showing the entire project graph by default overwhelms
— the graph earns its value when scoped to a specific chain.

## Acceptance Criteria

### Table (default view)
- [x] Each row is a card: `StatusIcon | ID (hash only) | primary label | title` → `kind arrow` → `StatusIcon | ID | primary label | title`
- [x] Filterable by dependency kind (Blocks, Depends on, Relates to, Duplicates)
- [x] Searchable by text (matches issue ID, title) — large centered search bar
- [x] "Show completed" toggle hides resolved dependencies (relationship-aware: blocks resolved when source DONE, depends_on resolved when target DONE)
- [x] Header shows blocker summary: "N of M blockers resolved"
- [x] Clicking a row opens the graph drill-down scoped to that issue's connected subgraph

### Graph drill-down
- [x] Shows the connected subgraph for the selected issue (all transitively connected issues via BFS)
- [x] Back button / Escape returns to the table
- [x] Title bar shows which issue's chain is being viewed
- [x] Nodes show: `StatusIcon`, truncated title (≤40 chars), issue ID
- [x] Nodes color-coded by status using existing CSS variables (`--color-status-*`)
- [x] Focus issue node has accent border to stand out
- [x] DONE nodes appear muted (reduced opacity, strikethrough title)
- [x] Edges are directed arrows colored by relationship kind
- [x] Clicking a node navigates to the issue detail (`onIssueClick`)
- [x] Hovering a node highlights its direct edges, dims unrelated nodes
- [x] Circular dependencies render with warning badge (yellow triangle) on involved nodes
- [x] Critical path highlighted: longest unresolved blocker chain gets thicker accent-colored edges
- [x] Pan (mouse drag) and zoom (scroll wheel)
- [x] Fit-to-screen button
- [x] "Show completed" toggle carries through from the table

### Entry from issue detail
- [x] Clicking a dependency row in the property sidebar navigates to Dependencies view
  with that issue's chain pre-selected, landing directly in the graph drill-down

## Decisions

- **Table as default, not graph**: the graph overwhelms when showing everything. The table
  is scannable and answers "what dependencies exist." The graph answers "what does this
  chain look like" — a drill-down, not an overview.
- **No epic filter**: filtering by epic is the wrong axis for dependencies. Kind and text
  search are the useful filters.
- **No child/parent in kind filter**: parent-child relationships use `parent_id`, not
  dependency edges. The actual kinds in the data are: blocks, depends_on, relates_to, duplicates.
- **Relationship-aware resolved logic**: `blocks` is resolved when the source (blocker) is DONE.
  `depends_on` is resolved when the target (dependency) is DONE. Other kinds: both DONE.
- **Connected subgraph, not single-hop**: when drilling into a chain, show the full
  transitively connected component — not just immediate neighbors.
- **dagre for layout**: deterministic Sugiyama layout, ~5kb gzip.
- **foreignObject for nodes**: reuses existing React components inside SVG.
- **Whole-row click in property sidebar**: navigates to graph drill-down. Delete button
  uses stopPropagation for safety.
