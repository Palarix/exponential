import type { InboxItem } from "../../api/types";
import { shortName, extractEmail } from "../../utils/format";

export interface IssueGroup {
  issueId: string;
  issueTitle: string;
  latestAt: number;
  hasUnread: boolean;
  events: InboxItem[];
}

export function groupByIssue(items: InboxItem[], lastReadTime: number): IssueGroup[] {
  const map = new Map<string, IssueGroup>();
  for (const item of items) {
    let group = map.get(item.issue_id);
    if (!group) {
      group = {
        issueId: item.issue_id,
        issueTitle: item.issue_title || item.issue_id,
        latestAt: 0,
        hasUnread: false,
        events: [],
      };
      map.set(item.issue_id, group);
    }
    const t = new Date(item.created_at).getTime();
    if (t > group.latestAt) group.latestAt = t;
    if (t > lastReadTime) group.hasUnread = true;
    group.events.push(item);
  }
  return Array.from(map.values()).sort((a, b) => b.latestAt - a.latestAt);
}

const STATUS_LABEL: Record<string, string> = {
  BACKLOG: "moved to Backlog",
  PLANNED: "marked as Planned",
  DOING: "started working",
  BLOCKED: "marked as Blocked",
  DONE: "completed",
};

export interface SummaryActor {
  principal: string;
  via: string;
  isYou: boolean;
}

// The person an agent acted for, principal-first like every other actor.
export function summaryActor(evt: InboxItem, userEmail: string): SummaryActor | null {
  if (!evt.on_behalf_of) return null;
  const principalEmail = extractEmail(evt.on_behalf_of);
  return {
    principal: shortName(evt.on_behalf_of),
    via: evt.created_by,
    isYou: !!principalEmail && principalEmail === userEmail,
  };
}

export interface ChangeSummary {
  actor: SummaryActor | null;
  parts: string[];
}

export function buildChangeSummary(events: InboxItem[], userEmail: string): ChangeSummary {
  const parts: string[] = [];
  let commentCount = 0;
  const statusChanges: string[] = [];
  let assigneeChange: string | null = null;
  let wasMerged = false;
  let wasCreated = false;
  let otherUpdates = 0;
  let actor: SummaryActor | null = null;

  for (const evt of events) {
    if (!actor) actor = summaryActor(evt, userEmail);
    const p = evt.payload || {};
    switch (evt.type) {
      case "COMMENT":
        commentCount++;
        break;
      case "CREATE":
        wasCreated = true;
        break;
      case "MERGE":
        wasMerged = true;
        break;
      case "UPDATE":
        if (p.status) {
          const label = STATUS_LABEL[String(p.status)] ?? `moved to ${String(p.status)}`;
          if (!statusChanges.includes(label)) statusChanges.push(label);
        } else if (p.assignee !== undefined) {
          const a = String(p.assignee);
          assigneeChange = a ? `reassigned to ${shortName(a)}` : "assignee removed";
        } else {
          otherUpdates++;
        }
        break;
      case "ARTIFACT": {
        const action = String(p.action || "updated");
        const filename = String(p.filename || "artifact");
        parts.push(`${filename} ${action}`);
        break;
      }
    }
  }

  if (wasCreated) parts.push("Issue created");
  for (const s of statusChanges) parts.push(`Status ${s}`);
  if (wasMerged) parts.push("Merged");
  if (assigneeChange) parts.push(assigneeChange.charAt(0).toUpperCase() + assigneeChange.slice(1));
  if (commentCount > 0) parts.push(`${commentCount} new comment${commentCount !== 1 ? "s" : ""}`);
  if (otherUpdates > 0) parts.push(`${otherUpdates} other update${otherUpdates !== 1 ? "s" : ""}`);
  return { actor, parts };
}
