import { useState, useRef, useCallback, useEffect, type WheelEvent as ReactWheelEvent, type MouseEvent as ReactMouseEvent } from 'react';
import type { Issue } from '../../api/types';
import type { DepGraph, GraphEdge } from './useDepGraph';
import StatusIcon from '../ui/StatusIcon';
import { Toggle } from '../ui';

const NODE_WIDTH = 240;
const NODE_HEIGHT = 56;

const KIND_COLORS: Record<string, string> = {
  blocks: 'var(--color-error)',
  depends_on: 'var(--color-error)',
  relates_to: 'var(--color-info)',
  related: 'var(--color-info)',
  duplicates: 'var(--color-text-muted)',
};

function truncate(s: string, max: number): string {
  return s.length > max ? s.slice(0, max) + '…' : s;
}

function edgePath(points: { x: number; y: number }[]): string {
  if (points.length === 0) return '';
  if (points.length === 1) return `M${points[0].x},${points[0].y}`;

  let d = `M${points[0].x},${points[0].y}`;
  if (points.length === 2) {
    d += `L${points[1].x},${points[1].y}`;
    return d;
  }

  for (let i = 1; i < points.length - 1; i++) {
    const prev = points[i - 1];
    const curr = points[i];
    const next = points[i + 1];
    const cpx1 = (prev.x + curr.x) / 2;
    const cpy1 = (prev.y + curr.y) / 2;
    const cpx2 = (curr.x + next.x) / 2;
    const cpy2 = (curr.y + next.y) / 2;
    if (i === 1) d = `M${cpx1},${cpy1}`;
    d += `Q${curr.x},${curr.y} ${cpx2},${cpy2}`;
  }

  return d;
}

interface DepGraphViewProps {
  graph: DepGraph;
  focusIssue: Issue;
  showCompleted: boolean;
  onShowCompletedChange: (v: boolean) => void;
  onIssueClick?: (issue: Issue) => void;
  onBack: () => void;
}

const MIN_SCALE = 0.2;
const MAX_SCALE = 2;

