import type { Issue, Release, ReleasesResponse } from "../../api/types";
import type { IssuePatch } from "../../api/query-utils";
import { isTerminal } from "../../constants";
import { releaseDropRef } from "./releases-utils";

/** A release the picker offers. A locked Next, and the issue's released release, show but can't be picked. */
export interface PickerOption {
  release: Release;
  slotLabel?: "Current" | "Next";
  disabled: boolean;
  /** The RELEASED release the issue is still in, shown so the picker says where it is. */
  released?: boolean;
}

/**
 * The open releases in order, placeholders included: only OPEN releases
 * accept issues. When `currentId` names a RELEASED release, it comes first,
 * disabled, so the picker still shows where the issue is.
 */
export function pickerOptions(board: ReleasesResponse, currentId?: string): PickerOption[] {
  const shipped = currentId ? board.past.find((r) => r.id === currentId) : undefined;
  return [
    ...(shipped ? [{ release: shipped, disabled: true, released: true }] : []),
    { release: board.current, slotLabel: "Current" as const, disabled: false },
    { release: board.next, slotLabel: "Next" as const, disabled: !!board.next.locked },
    ...board.upcoming.map((release) => ({ release, disabled: false })),
  ];
}

/** The options whose name, goal, `#N` or slot contains `query`, ignoring case. */
export function filterPickerOptions(options: PickerOption[], query: string): PickerOption[] {
  const q = query.trim().toLowerCase();
  if (!q) return options;
  return options.filter(({ release, slotLabel }) =>
    [release.name, release.goal ?? "", `#${release.number}`, slotLabel ?? ""].some((s) => s.toLowerCase().includes(q)));
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
export function isInRelease(issue: Issue, target: Release | null): boolean {
  if (!target) return !issue.release_id;
  return !target.placeholder && issue.release_id === target.id;
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
 * What to ask before moving `ids` to `target` (null is No release): the open
 * descendants that would move with them. Null when there are none. An
 * assigned issue whose ancestor is assigned too is asked about once, as
 * part of the ancestor.
 */
export function childrenPrompt(issues: Issue[], ids: string[], target: Release | null): ChildrenPrompt | null {
  const epics: Issue[] = [];
  const children: Issue[] = [];
  let doneCount = 0;
  for (const root of rootsOf(issues, ids)) {
    const moving = openDescendants(issues, root.id).filter((c) => !isInRelease(c, target));
    if (moving.length === 0) continue;
    epics.push(root);
    children.push(...moving);
    doneCount += allDescendants(issues, root.id).filter((c) => isTerminal(c.status)).length;
  }
  return children.length > 0 ? { epics, children, doneCount } : null;
}

/** One update to send: `ref` is a release ID, `@current`/`@next`, or "" for No release. */
export interface ReleaseMove {
  issueId: string;
  ref: string;
  withChildren: boolean;
}

/** The cached fields that show `target` right away; a placeholder's ID comes with the refetch. */
function optimisticPatch(target: Release | null): Partial<Issue> {
  if (!target) return { release_id: "", release: "" };
  if (target.placeholder) return { release: target.name };
  return { release_id: target.id, release: target.name };
}

/**
 * The updates that move `ids` to `target`, and the optimistic patches. With
 * children, each root sends one update with `with_children` (one feed
 * entry), and the open descendants it covers send none.
 */
export function planReleaseMoves(
  issues: Issue[], ids: string[], target: Release | null, withChildren: boolean,
): { moves: ReleaseMove[]; optimistic: IssuePatch[] } {
  const ref = releaseDropRef(target);
  const patch = optimisticPatch(target);
  const byId = new Map(issues.map((i) => [i.id, i]));
  const covered = new Set<string>();
  const moves: ReleaseMove[] = [];
  const optimistic: IssuePatch[] = [];
  const order = withChildren ? rootsOf(issues, ids) : ids.map((id) => byId.get(id)).filter((i): i is Issue => !!i);

  for (const issue of order) {
    const kids = withChildren ? openDescendants(issues, issue.id).filter((c) => !isInRelease(c, target)) : [];
    if (isInRelease(issue, target) && kids.length === 0) continue;
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
      if (!issue || covered.has(id) || isInRelease(issue, target) || !isTerminal(issue.status)) continue;
      moves.push({ issueId: id, ref, withChildren: false });
      optimistic.push({ issueId: id, patch });
    }
  }
  return { moves, optimistic };
}

/** The RELEASED release an open issue still sits in, for the reopen hint. */
export function reopenHint(issue: Issue, board: ReleasesResponse | undefined): Release | null {
  if (!board || !issue.release_id || isTerminal(issue.status)) return null;
  return board.past.find((r) => r.id === issue.release_id) ?? null;
}

/** "Release 1.2", and "· released" once it shipped. */
export function releaseChipTooltip(name: string, release?: Release | null): string {
  return `Release ${name}${release?.state === "RELEASED" ? " · released" : ""}`;
}
