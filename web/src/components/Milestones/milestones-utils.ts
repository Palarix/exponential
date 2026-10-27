import type { Issue, Milestone, MilestonesResponse } from "../../api/types";
import type { MilestoneIssuesMode } from "../../api/client";
import { isCompleted, isTerminal } from "../../constants";

/** The open milestones in order: Current, Next (either may be a placeholder), then Upcoming. */
export function openMilestones(board: MilestonesResponse): Milestone[] {
  return [board.current, board.next, ...board.upcoming];
}

/** The saved open milestones in order. Placeholders can't be reordered. */
export function savedOrder(board: MilestonesResponse): Milestone[] {
  return openMilestones(board).filter((r) => !r.placeholder);
}

/**
 * The milestone filter clicking an open milestone opens Issues with: Current and
 * Next by slot, so the filter follows them as milestones complete, Upcoming by
 * number. Null for a Past milestone, which expands in place instead.
 */
export function milestoneFilter(milestone: Milestone): string | null {
  if (milestone.state === "COMPLETED") return null;
  if (milestone.slot === "current") return "@current";
  if (milestone.slot === "next") return "@next";
  return `#${milestone.number}`;
}

/**
 * The milestones whose target date contradicts the manual order, keyed by
 * number, with the names of the milestones each one conflicts with. A pair
 * conflicts when the later milestone has the earlier target date; milestones
 * without a target date and placeholders are skipped. The names come in
 * board order.
 */
export function orderWarnings(open: Milestone[]): Map<number, string[]> {
  const dated = open.filter((r) => r.target_date && !r.placeholder);
  const warnings = new Map<number, string[]>();
  const add = (r: Milestone, other: Milestone) => warnings.set(r.number, [...(warnings.get(r.number) ?? []), other.name]);
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
 * milestone moved down lands after the target, one moved up lands before it.
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
 * The issues in the milestone `milestoneId` that match `query`: open ones first,
 * then terminal ones, each by sort order.
 */
export function issuesInMilestone(issues: Issue[], milestoneId: string | undefined, query = ""): Issue[] {
  if (!milestoneId) return [];
  const inMilestone = issues.filter((i) => i.milestone_id === milestoneId && matchesIssueSearch(i, query)).sort(bySortOrder);
  return [...inMilestone.filter((i) => !isTerminal(i.status)), ...inMilestone.filter((i) => isTerminal(i.status))];
}

/** Open issues without a milestone, matching `query` on title or ID. */
export function noMilestoneIssues(issues: Issue[], query: string): Issue[] {
  return issues
    .filter((i) => !i.milestone_id && !isTerminal(i.status) && matchesIssueSearch(i, query))
    .sort(bySortOrder);
}

/**
 * The `milestone_id` an issue dropped on `target` gets: the milestone ID, the
 * slot ref of a placeholder (the server saves it), or "" for No milestone.
 */
export function milestoneDropRef(target: Milestone | null): string {
  if (!target) return "";
  if (target.placeholder) return target.slot === "current" ? "@current" : "@next";
  return target.id ?? "";
}

/**
 * Whether `issue` can be dropped on `target` (null is No milestone). Only
 * OPEN milestones accept issues, a locked Next refuses writes, and a drop on
 * the issue's own milestone would change nothing.
 */
export function acceptsIssueDrop(target: Milestone | null, issue: Issue): boolean {
  if (!target) return !!issue.milestone_id;
  if (target.locked || target.state !== "OPEN") return false;
  return target.placeholder || issue.milestone_id !== target.id;
}

/** How a drag over a milestone shows: an issue drop, or where a dragged milestone lands. */
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
 * "in 1.2" or "no milestone". Null without a parent, or when the parent is not
 * in `all` (e.g. archived).
 */
export function parentContext(issue: Issue, all: Issue[]): { parent: Issue; where: string } | null {
  if (!issue.parent_id) return null;
  const parent = all.find((i) => i.id === issue.parent_id);
  if (!parent) return null;
  return { parent, where: parent.milestone ? `in ${parent.milestone}` : "no milestone" };
}

/** An issue's direct children, done of all, as backlog rows count them. Null without children. */
export function childProgress(issue: Issue, all: Issue[]): { done: number; total: number } | null {
  const children = all.filter((i) => i.parent_id === issue.id);
  if (children.length === 0) return null;
  return { done: children.filter((c) => isCompleted(c.status)).length, total: children.length };
}

/**
 * The estimate an issue row shows, as backlog rows show it: a parent's open
 * child points (its done ones once the parent is closed), each child counting
 * its estimate or 1; otherwise the issue's own estimate.
 */
export function rowEstimate(issue: Issue, all: Issue[]): number {
  const children = all.filter((i) => i.parent_id === issue.id);
  if (children.length === 0) return issue.estimate;
  const points = (list: Issue[]) => list.reduce((s, c) => s + (c.estimate || 1), 0);
  const done = points(children.filter((c) => isCompleted(c.status)));
  return isTerminal(issue.status) ? done : points(children) - done;
}

/** Where the move picker can send a milestone's issues: a milestone ID, "new" (a new milestone) or "none" (No milestone). */
export interface MoveOption { value: string; label: string }

/**
 * The move picker of the complete and delete dialogs: the other saved open
 * milestones in board order, labelled by slot, then No milestone. Without any,
 * "A new milestone" takes their place. The default is the server's: the milestone
 * right after this one, else the one right before it, else a new milestone.
 */
export function moveOptions(board: MilestonesResponse, milestone: Milestone): { options: MoveOption[]; value: string } {
  const saved = savedOrder(board);
  const others = saved.filter((r) => r.id !== milestone.id);
  const none: MoveOption = { value: "none", label: "No milestone" };
  if (others.length === 0) return { options: [{ value: "new", label: "A new milestone" }, none], value: "new" };
  const slotName = (r: Milestone) => (r.slot === "current" ? " (Current)" : r.slot === "next" ? " (Next)" : "");
  const at = saved.findIndex((r) => r.id === milestone.id);
  const neighbour = saved[at + 1] ?? saved[at - 1] ?? others[0];
  return {
    options: [...others.map((r) => ({ value: r.id!, label: `${r.name}${slotName(r)}` })), none],
    value: neighbour.id!,
  };
}

/** The complete or delete request for a picked move option. */
export function moveRequest(value: string): { issues: MilestoneIssuesMode; to?: string } {
  if (value === "none") return { issues: "remove" };
  if (value === "new") return { issues: "move_to_next" };
  return { issues: "move_to_next", to: value };
}

/** The issues a complete moves (open ones) and a delete moves (all), counted as the server does: parents included. */
export function milestoneMemberCounts(issues: Issue[], milestoneId: string | undefined): { open: number; all: number } {
  if (!milestoneId) return { open: 0, all: 0 };
  const members = issues.filter((i) => i.milestone_id === milestoneId);
  return { open: members.filter((i) => !isTerminal(i.status)).length, all: members.length };
}

/**
 * Whether a click on a milestone row's header toggles its issues: not when it
 * lands on one of the header's buttons, and not when it bubbles through React
 * from a portal (menu, popover or dialog) outside the header's DOM.
 */
export function headerClickToggles(
  target: { closest(selector: string): unknown } | null,
  header: { contains(node: unknown): boolean },
): boolean {
  if (!target || !header.contains(target)) return false;
  return !target.closest("button, a, input, [role='menu']");
}
