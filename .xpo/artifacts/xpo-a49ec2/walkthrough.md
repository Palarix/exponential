## Dependencies View: Interactive Dependency Graph

Replaced the flat grouped-by-kind dependency list with a two-level interface: a filterable
table for scanning dependencies, and a dagre-powered graph drill-down for exploring chains.

## Architecture

Three new/rewritten files in `web/src/components/Dependencies/`:

```
Dependencies.tsx  — orchestrator: table (default) ↔ graph drill-down
useDepGraph.ts    — data layer: edge collection, cycle detection, critical path, dagre layout
DepGraph.tsx      — SVG renderer: nodes via foreignObject, edge paths, pan/zoom
```

Plus routing changes in `App.tsx` and a navigation entry point in `PropertySidebar.tsx`.

## Data flow: useDepGraph.ts

The hook is the most complex piece. It processes the raw `issue.dependencies` array through
several stages:

### Edge collection (`collectEdges`)

Walks every issue's `dependencies` array and canonicalizes inverse pairs — `blocked_by`
becomes `blocks` with source/target swapped, `dependency_of` becomes `depends_on`, etc.
Deduplication uses a `sourceId->targetId:kind` key set. Uses suffix-match fallback for
`target_id` resolution (legacy IDs stored without prefix, per xpo-147414).

### Resolved detection (`isResolved`)

Relationship-aware: a `blocks` edge is resolved when the source (blocker) is DONE. A
`depends_on` edge is resolved when the target (dependency) is DONE. All other kinds require
both sides DONE. This drives both the "show completed" filter and the blocker stats.

### Connected component (`findConnectedComponent`)

BFS from the focus issue, following edges in both directions (undirected adjacency). Returns
the full set of transitively connected issue IDs. This scopes the graph to only the relevant
chain instead of the entire project.

### Cycle detection

DFS coloring (white/gray/black). Gray-to-gray back-edges mark cycle participants. These nodes
get excluded from dagre's input (it requires a DAG) but their edges are still rendered with
dashed stroke and warning markers.

### Critical path

Longest-path computation on the `blocks`/`depends_on` subgraph using topological sort. DONE
nodes have weight 0, others weight 1. The path with the highest total weight is the critical
chain — its edges render with thicker accent-colored strokes.

### Layout

dagre's Sugiyama algorithm (`rankdir: 'LR'`, 240×56 node dimensions) assigns x/y coordinates.
The hook returns positioned nodes, routed edge points, cycle/critical sets, and the bounding
box dimensions.

## Graph renderer: DepGraph.tsx

An SVG with a single `<g>` transform for pan/zoom:

- **Pan**: mouse drag on background (not on nodes) updates `translate(x, y)`
- **Zoom**: wheel events scale around the cursor position, clamped to [0.2, 2.0]
- **Fit-to-screen**: computes scale to fit dagre's bounding box within the container

Nodes use `<foreignObject>` wrapping a React component (`GraphNodeCard`). This lets us reuse
`StatusIcon` and Tailwind classes directly. The focus issue gets a thicker accent border.
Hovering a node dims all unrelated nodes and edges.

Edges use `<path>` with quadratic Bezier smoothing through dagre's routed points. Arrow markers
are defined per kind color, plus special markers for critical and cycle edges.

## Table view: Dependencies.tsx

The default view renders each dependency edge as a card row:

```
[StatusIcon] [hash-ID] [label badge] [title]  —kind→  [StatusIcon] [hash-ID] [label badge] [title]
```

The kind column is a fixed-width center divider with a colored line, label, and arrow.
IDs display just the hash portion (stripped `xpo-` prefix) for density.

Filters: kind dropdown (Blocks, Depends on, Relates to, Duplicates), centered search bar,
and "show completed" toggle. Clicking a row sets `depFocusId` in state, switching to the
graph drill-down.

## Routing: App.tsx

Extended the hash router with `#/dependencies/<issueId>`:

- `parseHash` returns `depFocusId` from the URL
- `setHash` writes it back
- `depFocusId` state threaded to Dependencies component as `focusIssueId`
- `handleViewChange` clears it (sidebar nav resets to table)
- `hashchange` handler reads it (external navigation, e.g. from PropertySidebar)

## Entry from issue detail: PropertySidebar.tsx

Each dependency row in the Relations card is now a clickable div that navigates to
`#/dependencies/<currentIssueId>`. This replaces the previous setup where only the
issue title was a link (to the issue detail) and a tiny graph icon was the graph entry.
The delete button uses `stopPropagation` to avoid accidentally navigating when removing
a dependency.

## Dependencies added

- `@dagrejs/dagre` (3.1.1) — Sugiyama layered DAG layout, ~5kb gzip
- `@types/dagre` (0.7.54) — TypeScript definitions
