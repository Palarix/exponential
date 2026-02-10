import { useState } from 'react';
import type { Issue } from '../../api/client';
import { Card, LabelBadge, StatusIcon } from '../ui';

interface BacklogProps {
  issues: Issue[];
  onRefresh: () => void;
  onIssueClick?: (issue: Issue) => void;
}

const STATUS_GROUPS = [
  { status: 'BACKLOG', label: 'Backlog', defaultExpanded: true },
  { status: 'PLANNED', label: 'Planned', defaultExpanded: true },
  { status: 'DOING', label: 'Doing', defaultExpanded: false },
  { status: 'DONE', label: 'Done', defaultExpanded: false },
] as const;

export default function Backlog({ issues, onIssueClick }: BacklogProps) {
  return (
    <div className="space-y-4 animate-fade-in">
      {/* Header */}
      <div className="flex items-center justify-between mb-2">
        <div>
          <h1 className="text-2xl font-bold text-[var(--color-text-primary)]">Backlog</h1>
          <p className="text-sm text-[var(--color-text-muted)] mt-1">
            {issues.length} total issues
          </p>
        </div>
      </div>

      {/* Status Tables */}
      {STATUS_GROUPS.map((group) => {
        const groupIssues = issues.filter((i) => i.status === group.status);
        return (
          <StatusTable
            key={group.status}
            status={group.status}
            label={group.label}
            issues={groupIssues}
            onIssueClick={onIssueClick}
            defaultExpanded={group.defaultExpanded}
          />
        );
      })}
    </div>
  );
}

/* ─── Helpers ───────────────────────────── */

function formatShortDate(dateStr: string): string {
  const d = new Date(dateStr);
  const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
  return `${months[d.getMonth()]} ${d.getDate()}`;
}

/* ─── StatusTable ───────────────────────── */

interface StatusTableProps {
  status: string;
  label: string;
  issues: Issue[];
  onIssueClick?: (issue: Issue) => void;
  defaultExpanded: boolean;
}

function StatusTable({
  status,
  label,
  issues,
  onIssueClick,
  defaultExpanded,
}: StatusTableProps) {
  const [isExpanded, setIsExpanded] = useState(defaultExpanded);

  return (
    <Card variant="default" padding="none">
      {/* Section Header */}
      <div
        className="
          flex items-center justify-between px-4 py-2.5
          border-b border-[var(--color-border-subtle)]
          select-none
        "
      >
        <button
          onClick={() => setIsExpanded(!isExpanded)}
          className="
            flex items-center gap-2
            text-[var(--color-text-primary)]
            hover:text-[var(--color-text-accent)]
            transition-colors duration-[var(--duration-fast)]
          "
        >
          {/* Chevron */}
          <svg
            className={`w-4 h-4 text-[var(--color-text-muted)] transition-transform duration-200 ${isExpanded ? 'rotate-90' : ''}`}
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            strokeWidth={2}
          >
            <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
          </svg>

          <span className="text-sm font-semibold">{label}</span>
          <span className="text-sm font-medium text-[var(--color-text-muted)] tabular-nums">
            ({issues.length})
          </span>
        </button>

        {/* Add Issue button */}
        <button
          onClick={() => console.log(`[TODO] Create new issue with status: ${status}`)}
          className="
            p-1 rounded-[var(--radius-sm)]
            text-[var(--color-text-muted)]
            hover:text-[var(--color-text-primary)]
            hover:bg-[var(--color-bg-hover)]
            transition-colors duration-[var(--duration-fast)]
          "
          title={`New ${label} issue`}
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
          </svg>
        </button>
      </div>

      {/* Table Body */}
      {isExpanded && (
        <div className="overflow-x-auto">
          <table className="w-full table-fixed">
            <colgroup>
              <col style={{ width: '120px' }} />  {/* issue ID */}
              <col style={{ width: '20px' }} />   {/* status icon */}
              <col />                              {/* title – fills remaining */}
              <col style={{ width: '140px' }} />   {/* labels */}
              <col style={{ width: '64px' }} />    {/* estimate */}
              <col style={{ width: '80px' }} />    {/* created date */}
            </colgroup>
            <tbody>
              {issues.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-[var(--color-text-muted)]">
                    <span className="text-sm">No issues</span>
                  </td>
                </tr>
              ) : (
                issues.map((issue) => (
                  <tr
                    key={issue.id}
                    onClick={() => onIssueClick?.(issue)}
                    className="
                      border-t border-[var(--color-border-subtle)]/30
                      hover:bg-[var(--color-bg-hover)]
                      cursor-pointer
                      transition-colors duration-[var(--duration-fast)]
                      group
                    "
                  >
                    {/* Issue ID */}
                    <td className="px-2 py-2.5 overflow-hidden">
                      <div className="flex items-center gap-1.5">
                        <span className="text-xs font-mono text-[var(--color-text-muted)] group-hover:text-[var(--color-text-accent)] transition-colors truncate">
                          {issue.id}
                        </span>
                        {issue.is_pending && (
                          <span className="inline-block w-1.5 h-1.5 rounded-full bg-[var(--color-warning)] animate-pulse-glow flex-shrink-0" />
                        )}
                      </div>
                    </td>

                    {/* Status Icon */}
                    <td className="py-2.5 pr-0">
                      <StatusIcon status={issue.status} size={14} />
                    </td>

                    {/* Title */}
                    <td className="px-2 py-2.5 overflow-hidden">
                      <span className="text-sm text-[var(--color-text-primary)] group-hover:text-[var(--color-text-accent)] transition-colors truncate block">
                        {issue.title}
                      </span>
                    </td>

                    {/* Labels */}
                    <td className="px-2 py-2.5 overflow-hidden">
                      <div className="flex gap-1 overflow-hidden">
                        {issue.labels?.map((label) => (
                          <LabelBadge key={label} label={label} />
                        ))}
                      </div>
                    </td>

                    {/* Estimate */}
                    <td className="px-2 py-2.5 text-right overflow-hidden">
                      <span className="text-xs text-[var(--color-text-muted)] tabular-nums">
                        {issue.estimate ? `${issue.estimate} pts` : '–'}
                      </span>
                    </td>

                    {/* Created Date */}
                    <td className="px-4 py-2.5 text-right overflow-hidden">
                      <span className="text-xs text-[var(--color-text-muted)] tabular-nums">
                        {formatShortDate(issue.created_at)}
                      </span>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  );
}
