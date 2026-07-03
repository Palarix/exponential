import type { PendingState, Issue } from '../../api/client';
import { StatusIcon, LabelBadge } from '../ui';
import { ChevronRight } from "lucide-react";

interface PendingChangesProps {
  pending: PendingState;
  issues: Issue[];
  autoCommit: boolean;
  onClose: () => void;
  onSave: () => void;
  onDiscard: () => void;
  onIssueClick: (issue: Issue) => void;
}

export default function PendingChanges({ pending, issues, autoCommit, onClose, onSave, onDiscard, onIssueClick }: PendingChangesProps) {
  const events = pending.events || [];
  const issueMap = new Map(issues.map(i => [i.id, i]));

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between px-5 h-11 border-b border-[var(--color-border-subtle)] shrink-0">
        <div className="flex items-center gap-2">
          <button
            onClick={onClose}
            className="text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors"
          >
            Issues
          </button>
          <ChevronRight className="w-3 h-3 text-[var(--color-text-muted)]" />
          <span className="text-sm font-medium text-[var(--color-text-primary)]">Pending Changes</span>
          <span className="text-xs text-[var(--color-text-muted)] tabular-nums">{events.length}</span>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={onDiscard}
            className="px-3 py-1 text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)] rounded-[var(--radius-md)] transition-colors"
          >
            Discard All
          </button>
          <button
            onClick={onSave}
            className="px-3 py-1 text-sm text-white bg-[var(--color-accent-primary)] hover:bg-[var(--color-accent-primary-hover)] rounded-[var(--radius-md)] transition-colors"
          >
            {autoCommit ? 'Save & Commit' : 'Save'}
          </button>
        </div>
      </div>

      {/* Changes list */}
      <div className="flex-1 overflow-y-auto">
        {events.length === 0 ? (
          <div className="flex items-center justify-center h-full text-sm text-[var(--color-text-muted)]">
            No pending changes
          </div>
        ) : (
          <div>
            {events.map((event, i) => {
              const issue = issueMap.get(event.issue_id);
              return (
                <div
                  key={i}
                  className="flex items-start gap-3 px-5 py-3 border-b border-[var(--color-border-subtle)] hover:bg-[var(--color-hover-surface)] transition-colors"
                >
                  {/* Type badge */}
                  <TypeBadge type={event.type} />

                  {/* Details */}
                  <div className="flex-1 min-w-0">
                    {/* Issue reference */}
                    {issue ? (
                      <button
                        onClick={() => onIssueClick(issue)}
                        className="flex items-center gap-2 mb-2 group"
                      >
                        <StatusIcon status={issue.status} size={14} isInferred={issue.is_inferred} />
                        <span className="text-sm font-medium text-[var(--color-text-primary)] group-hover:text-[var(--color-accent-primary)] transition-colors truncate">
                          {issue.title}
                        </span>
                        <span className="text-xs font-mono text-[var(--color-text-muted)] shrink-0">
                          {event.issue_id}
                        </span>
                      </button>
                    ) : (
                      <div className="flex items-center gap-2 mb-2">
                        <span className="text-sm font-mono text-[var(--color-text-muted)]">{event.issue_id}</span>
                      </div>
                    )}

                    {/* Payload details */}
                    {event.payload && (
                      <PayloadDetail type={event.type} payload={event.payload} />
                    )}

                    <span className="text-xs text-[var(--color-text-muted)]">
                      {formatTime(event.created_at)}
                    </span>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}

function TypeBadge({ type }: { type: string }) {
  const config: Record<string, { label: string; color: string }> = {
    CREATE: { label: 'New', color: 'var(--color-success)' },
    UPDATE: { label: 'Edit', color: 'var(--color-info)' },
    COMMENT: { label: 'Comment', color: 'var(--color-accent-primary)' },
    DELETE: { label: 'Delete', color: 'var(--color-error)' },
  };
  const c = config[type] || { label: type, color: 'var(--color-text-muted)' };

  return (
    <span
      className="text-xs font-medium px-2 py-1 rounded-[var(--radius-sm)] shrink-0 mt-1"
      style={{ background: `color-mix(in srgb, ${c.color} 15%, transparent)`, color: c.color }}
    >
      {c.label}
    </span>
  );
}

function PayloadDetail({ type, payload }: { type: string; payload: Record<string, unknown> }) {
  if (type === 'CREATE') {
    return (
      <div className="mb-2 space-y-1">
        {payload.title != null && <div className="text-sm text-[var(--color-text-secondary)]">Title: {String(payload.title)}</div>}
        {payload.labels != null && (
          <div className="flex items-center gap-2">
            {(payload.labels as string[]).map(l => <LabelBadge key={l} label={l} />)}
          </div>
        )}
      </div>
    );
  }

  if (type === 'UPDATE') {
    const entries = Object.entries(payload).filter(([, v]) => v !== undefined);
    if (entries.length === 0) return null;

    return (
      <div className="mb-2 space-y-1">
        {entries.map(([key, value]) => (
          <div key={key} className="text-sm">
            <span className="text-[var(--color-text-muted)]">{key}:</span>{' '}
            <span className="text-[var(--color-text-secondary)]">
              {Array.isArray(value) ? value.join(', ') : String(value)}
            </span>
          </div>
        ))}
      </div>
    );
  }

  if (type === 'COMMENT') {
    return (
      <div className="mb-2 text-sm text-[var(--color-text-secondary)] line-clamp-2">
        {String(payload.text || '')}
      </div>
    );
  }

  return null;
}

function formatTime(dateStr: string): string {
  return new Date(dateStr).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}
