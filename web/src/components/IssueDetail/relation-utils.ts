import type { Dependency, Issue } from "../../api/types";

export type LinkKind =
  | "relates_to"
  | "blocked_by"
  | "blocks"
  | "depends_on"
  | "dependency_of"
  | "duplicates"
  | "duplicated_by";

/** Link kinds in section order, worded from the issue being viewed. */
export const LINK_KINDS: { kind: LinkKind; section: string; menu: string }[] = [
  { kind: "relates_to", section: "Related", menu: "Related to…" },
  { kind: "blocked_by", section: "Blocked by", menu: "Blocked by…" },
  { kind: "blocks", section: "Blocking", menu: "Blocking…" },
  { kind: "depends_on", section: "Depends on", menu: "Depends on…" },
  { kind: "dependency_of", section: "Dependency of", menu: "Dependency of…" },
  { kind: "duplicates", section: "Duplicate of", menu: "Duplicate of…" },
  { kind: "duplicated_by", section: "Duplicated by", menu: "Duplicated by…" },
];

/** Kinds shown directly in the add and Change to menus. */
export const PRIMARY_LINK_KINDS: LinkKind[] = ["relates_to", "blocked_by", "blocks", "duplicates"];
/** Kinds behind the "More" submenu. */
export const MORE_LINK_KINDS: LinkKind[] = ["depends_on", "dependency_of", "duplicated_by"];

export const LINK_KIND_MENU_LABEL = Object.fromEntries(
  LINK_KINDS.map(k => [k.kind, k.menu]),
) as Record<LinkKind, string>;

export interface RelationItem {
  targetId: string;
  /** Undefined when the linked issue no longer exists. */
  target?: Issue;
}

export interface RelationSection {
  key: LinkKind;
  label: string;
  items: RelationItem[];
}

function resolve(issues: Issue[], id: string): Issue | undefined {
  return issues.find(i => i.id === id) ?? issues.find(i => i.id.endsWith(id));
}

/**
 * Groups the issue's links into sections, hiding empty ones. The parent is
 * left out: the "Sub-issue of" subtitle already shows it.
 */
export function groupRelations(issue: Issue, issues: Issue[]): RelationSection[] {
  const sections: RelationSection[] = [];
  const deps = issue.dependencies || [];
  for (const { kind, section } of LINK_KINDS) {
    const seen = new Set<string>();
    const items: RelationItem[] = [];
    for (const d of deps) {
      if (d.kind !== kind || seen.has(d.target_id)) continue;
      seen.add(d.target_id);
      items.push({ targetId: d.target_id, target: resolve(issues, d.target_id) });
    }
    if (items.length > 0) sections.push({ key: kind, label: section, items });
  }
  return sections;
}

/** Label for the sidebar Relationships button: the parent plus one per chip the card shows. */
export function relationshipsLabel(issue: Issue, issues: Issue[]): string {
  const links = groupRelations(issue, issues).reduce((n, s) => n + s.items.length, 0);
  const count = links + (issue.parent_id ? 1 : 0);
  if (count === 0) return "No relations";
  return count === 1 ? "1 Relationship" : `${count} Relationships`;
}

/**
 * Blocked-by targets that still block starting the issue. Only DONE resolves
 * a blocker: a CANCELED or DUPLICATE one is a stale link (xpo-35943d).
 * Missing targets are skipped, as the start gate does.
 */
export function unresolvedBlockers(issue: Issue, issues: Issue[]): Issue[] {
  const seen = new Set<string>();
  const blockers: Issue[] = [];
  for (const d of issue.dependencies || []) {
    if (d.kind !== "blocked_by" || seen.has(d.target_id)) continue;
    seen.add(d.target_id);
    const target = resolve(issues, d.target_id);
    if (target && target.status !== "DONE") blockers.push(target);
  }
  return blockers;
}

export function addLink(issueId: string, deps: Dependency[], targetId: string, kind: LinkKind): Dependency[] {
  return [...deps, { source_id: issueId, target_id: targetId, kind }];
}

/** Drops every row for this target and kind, stored or derived. */
export function removeLink(deps: Dependency[], targetId: string, kind: string): Dependency[] {
  return deps.filter(d => !(d.target_id === targetId && d.kind === kind));
}

/**
 * Replaces every link to the target with one of the given kind, at the
 * position of the first. One link per pair: legacy extra links are dropped.
 */
export function changeLinkKind(issueId: string, deps: Dependency[], targetId: string, kind: LinkKind): Dependency[] {
  const result: Dependency[] = [];
  let placed = false;
  for (const d of deps) {
    if (d.target_id !== targetId) {
      result.push(d);
    } else if (!placed) {
      result.push({ source_id: issueId, target_id: targetId, kind });
      placed = true;
    }
  }
  return result;
}

/** The Change to menu for a link of the given kind. */
export function changeToKinds(current: string): { primary: LinkKind[]; more: LinkKind[] } {
  return {
    primary: PRIMARY_LINK_KINDS.filter(k => k !== current),
    more: MORE_LINK_KINDS.filter(k => k !== current),
  };
}

function matches(i: Issue, query: string): boolean {
  const q = query.trim().toLowerCase();
  return !q || i.title.toLowerCase().includes(q) || i.id.toLowerCase().includes(q);
}

/** Issues that can get a new link: not this one and not linked either way. */
export function linkCandidates(issue: Issue, issues: Issue[], query: string): Issue[] {
  const linked = new Set((issue.dependencies || []).map(d => d.target_id));
  return issues.filter(i => i.id !== issue.id && !linked.has(i.id) && matches(i, query));
}

function descendantsOf(id: string, issues: Issue[]): Set<string> {
  const result = new Set<string>();
  const walk = (parent: string) => {
    for (const i of issues) {
      if (i.parent_id === parent && !result.has(i.id)) {
        result.add(i.id);
        walk(i.id);
      }
    }
  };
  walk(id);
  return result;
}

/** Issues this one can become a sub-issue of: open top-level issues outside its subtree. */
export function subIssueOfCandidates(issue: Issue, issues: Issue[], query: string): Issue[] {
  const descendants = descendantsOf(issue.id, issues);
  return issues.filter(i =>
    i.id !== issue.id &&
    !descendants.has(i.id) &&
    !i.parent_id &&
    i.status !== "DONE" &&
    matches(i, query),
  );
}

/** Issues that can become a sub-issue of this one: not itself, its ancestors or current sub-issues. */
export function parentOfCandidates(issue: Issue, issues: Issue[], query: string): Issue[] {
  const ancestors = new Set<string>();
  for (let p = issue.parent_id; p && !ancestors.has(p); p = issues.find(i => i.id === p)?.parent_id) {
    ancestors.add(p);
  }
  return issues.filter(i =>
    i.id !== issue.id &&
    !ancestors.has(i.id) &&
    i.parent_id !== issue.id &&
    matches(i, query),
  );
}
