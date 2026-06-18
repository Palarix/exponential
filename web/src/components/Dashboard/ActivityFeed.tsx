import type { ReactNode } from "react";
import type { ActivityEvent, Issue } from "../../api/client";
import { shortName, formatRelativeTime } from "../../utils/format";

type ActIconKey =
  | "create"
  | "comment"
  | "merge"
  | "status-done"
  | "status-doing"
  | "status-blocked"
  | "status-planned"
  | "status-backlog"
  | "estimate"
  | "rename"
  | "description"
  | "labels"
  | "assign"
  | "priority"
  | "parent"
  | "relations";

const ICON_COLORS: Partial<Record<ActIconKey, string>> = {
  "status-done": "text-[var(--color-success)]",
  "status-doing": "text-[var(--color-warning)]",
  "status-blocked": "text-[var(--color-error)]",
  "status-planned": "text-[var(--color-text-secondary)]",
};

function ActIcon({ k }: { k: ActIconKey }) {
  const color = ICON_COLORS[k] ?? "text-[var(--color-text-muted)]";
  const paths: Record<ActIconKey, ReactNode> = {
    create: (
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M12 4.5v15m7.5-7.5h-15"
      />
    ),
    comment: (
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M2.25 12.76c0 1.6 1.123 2.994 2.707 3.227 1.068.157 2.148.279 3.238.364.466.037.893.281 1.153.671L12 21l2.652-3.978c.26-.39.687-.634 1.153-.671 1.09-.085 2.17-.207 3.238-.364 1.584-.233 2.707-1.626 2.707-3.228V6.741c0-1.602-1.123-2.995-2.707-3.228A48.394 48.394 0 0012 3c-2.392 0-4.744.175-7.043.513C3.373 3.746 2.25 5.14 2.25 6.741v6.018z"
      />
    ),
    merge: (
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M3 7.5L7.5 3m0 0L12 7.5M7.5 3v13.5m13.5-3L16.5 18m0 0L12 13.5M16.5 18V4.5"
      />
    ),
    "status-done": (
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
      />
    ),
    "status-doing": (
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.347a1.125 1.125 0 010 1.972l-11.54 6.347a1.125 1.125 0 01-1.667-.986V5.653z"
      />
    ),
    "status-blocked": (
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636"
      />
    ),
    "status-planned": (
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M6.75 3v2.25M17.25 3v2.25M3 18.75V7.5a2.25 2.25 0 012.25-2.25h13.5A2.25 2.25 0 0121 7.5v11.25m-18 0A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75m-18 0v-7.5A2.25 2.25 0 015.25 9h13.5A2.25 2.25 0 0121 11.25v7.5"
      />
    ),
    "status-backlog": (
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5m6 4.125l2.25 2.25m0 0l2.25 2.25M12 13.875l2.25-2.25M12 13.875l-2.25 2.25M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z"
      />
    ),
    estimate: (
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M8 2L14 14H2L8 2Z"
      />
    ),
    rename: (
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M16.862 4.487l1.687-1.688a1.875 1.875 0 112.652 2.652L10.582 16.07a4.5 4.5 0 01-1.897 1.13L6 18l.8-2.685a4.5 4.5 0 011.13-1.897l8.932-8.931z"
      />
    ),
    description: (
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m2.25 0H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z"
      />
    ),
    labels: (
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M9.568 3H5.25A2.25 2.25 0 003 5.25v4.318c0 .597.237 1.17.659 1.591l9.581 9.581c.699.699 1.78.872 2.607.33a18.095 18.095 0 005.223-5.223c.542-.827.369-1.908-.33-2.607L11.16 3.66A2.25 2.25 0 009.568 3z"
      />
    ),
    assign: (
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M15.75 6a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0zM4.501 20.118a7.5 7.5 0 0114.998 0A17.933 17.933 0 0112 21.75c-2.676 0-5.216-.584-7.499-1.632z"
      />
    ),
    priority: (
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z"
      />
    ),
    parent: (
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M2.25 12.75V12A2.25 2.25 0 014.5 9.75h15A2.25 2.25 0 0121.75 12v.75m-8.69-6.44l-2.12-2.12a1.5 1.5 0 00-1.061-.44H4.5A2.25 2.25 0 002.25 6v12a2.25 2.25 0 002.25 2.25h15A2.25 2.25 0 0021.75 18V9a2.25 2.25 0 00-2.25-2.25h-5.379a1.5 1.5 0 01-1.06-.44z"
      />
    ),
    relations: (
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M13.19 8.688a4.5 4.5 0 011.242 7.244l-4.5 4.5a4.5 4.5 0 01-6.364-6.364l1.757-1.757m13.35-.622l1.757-1.757a4.5 4.5 0 00-6.364-6.364l-4.5 4.5a4.5 4.5 0 001.242 7.244"
      />
    ),
  };
  return (
    <svg
      className={`w-4 h-4 shrink-0 ${color}`}
      fill="none"
      viewBox="0 0 24 24"
      stroke="currentColor"
      strokeWidth={1.75}
    >
      {paths[k]}
    </svg>
  );
}

