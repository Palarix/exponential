import { useState, useRef, useEffect } from 'react';
import type { Issue } from '../../api/client';
import { LabelBadge, StatusIcon } from '../ui';

interface BacklogProps {
  issues: Issue[];
  onRefresh: () => void;
  onIssueClick?: (issue: Issue) => void;
  searchFocused?: boolean;
  onSearchBlur?: () => void;
}

type Tab = 'all' | 'active' | 'backlog';

const TAB_CONFIGS: Record<Tab, { label: string; statuses: string[] }> = {
  all: { label: 'All Issues', statuses: ['BACKLOG', 'PLANNED', 'DOING', 'BLOCKED', 'DONE'] },
  active: { label: 'Active', statuses: ['DOING', 'BLOCKED'] },
  backlog: { label: 'Backlog', statuses: ['BACKLOG', 'PLANNED'] },
};

const STATUS_META: Record<string, { label: string }> = {
  BACKLOG: { label: 'Backlog' },
  PLANNED: { label: 'Planned' },
  DOING: { label: 'In Progress' },
  BLOCKED: { label: 'Blocked' },
  DONE: { label: 'Done' },
};

export default function Backlog({ issues, onIssueClick, searchFocused, onSearchBlur }: BacklogProps) {
  const [activeTab, setActiveTab] = useState<Tab>('all');
  const [search, setSearch] = useState('');
  const searchRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (searchFocused && searchRef.current) {
      searchRef.current.focus();
    }
  }, [searchFocused]);

  const visibleStatuses = TAB_CONFIGS[activeTab].statuses;
  const query = search.toLowerCase();
  const filteredIssues = issues.filter((i) =>
    visibleStatuses.includes(i.status) &&
    (!query || i.title.toLowerCase().includes(query) || i.id.toLowerCase().includes(query) || i.labels?.some(l => l.toLowerCase().includes(query)))
  );

  const childrenByParent = new Map<string, Issue[]>();
  for (const issue of issues) {
    if (issue.parent_id) {
      const siblings = childrenByParent.get(issue.parent_id) || [];
      siblings.push(issue);
      childrenByParent.set(issue.parent_id, siblings);
    }
  }

  return (
    <div className="h-full flex flex-col">
      {/* Tab bar */}
      <div className="flex items-center gap-4 px-5 h-11 border-b border-[var(--color-border-subtle)] bg-[var(--color-bg-secondary)] shrink-0">
        {(Object.entries(TAB_CONFIGS) as [Tab, { label: string }][]).map(([id, config]) => (
          <button
            key={id}
            onClick={() => setActiveTab(id)}
            className={`
              text-[13px] font-medium h-full border-b-[1.5px] -mb-px transition-colors duration-[var(--duration-fast)]
              ${activeTab === id
                ? 'text-[var(--color-text-primary)] border-[var(--color-text-primary)]'
                : 'text-[var(--color-text-muted)] border-transparent hover:text-[var(--color-text-secondary)]'
              }
            `.trim().replace(/\s+/g, ' ')}
          >
            {config.label}
          </button>
        ))}
        <div className="ml-auto flex items-center gap-3">
          {(search || searchFocused) && (
            <div className="flex items-center gap-1.5 bg-[var(--color-bg-tertiary)] rounded-[var(--radius-md)] px-2 py-1">
              <svg className="w-3.5 h-3.5 text-[var(--color-text-muted)] shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z" />
              </svg>
              <input
                ref={searchRef}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                onBlur={() => { if (!search) onSearchBlur?.(); }}
                onKeyDown={(e) => { if (e.key === 'Escape') { setSearch(''); onSearchBlur?.(); } }}
                placeholder="Filter issues..."
                className="bg-transparent text-[12px] text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none w-36"
              />
            </div>
          )}
          <span className="text-[11px] text-[var(--color-text-muted)] tabular-nums">
            {filteredIssues.length} issue{filteredIssues.length !== 1 ? 's' : ''}
          </span>
        </div>
      </div>

      {/* Issue list */}
      <div className="flex-1 overflow-y-auto">
        {visibleStatuses.map((status) => {
          const groupIssues = issues.filter((i) => i.status === status);
          const meta = STATUS_META[status];
          if (!meta) return null;
          if (groupIssues.length === 0 && activeTab !== 'all') return null;

          return (
            <StatusGroup
              key={status}
              status={status}
              label={meta.label}
              issues={groupIssues}
              allIssues={issues}
              childrenByParent={childrenByParent}
              onIssueClick={onIssueClick}
              defaultExpanded={groupIssues.length > 0 && status !== 'DONE'}
            />
          );
        })}
      </div>
    </div>
  );
}