export default function DepGraphView({ graph, focusIssue, showCompleted, onShowCompletedChange, onIssueClick, onBack }: DepGraphViewProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [hoveredNode, setHoveredNode] = useState<string | null>(null);
  const [transform, setTransform] = useState({ x: 0, y: 0, scale: 1 });
  const dragRef = useRef<{ startX: number; startY: number; startTx: number; startTy: number } | null>(null);

  const fitToScreen = useCallback(() => {
    const container = containerRef.current;
    if (!container || graph.width === 0) return;
    const rect = container.getBoundingClientRect();
    const pad = 60;
    const scaleX = (rect.width - pad * 2) / graph.width;
    const scaleY = (rect.height - pad * 2) / graph.height;
    const scale = Math.min(Math.max(Math.min(scaleX, scaleY), MIN_SCALE), MAX_SCALE);
    const x = (rect.width - graph.width * scale) / 2;
    const y = (rect.height - graph.height * scale) / 2;
    setTransform({ x, y, scale });
  }, [graph.width, graph.height]);

  useEffect(() => {
    fitToScreen();
  }, [fitToScreen]);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') { e.preventDefault(); onBack(); }
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [onBack]);

  const handleWheel = useCallback((e: ReactWheelEvent) => {
    e.preventDefault();
    const container = containerRef.current;
    if (!container) return;
    const rect = container.getBoundingClientRect();
    const mouseX = e.clientX - rect.left;
    const mouseY = e.clientY - rect.top;

    setTransform(prev => {
      const factor = e.deltaY > 0 ? 0.92 : 1.08;
      const newScale = Math.min(Math.max(prev.scale * factor, MIN_SCALE), MAX_SCALE);
      const ratio = newScale / prev.scale;
      return {
        scale: newScale,
        x: mouseX - (mouseX - prev.x) * ratio,
        y: mouseY - (mouseY - prev.y) * ratio,
      };
    });
  }, []);

  const handleMouseDown = useCallback((e: ReactMouseEvent) => {
    if (e.button !== 0) return;
    if ((e.target as HTMLElement).closest('[data-graph-node]')) return;
    e.preventDefault();
    dragRef.current = { startX: e.clientX, startY: e.clientY, startTx: transform.x, startTy: transform.y };
  }, [transform.x, transform.y]);

  const handleMouseMove = useCallback((e: ReactMouseEvent) => {
    if (!dragRef.current) return;
    const dx = e.clientX - dragRef.current.startX;
    const dy = e.clientY - dragRef.current.startY;
    setTransform(prev => ({ ...prev, x: dragRef.current!.startTx + dx, y: dragRef.current!.startTy + dy }));
  }, []);

  const handleMouseUp = useCallback(() => {
    dragRef.current = null;
  }, []);

  const connectedEdges = hoveredNode
    ? new Set(graph.edges.filter(e => e.sourceId === hoveredNode || e.targetId === hoveredNode).map(e => `${e.sourceId}->${e.targetId}`))
    : null;

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="flex items-center gap-3 px-5 h-11 border-b border-[var(--color-border-subtle)] shrink-0">
        <button
          onClick={onBack}
          className="flex items-center justify-center w-7 h-7 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)] transition-colors"
          title="Back to table"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M19 12H5M12 19l-7-7 7-7" />
          </svg>
        </button>
        <span className="text-sm font-medium text-[var(--color-text-primary)] truncate">
          Dependencies for {truncate(focusIssue.title, 50)}
        </span>
        <span className="text-xs font-mono text-[var(--color-text-muted)]">{focusIssue.id}</span>

        <div className="flex-1" />

        <Toggle
          checked={showCompleted}
          onChange={onShowCompletedChange}
          label="Completed"
        />

        <button
          className="flex items-center justify-center w-7 h-7 rounded-[var(--radius-sm)] border border-[var(--color-border-subtle)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)] transition-colors"
          onClick={fitToScreen}
          title="Fit to screen"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M8 3H5a2 2 0 00-2 2v3m18 0V5a2 2 0 00-2-2h-3m0 18h3a2 2 0 002-2v-3M3 16v3a2 2 0 002 2h3" />
          </svg>
        </button>
      </div>

      {graph.nodes.length === 0 ? (
        <div className="flex-1 flex items-center justify-center">
          <p className="text-sm text-[var(--color-text-muted)]">This issue has no dependencies</p>
        </div>
      ) : (
        <div
          ref={containerRef}
          className="flex-1 min-h-0 overflow-hidden cursor-grab active:cursor-grabbing"
          onWheel={handleWheel}
          onMouseDown={handleMouseDown}
          onMouseMove={handleMouseMove}
          onMouseUp={handleMouseUp}
          onMouseLeave={handleMouseUp}
        >
          <svg width="100%" height="100%" style={{ overflow: 'visible' }}>
            <defs>
              {Object.entries(KIND_COLORS).map(([kind, color]) => (
                <marker
                  key={kind}
                  id={`arrow-${kind}`}
                  viewBox="0 0 10 10"
                  refX="9"
                  refY="5"
                  markerWidth="6"
                  markerHeight="6"
                  orient="auto-start-reverse"
                >
                  <path d="M 0 0 L 10 5 L 0 10 z" fill={color} />
                </marker>
              ))}
              <marker
                id="arrow-critical"
                viewBox="0 0 10 10"
                refX="9"
                refY="5"
                markerWidth="7"
                markerHeight="7"
                orient="auto-start-reverse"
              >
                <path d="M 0 0 L 10 5 L 0 10 z" fill="var(--color-accent-primary)" />
              </marker>
              <marker
                id="arrow-cycle"
                viewBox="0 0 10 10"
                refX="9"
                refY="5"
                markerWidth="6"
                markerHeight="6"
                orient="auto-start-reverse"
              >
                <path d="M 0 0 L 10 5 L 0 10 z" fill="var(--color-warning)" />
              </marker>
            </defs>

            <g transform={`translate(${transform.x}, ${transform.y}) scale(${transform.scale})`}>
              {graph.edges.map((edge, i) => (
                <EdgePath
                  key={i}
                  edge={edge}
                  dimmed={connectedEdges !== null && !connectedEdges.has(`${edge.sourceId}->${edge.targetId}`)}
                  highlighted={connectedEdges !== null && connectedEdges.has(`${edge.sourceId}->${edge.targetId}`)}
                />
              ))}

              {graph.nodes.map(node => (
                <foreignObject
                  key={node.issue.id}
                  x={node.x - NODE_WIDTH / 2}
                  y={node.y - NODE_HEIGHT / 2}
                  width={NODE_WIDTH}
                  height={NODE_HEIGHT}
                >
                  <GraphNodeCard
                    issue={node.issue}
                    isFocus={node.issue.id === focusIssue.id}
                    isCycle={graph.cycles.has(node.issue.id)}
                    isHovered={hoveredNode === node.issue.id}
                    isDimmed={hoveredNode !== null && hoveredNode !== node.issue.id &&
                      !graph.edges.some(e =>
                        (e.sourceId === hoveredNode && e.targetId === node.issue.id) ||
                        (e.targetId === hoveredNode && e.sourceId === node.issue.id)
                      )}
                    onMouseEnter={() => setHoveredNode(node.issue.id)}
                    onMouseLeave={() => setHoveredNode(null)}
                    onClick={() => onIssueClick?.(node.issue)}
                  />
                </foreignObject>
              ))}
            </g>
          </svg>
        </div>
      )}
    </div>
  );
}

