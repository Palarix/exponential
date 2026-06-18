import { useState, useEffect, useMemo } from "react";
import Markdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import { fetchIssueHistory } from "../../api/client";
import type { Issue, HistoryEvent } from "../../api/client";
import { Avatar, LabelBadge, StatusIcon, TriangleIcon, ChevronDownIcon } from "../ui";
import { shortName, formatRelativeTime, linkifyIssueIds } from "../../utils/format";

type ActivityEntry =
  | { kind: "system"; author: string; content: React.ReactNode; time: string }
  | { kind: "comment"; author: string; text: string; time: string };

const STATUS_LABELS: Record<string, string> = {
  BACKLOG: "Backlog",
  PLANNED: "Planned",
  DOING: "In Progress",
  BLOCKED: "Blocked",
  DONE: "Done",
};

function StatusChip({ status }: { status: string }) {
  return (
    <span className="flex items-center gap-1">
      <StatusIcon status={status} size={12} />
      <span className="font-medium text-[var(--color-text-primary)]">
        {STATUS_LABELS[status] || status}
      </span>
    </span>
  );
}

function EstimateChip({ points }: { points: number }) {
  return (
    <span className="flex items-center gap-1 font-medium text-[var(--color-text-primary)]">
      <TriangleIcon className="w-3 h-3" />
      {points} {points === 1 ? "Point" : "Points"}
    </span>
  );
}

function describeEvent(evt: HistoryEvent): React.ReactNode | null {
  const p = evt.payload || {};
  switch (evt.type) {
    case "CREATE":
      return "created this issue";
    case "UPDATE": {
      const fragments: React.ReactNode[] = [];
      if (p.status)
        fragments.push(
          <>
            changed status to <StatusChip status={String(p.status)} />
          </>,
        );
      if (p.estimate !== undefined)
        fragments.push(
          <>
            set estimate to <EstimateChip points={Number(p.estimate)} />
          </>,
        );
      if (p.title) fragments.push(<>updated the title</>);
      if (p.description !== undefined)
        fragments.push(<>updated the description</>);
      if (p.labels)
        fragments.push(
          <>
            updated labels to{" "}
            {(p.labels as string[]).map((l) => (
              <LabelBadge key={l} label={l} />
            ))}
          </>,
        );
      if (p.assignee)
        fragments.push(
          <>
            assigned to{" "}
            <span className="font-medium text-[var(--color-text-primary)]">
              {String(p.assignee)}
            </span>
          </>,
        );
      if (fragments.length === 0) return null;
      return fragments.reduce<React.ReactNode[]>((acc, f, i) => {
        if (i > 0) acc.push(<span key={`sep-${i}`}> and </span>);
        acc.push(f);
        return acc;
      }, []);
    }
    case "DELETE":
      return "deleted this issue";
    case "MERGE": {
      const branch = p.branch ? String(p.branch) : "";
      const shortBranch = branch.length > 40 ? branch.slice(0, 40) + "…" : branch;
      return (
        <>
          merged{" "}
          {branch && (
            <span className="font-mono text-xs text-[var(--color-accent-primary)]" title={branch}>{shortBranch}</span>
          )}
          {p.strategy && (
            <> via {String(p.strategy)}</>
          )}
          {" "}and closed this issue
        </>
      );
    }
    default:
      return null;
  }
}

