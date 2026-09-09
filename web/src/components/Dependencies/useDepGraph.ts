import { useState, useEffect } from 'react';
import ELK, { type ElkNode, type ElkExtendedEdge } from 'elkjs/lib/elk.bundled.js';
import type { Issue } from '../../api/types';

export interface GraphNode {
  issue: Issue;
  x: number;
  y: number;
}

export interface GraphEdge {
  sourceId: string;
  targetId: string;
  kind: string;
  points: { x: number; y: number }[];
  labelPos?: { x: number; y: number; width: number; height: number };
  isCycle: boolean;
  isCritical: boolean;
}

export interface DepGraph {
  nodes: GraphNode[];
  edges: GraphEdge[];
  cycles: Set<string>;
  criticalPath: Set<string>;
  width: number;
  height: number;
}

export interface DepStats {
  resolved: number;
  total: number;
}

export interface RawEdge {
  sourceId: string;
  targetId: string;
  kind: string;
  originalKind: string;
}

const NODE_WIDTH = 240;
const NODE_HEIGHT = 56;

const INVERSE_KINDS: Record<string, string> = {
  blocks: 'blocked_by',
  dependency_of: 'depends_on',
  duplicates: 'duplicated_by',
};

const elk = new ELK();

const KIND_DISPLAY: Record<string, string> = {
  blocked_by: 'blocked by',
  depends_on: 'depends on',
  relates_to: 'relates to',
  related: 'relates to',
  duplicated_by: 'duplicated by',
};

export function resolveIssue(issues: Issue[], targetId: string): Issue | undefined {
  return issues.find(i => i.id === targetId)
    ?? issues.find(i => i.id.endsWith(targetId));
}

export function collectEdges(issues: Issue[]): RawEdge[] {
  const seen = new Set<string>();
  const edges: RawEdge[] = [];

  for (const issue of issues) {
    if (!issue.dependencies) continue;
    for (const dep of issue.dependencies) {
      const target = resolveIssue(issues, dep.target_id);
      if (!target) continue;

      const canonKind = INVERSE_KINDS[dep.kind] ?? dep.kind;
      const isInverse = canonKind !== dep.kind;
      const sourceId = isInverse ? target.id : issue.id;
      const targetId = isInverse ? issue.id : target.id;

      const key = `${sourceId}->${targetId}:${canonKind}`;
      if (seen.has(key)) continue;
      seen.add(key);

      edges.push({ sourceId, targetId, kind: canonKind, originalKind: dep.kind });
    }
  }

  return edges;
}

export function isResolved(e: RawEdge, issueMap: Map<string, Issue>): boolean {
  const t = issueMap.get(e.targetId);
  switch (e.kind) {
    case 'blocked_by':
    case 'depends_on':
      return t?.status === 'DONE';
    default: {
      const s = issueMap.get(e.sourceId);
      return s?.status === 'DONE' && t?.status === 'DONE';
    }
  }
}

export function computeStats(issues: Issue[]): DepStats {
  const edges = collectEdges(issues);
  const blockerKinds = new Set(['blocked_by', 'depends_on']);
  let resolved = 0;
  let total = 0;
  const issueMap = new Map(issues.map(i => [i.id, i]));
  for (const e of edges) {
    if (!blockerKinds.has(e.kind)) continue;
    total++;
    if (isResolved(e, issueMap)) resolved++;
  }
  return { resolved, total };
}

function findConnectedComponent(focusId: string, edges: RawEdge[]): Set<string> {
  const adj = new Map<string, Set<string>>();
  for (const e of edges) {
    if (!adj.has(e.sourceId)) adj.set(e.sourceId, new Set());
    if (!adj.has(e.targetId)) adj.set(e.targetId, new Set());
    adj.get(e.sourceId)!.add(e.targetId);
    adj.get(e.targetId)!.add(e.sourceId);
  }

  const visited = new Set<string>();
  const queue = [focusId];
  while (queue.length > 0) {
    const id = queue.shift()!;
    if (visited.has(id)) continue;
    visited.add(id);
    for (const neighbor of adj.get(id) ?? []) {
      if (!visited.has(neighbor)) queue.push(neighbor);
    }
  }
  return visited;
}

