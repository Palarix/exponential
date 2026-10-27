import type { Issue, Milestone, MilestonesResponse } from "../../api/types";
import type { IssuePatch } from "../../api/query-utils";
import { isTerminal } from "../../constants";
import { milestoneDropRef } from "./milestones-utils";

/** A milestone the picker offers. A locked Next, and the issue's completed milestone, show but can't be picked. */
export interface PickerOption {
  milestone: Milestone;
  slotLabel?: "Current" | "Next";
  disabled: boolean;
  /** The COMPLETED milestone the issue is still in, shown so the picker says where it is. */
  completed?: boolean;
}

/**
 * The open milestones in order, placeholders included: only OPEN milestones
 * accept issues. When `currentId` names a COMPLETED milestone, it comes first,
 * disabled, so the picker still shows where the issue is.
 */
export function pickerOptions(board: MilestonesResponse, currentId?: string): PickerOption[] {
  const completed = currentId ? board.past.find((r) => r.id === currentId) : undefined;
  return [
    ...(completed ? [{ milestone: completed, disabled: true, completed: true }] : []),
    { milestone: board.current, slotLabel: "Current" as const, disabled: false },
    { milestone: board.next, slotLabel: "Next" as const, disabled: !!board.next.locked },
    ...board.upcoming.map((milestone) => ({ milestone, disabled: false })),
  ];
}

/** The options whose name, goal, `#N` or slot contains `query`, ignoring case. */
export function filterPickerOptions(options: PickerOption[], query: string): PickerOption[] {
  const q = query.trim().toLowerCase();
  if (!q) return options;
  return options.filter(({ milestone, slotLabel }) =>
    [milestone.name, milestone.goal ?? "", `#${milestone.number}`, slotLabel ?? ""].some((s) => s.toLowerCase().includes(q)));
}

/** The open (not terminal) descendants of `id`, at any depth. */
export function openDescendants(issues: Issue[], id: string): Issue[] {
  const byParent = new Map<string, Issue[]>();
  for (const i of issues) {
    if (i.parent_id) byParent.set(i.parent_id, [...(byParent.get(i.parent_id) ?? []), i]);
  }
  const seen = new Set([id]);
  const out: Issue[] = [];
  const queue = [id];
  while (queue.length > 0) {
    for (const child of byParent.get(queue.shift()!) ?? []) {
      if (seen.has(child.id)) continue;
      seen.add(child.id);
      queue.push(child.id);
      if (!isTerminal(child.status)) out.push(child);
    }
  }
  return out;
}

/** All descendants of `id`, open or not (for the done count). */
function allDescendants(issues: Issue[], id: string): Issue[] {
  const ids = new Set([id]);
  let grew = true;
  while (grew) {
    grew = false;
    for (const i of issues) {
      if (i.parent_id && ids.has(i.parent_id) && !ids.has(i.id)) {
        ids.add(i.id);
        grew = true;
      }
    }
  }
  ids.delete(id);
  return issues.filter((i) => ids.has(i.id));
}

/** Whether `issue` already is where `target` would put it. Nothing is in a placeholder yet. */
export function isInMilestone(issue: Issue, target: Milestone | null): boolean {
  if (!target) return !issue.milestone_id;
  return !target.placeholder && issue.milestone_id === target.id;
}

/** Whether an ancestor of `issue` is among `ids`. */
function hasAncestorIn(issue: Issue, byId: Map<string, Issue>, ids: Set<string>): boolean {
  const seen = new Set<string>();
  let parentId = issue.parent_id;
  while (parentId && !seen.has(parentId)) {
    if (ids.has(parentId)) return true;
    seen.add(parentId);
    parentId = byId.get(parentId)?.parent_id;
  }
  return false;
}

/** The assigned issues no other assigned issue contains: their prompt covers the rest. */
function rootsOf(issues: Issue[], ids: string[]): Issue[] {
  const byId = new Map(issues.map((i) => [i.id, i]));
  const set = new Set(ids);
  return ids.map((id) => byId.get(id)).filter((i): i is Issue => !!i && !hasAncestorIn(i, byId, set));
}