function formatShortDate(dateStr: string): string {
  const d = new Date(dateStr);
  const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
  return `${months[d.getMonth()]} ${d.getDate()}`;
}

interface StatusGroupProps {
  status: string;
  label: string;
  issues: Issue[];
  allIssues: Issue[];
  childrenByParent: Map<string, Issue[]>;
  onIssueClick?: (issue: Issue) => void;
  defaultExpanded: boolean;
}

function StatusGroup({ status, label, issues, allIssues, childrenByParent, onIssueClick, defaultExpanded }: StatusGroupProps) {
  const [isExpanded, setIsExpanded] = useState(defaultExpanded);
  const isEmpty = issues.length === 0;

  const topLevel = issues.filter(i => !i.parent_id || !allIssues.some(p => p.id === i.parent_id));

  return (
    <div>
      <button
        onClick={() => !isEmpty && setIsExpanded(!isExpanded)}
        className={`
          flex items-center gap-2 w-full px-5 py-2 border-b border-[var(--color-border-subtle)]
          bg-[var(--color-bg-secondary)] transition-colors duration-[var(--duration-fast)] select-none
          ${isEmpty ? 'opacity-40 cursor-default' : 'hover:bg-[var(--color-bg-hover)] cursor-pointer'}
        `.trim().replace(/\s+/g, ' ')}
      >
        <svg
          className={`w-3 h-3 text-[var(--color-text-muted)] transition-transform duration-100 ${isExpanded && !isEmpty ? 'rotate-90' : ''}`}
          fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}
        >
          <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
        </svg>
        <StatusIcon status={status} size={14} />
        <span className="text-[13px] font-medium text-[var(--color-text-primary)]">{label}</span>
        <span className="text-[12px] text-[var(--color-text-muted)] tabular-nums">{issues.length}</span>
      </button>

      {isExpanded && !isEmpty && topLevel.map((issue) => (
        <IssueTree
          key={issue.id}
          issue={issue}
          childrenByParent={childrenByParent}
          onIssueClick={onIssueClick}
          depth={0}
        />
      ))}
    </div>
  );
}

function IssueTree({ issue, childrenByParent, onIssueClick, depth }: {
  issue: Issue;
  childrenByParent: Map<string, Issue[]>;
  onIssueClick?: (issue: Issue) => void;
  depth: number;
}) {
  const children = childrenByParent.get(issue.id) || [];
  const hasChildren = children.length > 0;
  const [expanded, setExpanded] = useState(true);
  const doneChildren = children.filter(c => c.status === 'DONE').length;

  return (
    <>
      <IssueRow
        issue={issue}
        depth={depth}
        hasChildren={hasChildren}
        expanded={expanded}
        onToggle={() => setExpanded(!expanded)}
        subProgress={hasChildren ? { done: doneChildren, total: children.length } : undefined}
        onClick={() => onIssueClick?.(issue)}
      />
      {hasChildren && expanded && children.map((child) => (
        <IssueTree
          key={child.id}
          issue={child}
          childrenByParent={childrenByParent}
          onIssueClick={onIssueClick}
          depth={depth + 1}
        />
      ))}
    </>
  );
}