function describeActivity(
  evt: ActivityEvent,
  who: string,
): { icon: ActIconKey; sentence: ReactNode } | null {
  const p = evt.payload || {};
  const name = (
    <span className="text-[var(--color-text-primary)] mr-1">{who}</span>
  );
  switch (evt.type) {
    case "CREATE":
      return { icon: "create", sentence: <>{name} created</> };
    case "COMMENT":
      return { icon: "comment", sentence: <>{name} commented on</> };
    case "MERGE": {
      const strategy = p.strategy ? ` via ${String(p.strategy)}` : "";
      return {
        icon: "merge",
        sentence: (
          <>
            {name} merged{strategy}
          </>
        ),
      };
    }
    case "UPDATE": {
      if (p.status) {
        const verbs: Record<string, string> = {
          BACKLOG: "moved to Backlog",
          PLANNED: "marked as Planned",
          DOING: "started working on",
          BLOCKED: "marked as Blocked",
          DONE: "completed",
        };
        const icons: Record<string, ActIconKey> = {
          BACKLOG: "status-backlog",
          PLANNED: "status-planned",
          DOING: "status-doing",
          BLOCKED: "status-blocked",
          DONE: "status-done",
        };
        const status = String(p.status);
        return {
          icon: icons[status] ?? "status-backlog",
          sentence: (
            <>
              {name} {verbs[status] ?? `moved to ${status}`}
            </>
          ),
        };
      }
      if (p.assignee !== undefined) {
        const assignee = String(p.assignee);
        if (!assignee)
          return {
            icon: "assign",
            sentence: <>{name} removed the assignee from</>,
          };
        return {
          icon: "assign",
          sentence: (
            <>
              {name} assigned{" "}
              <span className="text-[var(--color-text-primary)]">
                {shortName(assignee)}
              </span>{" "}
              to
            </>
          ),
        };
      }
      if (Array.isArray(p.labels))
        return { icon: "labels", sentence: <>{name} relabeled</> };
      if (p.estimate !== undefined)
        return { icon: "estimate", sentence: <>{name} estimated</> };
      if (p.priority !== undefined)
        return { icon: "priority", sentence: <>{name} changed priority of</> };
      if (p.title) return { icon: "rename", sentence: <>{name} renamed</> };
      if (p.description !== undefined)
        return {
          icon: "description",
          sentence: <>{name} updated the description of</>,
        };
      if (p.parent_id !== undefined)
        return { icon: "parent", sentence: <>{name} changed parent of</> };
      if (Array.isArray(p.dependencies))
        return {
          icon: "relations",
          sentence: <>{name} updated relationships of</>,
        };
      return null;
    }
    default:
      return null;
  }
}

export default function ActivityFeed({
  activity,
  issues,
  onIssueClick,
}: {
  activity: ActivityEvent[];
  issues: Issue[];
  onIssueClick?: (issue: Issue) => void;
}) {
  if (activity.length === 0) {
    return (
      <div className="px-5 py-10 flex items-center justify-center">
        <p className="text-sm text-[var(--color-text-muted)] text-center">
          No activity yet.
        </p>
      </div>
    );
  }

  return (
    <div className="py-1">
      {activity.map((evt, i) => {
        const who = shortName(evt.created_by);
        const desc = describeActivity(evt, who);
        if (!desc) return null;
        const issue = issues.find((it) => it.id === evt.issue_id);
        const title = evt.issue_title || evt.issue_id;
        const key = `${evt.issue_id}-${evt.created_at}-${i}`;
        return (
          <div
            key={key}
            className="flex items-center px-5 py-2 text-sm text-[var(--color-text-muted)] hover:bg-[var(--color-hover-surface)] transition-colors"
          >
            <span className="shrink-0 mr-2">
              <ActIcon k={desc.icon} />
            </span>
            <span className="shrink-0">{desc.sentence}</span>
            <button
              onClick={() => issue && onIssueClick?.(issue)}
              disabled={!issue}
              className="font-medium text-[var(--color-text-primary)] hover:text-[var(--color-accent-primary)] truncate min-w-0 ml-1.5 disabled:opacity-60 disabled:cursor-default transition-colors"
              title={title}
            >
              {title}
            </button>
            <span className="ml-auto text-xs tabular-nums shrink-0 whitespace-nowrap">
              {formatRelativeTime(evt.created_at)}
            </span>
          </div>
        );
      })}
    </div>
  );
}
