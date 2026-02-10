import { useState } from 'react';
import type { Issue } from '../../api/client';
import { Card, LabelBadge } from '../ui';

interface BoardProps {
  issues: Issue[];
  onRefresh: () => void;
  onIssueClick?: (issue: Issue) => void;
}

const COLUMNS = [
  { id: 'PLANNED', label: 'Planned', color: 'var(--color-status-planned)', icon: '📋' },
  { id: 'DOING', label: 'In Progress', color: 'var(--color-status-doing)', icon: '🔄' },
  { id: 'BLOCKED', label: 'Blocked', color: 'var(--color-status-blocked)', icon: '🚫' },
  { id: 'DONE', label: 'Done', color: 'var(--color-status-done)', icon: '✅' },
];

export default function Board({ issues, onIssueClick }: BoardProps) {
  const [hoveredColumn, setHoveredColumn] = useState<string | null>(null);

  // Filter out BACKLOG by default and group by status
  const boardIssues = issues.filter((i) => i.status !== 'BACKLOG');

  return (
    <div className="space-y-6 animate-fade-in">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-[var(--color-text-primary)]">Board</h1>
          <p className="text-sm text-[var(--color-text-muted)] mt-1">
            {boardIssues.length} issue{boardIssues.length !== 1 ? 's' : ''} on board
          </p>
        </div>
      </div>

      {/* Kanban Columns */}
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
        {COLUMNS.map((column) => {
          const columnIssues = boardIssues.filter((i) => i.status === column.id);

          return (
            <div
              key={column.id}
              className={`
                flex flex-col rounded-[var(--radius-lg)] 
                bg-[var(--color-bg-secondary)]/50
                border border-[var(--color-border-subtle)]
                transition-all duration-[var(--duration-normal)]
                ${hoveredColumn === column.id ? 'border-[var(--color-border-accent)]' : ''}
              `.trim().replace(/\s+/g, ' ')}
              onMouseEnter={() => setHoveredColumn(column.id)}
              onMouseLeave={() => setHoveredColumn(null)}
            >
              {/* Column Header */}
              <div
                className="px-4 py-3 border-b border-[var(--color-border-subtle)] flex items-center justify-between"
                style={{
                  background: `linear-gradient(135deg, ${column.color}20, transparent)`
                }}
              >
                <div className="flex items-center gap-2">
                  <span
                    className="font-semibold text-sm"
                    style={{ color: column.color }}
                  >
                    {column.label}
                  </span>
                </div>
                <span
                  className="px-2 py-0.5 text-xs font-medium rounded-[var(--radius-full)]"
                  style={{
                    background: `${column.color}30`,
                    color: column.color
                  }}
                >
                  {columnIssues.length}
                </span>
              </div>

              {/* Cards */}
              <div className="p-2 space-y-2 flex-1 min-h-[300px] overflow-y-auto">
                {columnIssues.map((issue) => (
                  <IssueCard
                    key={issue.id}
                    issue={issue}
                    onClick={() => onIssueClick?.(issue)}
                  />
                ))}
                {columnIssues.length === 0 && (
                  <div className="flex flex-col items-center justify-center h-full py-8 text-[var(--color-text-muted)]">
                    <svg className="w-8 h-8 mb-2 opacity-50" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
                    </svg>
                    <span className="text-xs">No issues</span>
                  </div>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function IssueCard({ issue, onClick }: { issue: Issue; onClick?: () => void }) {


  return (
    <Card
      variant="elevated"
      interactive
      glow
      padding="sm"
      onClick={onClick}
      className="group"
    >
      {/* Header */}
      <div className="flex items-center justify-between gap-2 mb-2">
        <div className="flex gap-1 overflow-hidden">
          {issue.labels?.map(label => (
            <LabelBadge key={label} label={label} />
          ))}
        </div>
        <span className="text-[10px] font-mono text-[var(--color-text-muted)] opacity-60 group-hover:opacity-100 transition-opacity">
          {issue.id}
        </span>
      </div>

      {/* Title */}
      <p className="text-sm font-medium text-[var(--color-text-primary)] line-clamp-2 mb-2 group-hover:text-[var(--color-text-accent)] transition-colors">
        {issue.title}
      </p>

      {/* Footer */}
      <div className="flex items-center justify-between text-xs text-[var(--color-text-muted)]">
        <div className="flex items-center gap-3">
          {issue.estimate !== undefined && issue.estimate > 0 && (
            <span className="flex items-center gap-1">
              <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              {issue.estimate} pts
            </span>
          )}
        </div>
        {issue.is_pending && (
          <span className="px-1.5 py-0.5 rounded-[var(--radius-sm)] bg-[var(--color-warning-bg)] text-[var(--color-warning)] text-[10px]">
            pending
          </span>
        )}
      </div>


    </Card>
  );
}