export default function ActivityTimeline({
  issue,
  newComment,
  onNewCommentChange,
  onAddComment,
  saving,
  commentRef,
  prefix,
}: {
  issue: Issue;
  newComment: string;
  onNewCommentChange: (v: string) => void;
  onAddComment: () => void;
  saving: boolean;
  commentRef: React.RefObject<HTMLTextAreaElement | null>;
  prefix: string;
}) {
  const [history, setHistory] = useState<HistoryEvent[]>([]);
  const [sortNewest, setSortNewest] = useState(true);

  useEffect(() => {
    fetchIssueHistory(issue.id)
      .then(setHistory)
      .catch(() => {});
  }, [issue.id, issue.updated_at]);

  const entries = useMemo(() => {
    const items: ActivityEntry[] = [];

    for (const evt of history) {
      if (evt.type === "COMMENT") {
        const p = evt.payload || {};
        items.push({
          kind: "comment",
          author: evt.created_by,
          text: String(p.text || ""),
          time: evt.created_at,
        });
      } else {
        const desc = describeEvent(evt);
        if (desc) {
          items.push({
            kind: "system",
            author: evt.created_by,
            content: desc,
            time: evt.created_at,
          });
        }
      }
    }

    const dir = sortNewest ? -1 : 1;
    items.sort(
      (a, b) => dir * (new Date(a.time).getTime() - new Date(b.time).getTime()),
    );
    return items;
  }, [history, sortNewest]);

  return (
    <div className="mt-8 pt-6 border-t border-[var(--color-border-subtle)]">
      <div className="flex items-center justify-between mb-5">
        <h3 className="text-sm font-semibold text-[var(--color-text-primary)]">
          Activity
        </h3>
        <button
          onClick={() => setSortNewest(!sortNewest)}
          className="flex items-center gap-1 text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] transition-colors"
          title={sortNewest ? "Showing newest first" : "Showing oldest first"}
        >
          {sortNewest ? "Newest" : "Oldest"}
          <ChevronDownIcon className={`w-3 h-3 transition-transform ${sortNewest ? "" : "rotate-180"}`} />
        </button>
      </div>

      <div className="space-y-4">
        {entries.map((entry, i) => {
          if (entry.kind === "system") {
            return (
              <div
                key={`sys-${i}`}
                className="flex items-center gap-2 px-4 py-2 flex-wrap text-sm text-[var(--color-text-muted)]"
              >
                <Avatar name={entry.author} size="sm" />
                <span>{shortName(entry.author)}</span>
                {entry.content}
                <span>·</span>
                <span>{formatRelativeTime(entry.time)}</span>
              </div>
            );
          }

          return (
            <div
              key={`cmt-${i}`}
              className="rounded-[var(--radius-lg)] bg-[var(--color-surface-1)] py-3 px-4 border border-[var(--color-border-subtle)]"
            >
              <div className="flex items-center gap-3 mb-2">
                <Avatar name={entry.author} size="sm" />
                <span className="text-sm font-medium text-[var(--color-text-primary)]">
                  {shortName(entry.author)}
                </span>
                <span className="text-sm text-[var(--color-text-muted)]">
                  {formatRelativeTime(entry.time)}
                </span>
              </div>
              <div className="prose-exponential text-base">
                <Markdown remarkPlugins={[remarkGfm, remarkBreaks]}>
                  {linkifyIssueIds(entry.text, prefix)}
                </Markdown>
              </div>
            </div>
          );
        })}
      </div>

      {/* Comment input */}
      <div className="mt-5 rounded-[var(--radius-lg)] bg-[var(--color-surface-1)] border border-[var(--color-border-subtle)] overflow-hidden">
        <textarea
          ref={commentRef}
          value={newComment}
          onChange={(e) => onNewCommentChange(e.target.value)}
          placeholder="Leave a comment..."
          rows={1}
          className="w-full text-base bg-transparent text-[var(--color-text-primary)] px-4 py-3 outline-none placeholder:text-[var(--color-text-muted)] resize-none"
          onKeyDown={(e) => {
            if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) onAddComment();
          }}
        />
        <div className="flex items-center justify-end gap-2 px-3 py-2">
          <button
            onClick={onAddComment}
            disabled={!newComment.trim() || saving}
            className="w-7 h-7 flex items-center justify-center rounded-full bg-[var(--color-accent-primary)] text-white disabled:opacity-20 hover:bg-[var(--color-accent-primary-hover)] transition-colors"
          >
            <svg
              className="w-4 h-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2.5}
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M4.5 10.5L12 3m0 0l7.5 7.5M12 3v18"
              />
            </svg>
          </button>
        </div>
      </div>
    </div>
  );
}
