import { useState } from 'react';
import type { Issue } from '../../api/client';
import { Card, LabelBadge, StatusBadge } from '../ui';

interface BacklogProps {
  issues: Issue[];
  onRefresh: () => void;
  onIssueClick?: (issue: Issue) => void;
}

export default function Backlog({ issues, onIssueClick }: BacklogProps) {
  const backlogIssues = issues.filter((i) => i.status === 'BACKLOG');
  const activeIssues = issues.filter((i) => ['PLANNED', 'DOING', 'BLOCKED'].includes(i.status));
  const doneIssues = issues.filter((i) => i.status === 'DONE');

  return (
    <div className="space-y-6 animate-fade-in">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-[var(--color-text-primary)]">Backlog</h1>
          <p className="text-sm text-[var(--color-text-muted)] mt-1">
            {issues.length} total issues
          </p>
        </div>
      </div>

      {/* Planned Table */}
      <IssueTable
        title={`Planned (${backlogIssues.length})`}
        issues={backlogIssues}
        onIssueClick={onIssueClick}
        collapsible
      />

      {/* Active Table */}
      <IssueTable
        title={`Active (${activeIssues.length})`}
        issues={activeIssues}
        onIssueClick={onIssueClick}
        collapsible
      />

      {/* Done Table */}
      <IssueTable
        title={`Done (${doneIssues.length})`}
        issues={doneIssues}
        onIssueClick={onIssueClick}
        collapsible
        defaultCollapsed
      />
    </div>
  );
}

interface IssueTableProps {
  title: string;
  issues: Issue[];
  onIssueClick?: (issue: Issue) => void;
  collapsible?: boolean;
  defaultCollapsed?: boolean;
}

function IssueTable({
  title,
  issues,
  onIssueClick,
  collapsible = false,
  defaultCollapsed = false
}: IssueTableProps) {
  const [isCollapsed, setIsCollapsed] = useState(defaultCollapsed);

  if (issues.length === 0 && !title) return null;

  return (
    <Card variant="default" padding="none">
      {title && (
        <div className="px-4 py-3 border-b border-[var(--color-border-subtle)] flex items-center justify-between">
          <h2 className="text-base font-semibold text-[var(--color-text-primary)]">{title}</h2>
          {collapsible && (
            <button
              onClick={() => setIsCollapsed(!isCollapsed)}
              className="
                p-1 rounded-md
                text-[var(--color-text-muted)]
                hover:text-[var(--color-text-primary)]
                hover:bg-[var(--color-bg-hover)]
                transition-colors duration-[var(--duration-fast)]
              "
              aria-label={isCollapsed ? 'Expand' : 'Collapse'}
            >
              <svg
                className={`w-5 h-5 transition-transform duration-200 ${isCollapsed ? '' : 'rotate-180'}`}
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={2}
              >
                <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
              </svg>
            </button>
          )}
        </div>
      )}
      {!isCollapsed && (
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-[var(--color-border-subtle)] bg-[var(--color-bg-tertiary)]/30">
                <th className="text-left px-4 py-3 text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">ID</th>
                <th className="text-left px-4 py-3 text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">Labels</th>
                <th className="text-left px-4 py-3 text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">Title</th>
                <th className="text-left px-4 py-3 text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">Status</th>
                <th className="text-right px-4 py-3 text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider">Estimate</th>
              </tr>
            </thead>
            <tbody>
              {issues.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-4 py-12 text-center text-[var(--color-text-muted)]">
                    <svg className="w-8 h-8 mx-auto mb-2 opacity-50" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
                    </svg>
                    <span className="text-sm">No issues</span>
                  </td>
                </tr>
              ) : (
                issues.map((issue) => (
                  <tr
                    key={issue.id}
                    onClick={() => onIssueClick?.(issue)}
                    className="
                      border-t border-[var(--color-border-subtle)]/50 
                      hover:bg-[var(--color-bg-hover)] 
                      cursor-pointer
                      transition-colors duration-[var(--duration-fast)]
                      group
                    "
                  >
                    <td className="px-4 py-3">
                      <span className="text-xs font-mono text-[var(--color-text-muted)] group-hover:text-[var(--color-text-accent)] transition-colors">
                        {issue.id}
                      </span>
                      {issue.is_pending && (
                        <span className="ml-2 inline-block w-2 h-2 rounded-full bg-[var(--color-warning)] animate-pulse-glow" />
                      )}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex gap-1 overflow-hidden">
                        {issue.labels?.map(label => (
                          <LabelBadge key={label} label={label} />
                        ))}
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <span className="text-sm text-[var(--color-text-primary)] group-hover:text-[var(--color-text-accent)] transition-colors">
                        {issue.title}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <StatusBadge status={issue.status} />
                    </td>
                    <td className="px-4 py-3 text-right">
                      <span className="text-sm text-[var(--color-text-muted)]">
                        {issue.estimate ? `${issue.estimate} pts` : '–'}
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