function detectCycles(edges: RawEdge[], nodeIds: Set<string>): Set<string> {
  const adj = new Map<string, string[]>();
  for (const id of nodeIds) adj.set(id, []);
  for (const e of edges) adj.get(e.sourceId)?.push(e.targetId);

  const cycleNodes = new Set<string>();
  const WHITE = 0, GRAY = 1, BLACK = 2;
  const color = new Map<string, number>();
  for (const id of nodeIds) color.set(id, WHITE);

  const ancestors: string[] = [];

  function dfs(u: string) {
    color.set(u, GRAY);
    ancestors.push(u);
    for (const v of adj.get(u) ?? []) {
      if (color.get(v) === GRAY) {
        const cycleStart = ancestors.indexOf(v);
        for (let i = cycleStart; i < ancestors.length; i++) {
          cycleNodes.add(ancestors[i]);
        }
      } else if (color.get(v) === WHITE) {
        dfs(v);
      }
    }
    ancestors.pop();
    color.set(u, BLACK);
  }

  for (const id of nodeIds) {
    if (color.get(id) === WHITE) dfs(id);
  }

  return cycleNodes;
}

function edgeKey(sourceId: string, targetId: string): string {
  return `${sourceId}->${targetId}`;
}

function computeCriticalPath(edges: RawEdge[], issueMap: Map<string, Issue>): Set<string> {
  const blockerKinds = new Set(['blocked_by', 'depends_on']);
  const blockerEdges = edges.filter(e => blockerKinds.has(e.kind));

  const adj = new Map<string, { targetId: string; key: string }[]>();
  const inDegree = new Map<string, number>();
  const nodeIds = new Set<string>();

  for (const e of blockerEdges) {
    nodeIds.add(e.sourceId);
    nodeIds.add(e.targetId);
    if (!adj.has(e.sourceId)) adj.set(e.sourceId, []);
    adj.get(e.sourceId)!.push({ targetId: e.targetId, key: edgeKey(e.sourceId, e.targetId) });
    inDegree.set(e.targetId, (inDegree.get(e.targetId) ?? 0) + 1);
  }

  for (const id of nodeIds) {
    if (!inDegree.has(id)) inDegree.set(id, 0);
  }

  const dist = new Map<string, number>();
  const prev = new Map<string, { nodeId: string; edgeKey: string } | null>();
  for (const id of nodeIds) {
    const issue = issueMap.get(id);
    const weight = issue && issue.status === 'DONE' ? 0 : 1;
    dist.set(id, weight);
    prev.set(id, null);
  }

  const queue = [...nodeIds].filter(id => inDegree.get(id) === 0);
  while (queue.length > 0) {
    const u = queue.shift()!;
    for (const { targetId: v, key } of adj.get(u) ?? []) {
      const issue = issueMap.get(v);
      const weight = issue && issue.status === 'DONE' ? 0 : 1;
      const newDist = dist.get(u)! + weight;
      if (newDist > dist.get(v)!) {
        dist.set(v, newDist);
        prev.set(v, { nodeId: u, edgeKey: key });
      }
      inDegree.set(v, inDegree.get(v)! - 1);
      if (inDegree.get(v) === 0) queue.push(v);
    }
  }

  let maxDist = 0;
  let endNode = '';
  for (const [id, d] of dist) {
    if (d > maxDist) { maxDist = d; endNode = id; }
  }

  const criticalEdges = new Set<string>();
  if (maxDist <= 1) return criticalEdges;

  let current: string | null = endNode;
  while (current) {
    const p = prev.get(current);
    if (!p) break;
    criticalEdges.add(p.edgeKey);
    current = p.nodeId;
  }

  return criticalEdges;
}

const EMPTY_GRAPH: DepGraph = { nodes: [], edges: [], cycles: new Set(), criticalPath: new Set(), width: 0, height: 0 };

