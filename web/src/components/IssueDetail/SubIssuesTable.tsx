import { useState, useRef, useCallback, useMemo } from "react";
import { createIssue } from "../../api/client";
import type { Issue } from "../../api/client";
import { computeAppendKey } from "../../utils/sort";
import { Avatar, LabelBadge, StatusIcon } from "../ui";

export default function SubIssuesTable({ issue, issues, onRefresh }: { issue: Issue; issues: Issue[]; onRefresh: () => void }) {
  const children = useMemo(() => issues.filter(i => i.parent_id === issue.id), [issues, issue.id]);
  const [expanded, setExpanded] = useState(true);
  const [inlineTitle, setInlineTitle] = useState("");
  const [showInline, setShowInline] = useState(false);
  const inlineRef = useRef<HTMLInputElement>(null);

  const hasChildren = children.length > 0;
  const doneCount = children.filter(c => c.status === 'DONE').length;

  const handleInlineCreate = useCallback(async (title: string) => {
    if (!title.trim()) return;
    const sortOrder = computeAppendKey(issues, "BACKLOG", issue.id);
    await createIssue({ title: title.trim(), labels: ['feature'], parent_id: issue.id, sort_order: sortOrder });
    setInlineTitle("");
    setShowInline(false);
    onRefresh();
  }, [issue.id, issues, onRefresh]);

  const startInline = useCallback(() => {
    setShowInline(true);
    setExpanded(true);
    setTimeout(() => inlineRef.current?.focus(), 0);
  }, []);

  if (!hasChildren && !showInline) {
    return (
      <div className="mt-6">
        <button
          onClick={startInline}
          className="flex items-center gap-2 text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] transition-colors"
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
          </svg>
          Add sub-issue
        </button>
      </div>
    );
  }

  return (
    <div className="mt-6">
      <div className="flex items-center gap-2 mb-3">
        <button
          onClick={() => setExpanded(v => !v)}
          className="flex items-center gap-2 text-sm font-semibold text-[var(--color-text-primary)] hover:text-[var(--color-text-secondary)] transition-colors"
        >
          <svg
            className={`w-3 h-3 text-[var(--color-text-muted)] transition-transform duration-100 ${expanded ? 'rotate-90' : ''}`}
            fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}
          >
            <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
          </svg>
          Sub-issues
          {hasChildren && <span className="text-xs font-normal text-[var(--color-text-muted)]">{doneCount}/{children.length}</span>}
        </button>
        <button
          onClick={startInline}
          className="p-0.5 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors"
          title="Add sub-issue"
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
          </svg>
        </button>
      </div>
      {expanded && (
        <div className="rounded-[var(--radius-md)] border border-[var(--color-border-default)] overflow-hidden">
          {children.map((child, i) => (
            <a
              key={child.id}
              href={`#/issues/${child.id}`}
              className={`flex items-center gap-2.5 px-3 py-2 text-sm hover:bg-[var(--color-bg-hover)] transition-colors ${i > 0 ? 'border-t border-[var(--color-border-subtle)]' : ''}`}
            >
              <StatusIcon status={child.status} size={14} />
              <span className="text-[var(--color-text-primary)] truncate min-w-0">{child.title}</span>
              {child.priority > 0 && (
                <span className={`text-xs font-medium shrink-0 ${child.priority === 1 ? 'text-[var(--color-error)]' : child.priority === 2 ? 'text-[var(--color-warning)]' : 'text-[var(--color-text-muted)]'}`}>
                  {child.priority === 1 ? '!!!' : child.priority === 2 ? '!!' : '!'}
                </span>
              )}
              <div className="flex-1" />
              {child.labels?.map(label => <LabelBadge key={label} label={label} />)}
              {child.estimate > 0 && (
                <span className="flex items-center gap-1 text-xs text-[var(--color-text-muted)] tabular-nums shrink-0">
                  <svg className="w-3 h-3" viewBox="0 0 16 16" fill="none"><path d="M8 2L14 14H2L8 2Z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" /></svg>
                  {child.estimate}
                </span>
              )}
              {child.created_by && (
                <Avatar name={child.created_by} size="sm" />
              )}
            </a>
          ))}
          {showInline && (
            <div className={`flex items-center gap-2.5 px-3 py-2 ${hasChildren ? 'border-t border-[var(--color-border-subtle)]' : ''}`}>
              <StatusIcon status="BACKLOG" size={14} />
              <input
                ref={inlineRef}
                value={inlineTitle}
                onChange={(e) => setInlineTitle(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && inlineTitle.trim()) handleInlineCreate(inlineTitle);
                  if (e.key === 'Escape') { setShowInline(false); setInlineTitle(""); }
                }}
                onBlur={() => { if (!inlineTitle.trim()) { setShowInline(false); setInlineTitle(""); } }}
                placeholder="Sub-issue title... (Enter to create, Esc to cancel)"
                className="flex-1 bg-transparent text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none"
              />
            </div>
          )}
        </div>
      )}
    </div>
  );
}
