import type { Issue } from "../api/types";

export function buildChildrenByParent(issues: Issue[]): Map<string, Issue[]> {
  const map = new Map<string, Issue[]>();
  for (const issue of issues) {
    if (issue.parent_id) {
      const siblings = map.get(issue.parent_id) || [];
      siblings.push(issue);
      map.set(issue.parent_id, siblings);
    }
  }
  return map;
}

export function collectKnownPeople(issues: Issue[], contributors: string[]): string[] {
  const byEmail = new Map<string, string>();
  for (const val of contributors) {
    const email = val.match(/<([^>]+@[^>]+)>/)?.[1]?.toLowerCase();
    if (email && !byEmail.has(email)) byEmail.set(email, val);
  }
  for (const i of issues) {
    for (const val of [i.created_by, i.assignee]) {
      if (!val) continue;
      const email = val.match(/<([^>]+@[^>]+)>/)?.[1]?.toLowerCase();
      if (email && !byEmail.has(email)) byEmail.set(email, val);
    }
  }
  return Array.from(byEmail.values()).sort((a, b) =>
    a.split(" <")[0].localeCompare(b.split(" <")[0]),
  );
}