export function useDepGraph(issues: Issue[], focusIssueId: string, showCompleted: boolean): DepGraph {
  const [graph, setGraph] = useState<DepGraph>(EMPTY_GRAPH);

  useEffect(() => {
    const allEdges = collectEdges(issues);
    const issueMap = new Map(issues.map(i => [i.id, i]));

    if (!issueMap.has(focusIssueId)) {
      Promise.resolve().then(() => setGraph(EMPTY_GRAPH));
      return;
    }

    const componentIds = findConnectedComponent(focusIssueId, allEdges);
    const componentEdges = allEdges.filter(e => componentIds.has(e.sourceId) && componentIds.has(e.targetId));

    const TERMINAL = new Set(['DONE', 'CANCELED', 'DUPLICATE']);
    const filteredEdges = showCompleted
      ? componentEdges
      : componentEdges.filter(e => {
          const s = issueMap.get(e.sourceId);
          const t = issueMap.get(e.targetId);
          return !(s && TERMINAL.has(s.status)) && !(t && TERMINAL.has(t.status));
        });

    const nodeIds = new Set<string>();
    for (const e of filteredEdges) {
      nodeIds.add(e.sourceId);
      nodeIds.add(e.targetId);
    }

    if (!nodeIds.has(focusIssueId) && issueMap.has(focusIssueId)) {
      nodeIds.add(focusIssueId);
    }

    if (nodeIds.size === 0) {
      Promise.resolve().then(() => setGraph(EMPTY_GRAPH));
      return;
    }

    const cycleNodes = detectCycles(filteredEdges, nodeIds);
    const criticalEdges = computeCriticalPath(filteredEdges, issueMap);

    const elkGraph: ElkNode = {
      id: 'root',
      layoutOptions: {
        'elk.algorithm': 'layered',
        'elk.direction': 'RIGHT',
        'elk.spacing.nodeNode': '40',
        'elk.layered.spacing.nodeNodeBetweenLayers': '80',
        'elk.spacing.edgeNode': '20',
        'elk.spacing.edgeEdge': '15',
        'elk.edgeRouting': 'ORTHOGONAL',
        'elk.layered.mergeEdges': 'false',
        'elk.padding': '[top=40,left=40,bottom=40,right=40]',
      },
      children: [...nodeIds].map(id => ({
        id,
        width: NODE_WIDTH,
        height: NODE_HEIGHT,
      })),
      edges: filteredEdges
        .filter(e => !(cycleNodes.has(e.sourceId) && cycleNodes.has(e.targetId)))
        .map((e, i) => ({
          id: `e${i}`,
          sources: [e.sourceId],
          targets: [e.targetId],
          labels: [{
            text: KIND_DISPLAY[e.kind] ?? e.kind.replace(/_/g, ' '),
            width: (KIND_DISPLAY[e.kind] ?? e.kind).length * 6,
            height: 12,
            layoutOptions: { 'elk.edgeLabels.placement': 'CENTER' },
          }],
        } as ElkExtendedEdge)),
    };

    let cancelled = false;
    elk.layout(elkGraph).then(result => {
      if (cancelled) return;

      const nodeMap = new Map<string, { x: number; y: number }>();
      for (const child of result.children ?? []) {
        nodeMap.set(child.id, { x: (child.x ?? 0) + NODE_WIDTH / 2, y: (child.y ?? 0) + NODE_HEIGHT / 2 });
      }

      const nodes: GraphNode[] = [];
      for (const [id, pos] of nodeMap) {
        const issue = issueMap.get(id);
        if (issue) nodes.push({ issue, x: pos.x, y: pos.y });
      }

      const edgeById = new Map<string, ElkExtendedEdge>();
      for (const e of result.edges ?? []) {
        edgeById.set(`${(e as ElkExtendedEdge).sources[0]}->${(e as ElkExtendedEdge).targets[0]}`, e as ElkExtendedEdge);
      }

      const edges: GraphEdge[] = filteredEdges.map(e => {
        const elkEdge = edgeById.get(`${e.sourceId}->${e.targetId}`);
        const section = elkEdge?.sections?.[0];
        const points: { x: number; y: number }[] = [];
        if (section) {
          points.push(section.startPoint);
          if (section.bendPoints) points.push(...section.bendPoints);
          points.push(section.endPoint);
        } else {
          const sp = nodeMap.get(e.sourceId) ?? { x: 0, y: 0 };
          const tp = nodeMap.get(e.targetId) ?? { x: 0, y: 0 };
          points.push({ x: sp.x + NODE_WIDTH / 2, y: sp.y }, { x: tp.x - NODE_WIDTH / 2, y: tp.y });
        }

        const elkLabel = elkEdge?.labels?.[0];
        const labelPos = elkLabel && elkLabel.x != null && elkLabel.y != null
          ? { x: elkLabel.x, y: elkLabel.y, width: elkLabel.width ?? 0, height: elkLabel.height ?? 0 }
          : undefined;

        const isCycle = cycleNodes.has(e.sourceId) && cycleNodes.has(e.targetId);
        const isCritical = criticalEdges.has(edgeKey(e.sourceId, e.targetId));
        return { sourceId: e.sourceId, targetId: e.targetId, kind: e.kind, points, labelPos, isCycle, isCritical };
      });

      setGraph({
        nodes,
        edges,
        cycles: cycleNodes,
        criticalPath: criticalEdges,
        width: result.width ?? 800,
        height: result.height ?? 400,
      });
    });

    return () => { cancelled = true; };
  }, [issues, focusIssueId, showCompleted]);

  return graph;
}
