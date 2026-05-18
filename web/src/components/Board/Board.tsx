import { useState } from 'react';
import type { Issue } from '../../api/client';
import { LabelBadge, StatusIcon } from '../ui';

interface BoardProps {
  issues: Issue[];
  onRefresh: () => void;
  onIssueClick?: (issue: Issue) => void;
}

const COLUMNS = [
  { id: 'PLANNED', label: 'Planned' },
  { id: 'DOING', label: 'In Progress' },
  { id: 'BLOCKED', label: 'Blocked' },
  { id: 'DONE', label: 'Done' },
];

const DONE_VISIBLE_COUNT = 5;

export default function Board({ issues, onIssueClick }: BoardProps) {
  const boardIssues = issues.filter((i) => i.status !== 'BACKLOG');

  return (
    <div className="h-full flex flex-col">
      <div className="flex items-center gap-3 px-5 h-11 border-b border-[var(--color-border-subtle)] bg-[var(--color-bg-secondary)] shrink-0">
        <span className="text-[13px] font-medium text-[var(--color-text-primary)]">Board</span>
        <span className="text-[11px] text-[var(--color-text-muted)] tabular-nums">
          {boardIssues.length} issue{boardIssues.length !== 1 ? 's' : ''}
        </span>
      </div>

      <div className="flex-1 overflow-hidden p-3">
        <div className="flex gap-2.5 h-full">
          {COLUMNS.map((column) => {
            const columnIssues = boardIssues.filter((i) => i.status === column.id);
            return (
              <BoardColumn
                key={column.id}
                column={column}
                issues={columnIssues}
                onIssueClick={onIssueClick}
              />
            );
          })}
        </div>
      </div>
    </div>
  );
}

function BoardColumn({ column, issues, onIssueClick }: { column: { id: string; label: string }; issues: Issue[]; onIssueClick?: (issue: Issue) => void }) {
  const [showAll, setShowAll] = useState(false);
  const isDone = column.id === 'DONE';
  const shouldTruncate = isDone && issues.length > DONE_VISIBLE_COUNT && !showAll;
  const visibleIssues = shouldTruncate ? issues.slice(0, DONE_VISIBLE_COUNT) : issues;
  const hiddenCount = issues.length - DONE_VISIBLE_COUNT;

  return (
    <div className="flex flex-col flex-1 min-w-0 rounded-[var(--radius-md)] bg-[var(--color-bg-secondary)]/60">
      {/* Column header */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-[var(--color-border-subtle)]">
        <div className="flex items-center gap-2">
          <StatusIcon status={column.id} size={14} />
          <span className="text-[13px] font-medium text-[var(--color-text-primary)]">{column.label}</span>
        </div>
        <span className="text-[11px] text-[var(--color-text-muted)] tabular-nums">{issues.length}</span>
      </div>

      {/* Cards */}
      <div className="flex-1 p-1.5 space-y-1 overflow-y-auto">
        {visibleIssues.map((issue) => (
          <BoardCard key={issue.id} issue={issue} onClick={() => onIssueClick?.(issue)} />
        ))}
        {shouldTruncate && (
          <button
            onClick={() => setShowAll(true)}
            className="w-full py-2 text-[11px] text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] transition-colors"
          >
            + {hiddenCount} more
          </button>
        )}
        {issues.length === 0 && (
          <div className="flex items-center justify-center h-16 text-[var(--color-text-muted)] text-[11px]">
            No issues
          </div>
        )}
      </div>
    </div>
  );
}

function BoardCard({ issue, onClick }: { issue: Issue; onClick?: () => void }) {
  return (
    <div
      onClick={onClick}
      className="px-2.5 py-2 rounded-[var(--radius-sm)] bg-[var(--color-surface-elevated)] border border-[var(--color-border-subtle)] hover:border-[var(--color-border-default)] hover:bg-[var(--color-bg-hover)] cursor-pointer transition-colors duration-[var(--duration-fast)]"
    >
      {/* Title first — it's the most important thing */}
      <p className="text-[13px] text-[var(--color-text-primary)] leading-snug line-clamp-2 mb-1.5">
        {issue.title}
      </p>

      {/* Meta row */}
      <div className="flex items-center gap-2 text-[11px]">
        <span className="font-mono text-[var(--color-text-muted)]">{issue.id.replace('beats-', '')}</span>
        {issue.labels?.map((label) => (
          <LabelBadge key={label} label={label} />
        ))}
        {issue.estimate > 0 && (
          <span className="text-[var(--color-text-muted)] ml-auto tabular-nums">{issue.estimate}pt</span>
        )}
        {issue.is_pending && (
          <span className="w-1.5 h-1.5 rounded-full bg-[var(--color-warning)] ml-auto" />
        )}
      </div>
    </div>
  );
}
