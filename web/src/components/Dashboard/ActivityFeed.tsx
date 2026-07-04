import type { ReactNode } from "react";
import type { ActivityEvent, Issue } from "../../api/client";
import { shortName, formatRelativeTime, displayActor } from "../../utils/format";
import {
  Plus,
  MessageSquareMore,
  GitMerge,
  CheckCircle,
  Play,
  Ban,
  Archive,
  Triangle,
  User,
  Folder,
  Link,
  CircleDot,
  Pencil,
  SquareDashedBottomCode,
  Tag,
  TriangleAlert,
  Paperclip,
  Bot,
} from "lucide-react";

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
  | "relations"
  | "artifact";

const ICON_COLORS: Partial<Record<ActIconKey, string>> = {
  "status-done": "text-[var(--color-success)]",
  "status-doing": "text-[var(--color-warning)]",
  "status-blocked": "text-[var(--color-error)]",
  "status-planned": "text-[var(--color-text-secondary)]",
};

/* Map keys that have an exact-match shared icon component */
const SHARED_ICONS: Partial<
  Record<ActIconKey, (props: { className?: string }) => ReactNode>
> = {
  create: Plus,
  comment: MessageSquareMore,
  merge: GitMerge,
  "status-done": CheckCircle,
  "status-doing": Play,
  "status-blocked": Ban,
  "status-backlog": Archive,
  "status-planned": CircleDot,
  estimate: Triangle,
  assign: User,
  parent: Folder,
  relations: Link,
  rename: Pencil,
  description: SquareDashedBottomCode,
  labels: Tag,
  priority: TriangleAlert,
  artifact: Paperclip,
};

function ActIcon({ k }: { k: ActIconKey }) {
  const color = ICON_COLORS[k] ?? "text-[var(--color-text-muted)]";
  const cls = `w-4 h-4 shrink-0 ${color}`;
  const Icon = SHARED_ICONS[k];
  if (Icon) return <Icon className={cls} />;
  return null;
}

function describeActivity(
  evt: ActivityEvent,
  who: string,
  via?: string,
): { icon: ActIconKey; sentence: ReactNode } | null {
  const p = evt.payload || {};
  const name = (
    <span className="text-[var(--color-text-primary)] mr-1">{who}</span>
  );
  const viaLabel = via ? <span title={via}><Bot className="w-3.5 h-3.5 inline-block align-[-2px] ml-1 mr-0.5 text-[var(--color-text-muted)]" /></span> : null;
  switch (evt.type) {
    case "CREATE":
      return { icon: "create", sentence: <>{name}{viaLabel} created</> };
    case "COMMENT":
      return { icon: "comment", sentence: <>{name}{viaLabel} commented on</> };
    case "MERGE": {
      const strategy = p.strategy ? ` via ${String(p.strategy)}` : "";
      return {
        icon: "merge",
        sentence: (
          <>
            {name}{viaLabel} merged{strategy}
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
              {name}{viaLabel} {verbs[status] ?? `moved to ${status}`}
            </>
          ),
        };
      }
      if (p.assignee !== undefined) {
        const assignee = String(p.assignee);
        if (!assignee)
          return {
            icon: "assign",
            sentence: <>{name}{viaLabel} removed the assignee from</>,
          };
        return {
          icon: "assign",
          sentence: (
            <>
              {name}{viaLabel} assigned{" "}
              <span className="text-[var(--color-text-primary)]">
                {shortName(assignee)}
              </span>{" "}
              to
            </>
          ),
        };
      }
      if (Array.isArray(p.labels))
        return { icon: "labels", sentence: <>{name}{viaLabel} relabeled</> };
      if (p.estimate !== undefined)
        return { icon: "estimate", sentence: <>{name}{viaLabel} estimated</> };
      if (p.priority !== undefined)
        return { icon: "priority", sentence: <>{name}{viaLabel} changed priority of</> };
      if (p.title) return { icon: "rename", sentence: <>{name}{viaLabel} renamed</> };
      if (p.description !== undefined)
        return {
          icon: "description",
          sentence: <>{name}{viaLabel} updated the description of</>,
        };
      if (p.parent_id !== undefined)
        return { icon: "parent", sentence: <>{name}{viaLabel} changed parent of</> };
      if (Array.isArray(p.dependencies))
        return {
          icon: "relations",
          sentence: <>{name}{viaLabel} updated relationships of</>,
        };
      return null;
    }
    case "ARTIFACT": {
      const action = String(p.action || "updated");
      const filename = String(p.filename || "artifact");
      return {
        icon: "artifact" as ActIconKey,
        sentence: (
          <>
            {name}{viaLabel} {action}{" "}
            <span className="font-mono text-[var(--color-text-primary)]">
              {filename}
            </span>{" "}
            on
          </>
        ),
      };
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
        const actor = displayActor(evt.created_by, evt.on_behalf_of);
        const desc = describeActivity(evt, actor.principal, actor.via);
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