function IssueRow({ issue, depth, hasChildren, expanded, onToggle, subProgress, onClick }: {
  issue: Issue;
  depth: number;
  hasChildren: boolean;
  expanded: boolean;
  onToggle: () => void;
  subProgress?: { done: number; total: number };
  onClick?: () => void;
}) {
  const indent = depth * 24;

  return (
    <div
      onClick={onClick}
      className="flex items-center gap-2.5 px-5 h-[38px] border-b border-[var(--color-border-subtle)] hover:bg-[var(--color-bg-hover)] cursor-pointer transition-colors duration-[var(--duration-fast)] group"
      style={{ paddingLeft: `${20 + indent}px` }}
    >
      {/* Tree toggle or connector */}
      {hasChildren ? (
        <button
          onClick={(e) => { e.stopPropagation(); onToggle(); }}
          className="w-4 shrink-0 flex items-center justify-center text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]"
        >
          <svg
            className={`w-3 h-3 transition-transform duration-100 ${expanded ? 'rotate-90' : ''}`}
            fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}
          >
            <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
          </svg>
        </button>
      ) : depth > 0 ? (
        <span className="w-4 shrink-0 flex items-center justify-center text-[var(--color-border-default)]">
          <svg width="12" height="16" viewBox="0 0 12 16" fill="none" stroke="currentColor" strokeWidth="1.5">
            <path d="M1 0v9h10" />
          </svg>
        </span>
      ) : (
        <span className="text-[var(--color-text-muted)] opacity-0 group-hover:opacity-30 transition-opacity w-4 shrink-0 flex items-center justify-center">
          <svg width="6" height="10" viewBox="0 0 6 10" fill="currentColor">
            <circle cx="1" cy="1" r="1" /><circle cx="5" cy="1" r="1" />
            <circle cx="1" cy="5" r="1" /><circle cx="5" cy="5" r="1" />
            <circle cx="1" cy="9" r="1" /><circle cx="5" cy="9" r="1" />
          </svg>
        </span>
      )}

      {/* Issue ID */}
      <span className="text-[11px] font-mono text-[var(--color-text-muted)] w-[88px] shrink-0 truncate tabular-nums">
        {issue.id}
      </span>

      {/* Status icon */}
      <StatusIcon status={issue.status} size={14} className="shrink-0" />

      {/* Title */}
      <span className="text-[13px] font-medium text-[var(--color-text-primary)] truncate flex-1 min-w-0 group-hover:text-white">
        {issue.title}
      </span>

      {/* Sub-issue progress */}
      {subProgress && (
        <span className="flex items-center gap-1.5 text-[11px] text-[var(--color-text-muted)] shrink-0">
          <SubProgress done={subProgress.done} total={subProgress.total} />
          {subProgress.done}/{subProgress.total}
        </span>
      )}

      {/* Pending dot */}
      {issue.is_pending && (
        <span className="w-1.5 h-1.5 rounded-full bg-[var(--color-warning)] shrink-0" />
      )}

      {/* Labels */}
      <div className="flex items-center gap-2.5 shrink-0">
        {issue.labels?.map((label) => (
          <LabelBadge key={label} label={label} />
        ))}
      </div>

      {/* Estimate */}
      {issue.estimate > 0 && (
        <span className="text-[11px] text-[var(--color-text-muted)] tabular-nums shrink-0">
          {issue.estimate}
        </span>
      )}

      {/* Date */}
      <span className="text-[11px] text-[var(--color-text-muted)] tabular-nums shrink-0 w-12 text-right">
        {formatShortDate(issue.created_at)}
      </span>
    </div>
  );
}

function SubProgress({ done, total }: { done: number; total: number }) {
  const pct = total > 0 ? (done / total) * 100 : 0;
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" className="shrink-0">
      <circle cx="8" cy="8" r="6" fill="none" stroke="var(--color-bg-tertiary)" strokeWidth="2" />
      <circle
        cx="8" cy="8" r="6" fill="none"
        stroke={pct === 100 ? 'var(--color-success)' : 'var(--color-accent-primary)'}
        strokeWidth="2"
        strokeLinecap="round"
        strokeDasharray={`${pct * 0.377} 100`}
        transform="rotate(-90 8 8)"
      />
    </svg>
  );
}