export interface ChildrenPrompt {
  /** The assigned issues with open children to move. */
  epics: Issue[];
  /** Their open descendants that are not in the target yet. */
  children: Issue[];
  /** Terminal descendants, which never move. */
  doneCount: number;
}

/**
 * What to ask before moving `ids` to `target` (null is No milestone): the open
 * descendants that would move with them. Null when there are none. An
 * assigned issue whose ancestor is assigned too is asked about once, as
 * part of the ancestor.
 */
export function childrenPrompt(issues: Issue[], ids: string[], target: Milestone | null): ChildrenPrompt | null {
  const epics: Issue[] = [];
  const children: Issue[] = [];
  let doneCount = 0;
  for (const root of rootsOf(issues, ids)) {
    const moving = openDescendants(issues, root.id).filter((c) => !isInMilestone(c, target));
    if (moving.length === 0) continue;
    epics.push(root);
    children.push(...moving);
    doneCount += allDescendants(issues, root.id).filter((c) => isTerminal(c.status)).length;
  }
  return children.length > 0 ? { epics, children, doneCount } : null;
}

/** One update to send: `ref` is a milestone ID, `@current`/`@next`, or "" for No milestone. */
export interface MilestoneMove {
  issueId: string;
  ref: string;
  withChildren: boolean;
}

/** The cached fields that show `target` right away; a placeholder's ID comes with the refetch. */
function optimisticPatch(target: Milestone | null): Partial<Issue> {
  if (!target) return { milestone_id: "", milestone: "" };
  if (target.placeholder) return { milestone: target.name };
  return { milestone_id: target.id, milestone: target.name };
}

/**
 * The updates that move `ids` to `target`, and the optimistic patches. With
 * children, each root sends one update with `with_children` (one feed
 * entry), and the open descendants it covers send none.
 */
export function planMilestoneMoves(
  issues: Issue[], ids: string[], target: Milestone | null, withChildren: boolean,
): { moves: MilestoneMove[]; optimistic: IssuePatch[] } {
  const ref = milestoneDropRef(target);
  const patch = optimisticPatch(target);
  const byId = new Map(issues.map((i) => [i.id, i]));
  const covered = new Set<string>();
  const moves: MilestoneMove[] = [];
  const optimistic: IssuePatch[] = [];
  const order = withChildren ? rootsOf(issues, ids) : ids.map((id) => byId.get(id)).filter((i): i is Issue => !!i);

  for (const issue of order) {
    const kids = withChildren ? openDescendants(issues, issue.id).filter((c) => !isInMilestone(c, target)) : [];
    if (isInMilestone(issue, target) && kids.length === 0) continue;
    moves.push({ issueId: issue.id, ref, withChildren: kids.length > 0 });
    for (const i of [issue, ...kids]) {
      covered.add(i.id);
      optimistic.push({ issueId: i.id, patch });
    }
  }
  // A terminal issue assigned on its own under a moving root still moves on its own.
  if (withChildren) {
    for (const id of ids) {
      const issue = byId.get(id);
      if (!issue || covered.has(id) || isInMilestone(issue, target) || !isTerminal(issue.status)) continue;
      moves.push({ issueId: id, ref, withChildren: false });
      optimistic.push({ issueId: id, patch });
    }
  }
  return { moves, optimistic };
}

/** The COMPLETED milestone an open issue still sits in, for the reopen hint. */
export function reopenHint(issue: Issue, board: MilestonesResponse | undefined): Milestone | null {
  if (!board || !issue.milestone_id || isTerminal(issue.status)) return null;
  return board.past.find((r) => r.id === issue.milestone_id) ?? null;
}

/** "Milestone 1.2", and "· completed" once it completed. */
export function milestoneChipTooltip(name: string, milestone?: Milestone | null): string {
  return `Milestone ${name}${milestone?.state === "COMPLETED" ? " · completed" : ""}`;
}