function EdgePath({ edge, dimmed, highlighted }: { edge: GraphEdge; dimmed: boolean; highlighted: boolean }) {
  const color = edge.isCycle
    ? 'var(--color-warning)'
    : edge.isCritical
      ? 'var(--color-accent-primary)'
      : KIND_COLORS[edge.kind] ?? 'var(--color-text-muted)';

  const markerId = edge.isCycle ? 'arrow-cycle' : edge.isCritical ? 'arrow-critical' : `arrow-${edge.kind}`;
  const strokeWidth = edge.isCritical ? 2.5 : highlighted ? 2 : 1.5;

  return (
    <path
      d={edgePath(edge.points)}
      fill="none"
      stroke={color}
      strokeWidth={strokeWidth}
      strokeDasharray={edge.isCycle ? '6 4' : undefined}
      markerEnd={`url(#${markerId})`}
      opacity={dimmed ? 0.15 : 1}
      style={{ transition: 'opacity 150ms' }}
    />
  );
}

function GraphNodeCard({
  issue,
  isFocus,
  isCycle,
  isHovered,
  isDimmed,
  onMouseEnter,
  onMouseLeave,
  onClick,
}: {
  issue: Issue;
  isFocus: boolean;
  isCycle: boolean;
  isHovered: boolean;
  isDimmed: boolean;
  onMouseEnter: () => void;
  onMouseLeave: () => void;
  onClick: () => void;
}) {
  const isDone = issue.status === 'DONE';

  return (
    <div
      data-graph-node
      className="h-full flex items-center gap-2.5 px-3 rounded-lg cursor-pointer select-none border transition-all duration-150"
      style={{
        background: 'var(--color-surface-1)',
        borderColor: isHovered
          ? 'var(--color-accent-primary)'
          : isFocus
            ? 'var(--color-accent-primary)'
            : isCycle
              ? 'var(--color-warning)'
              : 'var(--color-border-subtle)',
        borderWidth: isFocus ? 2 : 1,
        opacity: isDimmed ? 0.3 : isDone ? 0.55 : 1,
        boxShadow: isHovered ? '0 0 0 1px var(--color-accent-primary)' : undefined,
      }}
      onMouseEnter={onMouseEnter}
      onMouseLeave={onMouseLeave}
      onClick={onClick}
    >
      <StatusIcon status={issue.status} size={14} />
      <div className="flex-1 min-w-0">
        <p
          className="text-xs leading-tight truncate"
          style={{
            color: 'var(--color-text-primary)',
            textDecoration: isDone ? 'line-through' : undefined,
          }}
          title={issue.title}
        >
          {truncate(issue.title, 40)}
        </p>
        <div className="flex items-center gap-1.5 mt-0.5">
          <span className="text-[10px] font-mono" style={{ color: 'var(--color-text-muted)' }}>
            {issue.id}
          </span>
          {isCycle && (
            <svg width="12" height="12" viewBox="0 0 16 16" fill="var(--color-warning)">
              <path d="M8 1l7 13H1L8 1z" />
              <text x="8" y="12" textAnchor="middle" fill="var(--color-surface-1)" fontSize="9" fontWeight="bold">!</text>
            </svg>
          )}
        </div>
      </div>
    </div>
  );
}
