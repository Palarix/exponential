import type { Issue, Release, ReleasesResponse } from "../../api/types";
import { isCompleted, isTerminal } from "../../constants";

/** The open releases in order: Current, Next (either may be a placeholder), then Upcoming. */
export function openReleases(board: ReleasesResponse): Release[] {
  return [board.current, board.next, ...board.upcoming];
}

/** The saved open releases in order. Placeholders can't be reordered. */
export function savedOrder(board: ReleasesResponse): Release[] {
  return openReleases(board).filter((r) => !r.placeholder);
}

/** The release with `number` anywhere on the board, placeholders included. */
export function findRelease(board: ReleasesResponse, number: number): Release | null {
  return [board.current, board.next, ...board.upcoming, ...board.past].find((r) => r.number === number) ?? null;
}

/**
 * The releases whose target date contradicts the manual order, keyed by
 * number, with the names of the releases each one conflicts with. A pair
 * conflicts when the later release has the earlier target date; releases
 * without a target date and placeholders are skipped. The names come in
 * board order.
 */
export function orderWarnings(open: Release[]): Map<number, string[]> {
  const dated = open.filter((r) => r.target_date && !r.placeholder);
  const warnings = new Map<number, string[]>();
  const add = (r: Release, other: Release) => warnings.set(r.number, [...(warnings.get(r.number) ?? []), other.name]);
  for (let i = 0; i < dated.length; i++) {
    for (let j = i + 1; j < dated.length; j++) {
      if (dated[j].target_date! < dated[i].target_date!) {
        add(dated[i], dated[j]);
        add(dated[j], dated[i]);
      }
    }
  }
  return warnings;
}

/**
 * The reorder request for dropping `activeId` on `overId` in `ids`: a
 * release moved down lands after the target, one moved up lands before it.
 */
export function reorderMove(ids: string[], activeId: string, overId: string): { id: string; before?: string; after?: string } | null {
  const from = ids.indexOf(activeId);
  const to = ids.indexOf(overId);
  if (from < 0 || to < 0 || from === to) return null;
  return from < to ? { id: activeId, after: overId } : { id: activeId, before: overId };
}

/**
 * How much of the time from `start` to `target` (YYYY-MM-DD, or an RFC 3339
 * timestamp for `start`) has passed by `today`. Overdue means the target
 * date has passed with open scope left.
 */
export function timeBar(
  start: string | undefined,
  target: string | undefined,
  today: string,
  hasOpenScope: boolean,
): { fraction: number; overdue: boolean } | null {
  if (!start || !target) return null;
  const day = (d: string) => Date.parse(d.slice(0, 10) + "T00:00:00Z");
  const from = day(start);
  const to = day(target);
  const now = day(today);
  const span = to - from;
  const fraction = span <= 0 ? (now >= to ? 1 : 0) : Math.min(1, Math.max(0, (now - from) / span));
  return { fraction, overdue: hasOpenScope && now > to };
}

const bySortOrder = (a: Issue, b: Issue) => (a.sort_order < b.sort_order ? -1 : a.sort_order > b.sort_order ? 1 : 0);

/** Whether `issue` matches the view's search: its title or ID, ignoring case. An empty search matches all. */
export function matchesIssueSearch(issue: Issue, query: string): boolean {
  const q = query.trim().toLowerCase();
  return !q || issue.title.toLowerCase().includes(q) || issue.id.toLowerCase().includes(q);
}

/**
 * The issues in the release `releaseId` that match `query`: open ones first,
 * then terminal ones, each by sort order.
 */
export function issuesInRelease(issues: Issue[], releaseId: string | undefined, query = ""): Issue[] {
  if (!releaseId) return [];
  const inRelease = issues.filter((i) => i.release_id === releaseId && matchesIssueSearch(i, query)).sort(bySortOrder);
  return [...inRelease.filter((i) => !isTerminal(i.status)), ...inRelease.filter((i) => isTerminal(i.status))];
}

/** Open issues without a release, matching `query` on title or ID. */
export function noReleaseIssues(issues: Issue[], query: string): Issue[] {
  return issues
    .filter((i) => !i.release_id && !isTerminal(i.status) && matchesIssueSearch(i, query))
    .sort(bySortOrder);
}

/**
 * The `release_id` an issue dropped on `target` gets: the release ID, the
 * slot ref of a placeholder (the server saves it), or "" for No release.
 */
export function releaseDropRef(target: Release | null): string {
  if (!target) return "";
  if (target.placeholder) return target.slot === "current" ? "@current" : "@next";
  return target.id ?? "";
}

/**
 * Whether `issue` can be dropped on `target` (null is No release). Only
 * OPEN releases accept issues, a locked Next refuses writes, and a drop on
 * the issue's own release would change nothing.
 */
export function acceptsIssueDrop(target: Release | null, issue: Issue): boolean {
  if (!target) return !!issue.release_id;
  if (target.locked || target.state !== "OPEN") return false;
  return target.placeholder || issue.release_id !== target.id;
}

/** How a drag over a release shows: an issue drop, or where a dragged release lands. */
export type DropHighlight = "drop" | "before" | "after" | null;

export function dropHighlightClass(highlight: DropHighlight): string {
  switch (highlight) {
    case "drop": return "ring-2 ring-[var(--color-accent-primary)]";
    case "before": return "shadow-[0_-2px_0_0_var(--color-accent-primary)]";
    case "after": return "shadow-[0_2px_0_0_var(--color-accent-primary)]";
    default: return "";
  }
}

/**
 * Where a sub-issue's parent is, for the chip on its row: the parent and
 * "in 1.2" or "no release". Null without a parent, or when the parent is not
 * in `all` (e.g. archived).
 */
export function parentContext(issue: Issue, all: Issue[]): { parent: Issue; where: string } | null {
  if (!issue.parent_id) return null;
  const parent = all.find((i) => i.id === issue.parent_id);
  if (!parent) return null;
  return { parent, where: parent.release ? `in ${parent.release}` : "no release" };
}

/** An issue's direct children, done of all, as backlog rows count them. Null without children. */
export function childProgress(issue: Issue, all: Issue[]): { done: number; total: number } | null {
  const children = all.filter((i) => i.parent_id === issue.id);
  if (children.length === 0) return null;
  return { done: children.filter((c) => isCompleted(c.status)).length, total: children.length };
}
