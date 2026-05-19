import { generateNKeysBetween } from 'fractional-indexing';
import type { Issue } from '../api/types';

export type SortKey = 'manual' | 'priority' | 'created' | 'updated' | 'title' | 'estimate';

export const SORT_OPTIONS: { value: SortKey; label: string }[] = [
  { value: 'manual', label: 'Manual' },
  { value: 'priority', label: 'Priority' },
  { value: 'created', label: 'Created' },
  { value: 'updated', label: 'Updated' },
  { value: 'title', label: 'Title' },
  { value: 'estimate', label: 'Estimate' },
];

export const STATUS_ORDER = ['BACKLOG', 'PLANNED', 'DOING', 'BLOCKED', 'DONE'];

export const PRIORITY_LABELS: Record<number, string> = {
  0: 'No priority',
  1: 'Urgent',
  2: 'High',
  3: 'Medium',
  4: 'Low',
};

function compareKeys(a: string, b: string): number {
  if (a < b) return -1;
  if (a > b) return 1;
  return 0;
}

export function getEffectiveKeys(issues: Issue[]): Map<string, string> {
  const byCreated = [...issues].sort((a, b) =>
    new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
  );
  if (byCreated.length === 0) return new Map();
  const virtualKeys = generateNKeysBetween(null, null, byCreated.length);
  const keyMap = new Map<string, string>();
  for (let i = 0; i < byCreated.length; i++) {
    keyMap.set(byCreated[i].id, byCreated[i].sort_order || virtualKeys[i]);
  }
  return keyMap;
}

function compareBySortKey(a: Issue, b: Issue, sortKey: SortKey): number {
  switch (sortKey) {
    case 'priority': {
      const pa = a.priority || 0;
      const pb = b.priority || 0;
      if (pa === 0 && pb === 0) return 0;
      if (pa === 0) return 1;
      if (pb === 0) return -1;
      return pa - pb;
    }
    case 'updated':
      return new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime();
    case 'title':
      return a.title.localeCompare(b.title);
    case 'estimate':
      return (b.estimate || 0) - (a.estimate || 0);
    case 'created':
    default:
      return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
  }
}

export function sortIssuesWithinGroups(issues: Issue[], sortKey: SortKey): Issue[] {
  if (sortKey === 'manual') {
    const keyMap = getEffectiveKeys(issues);
    return [...issues].sort((a, b) => {
      const sa = STATUS_ORDER.indexOf(a.status);
      const sb = STATUS_ORDER.indexOf(b.status);
      if (sa !== sb) return sa - sb;
      return compareKeys(keyMap.get(a.id) || '', keyMap.get(b.id) || '');
    });
  }
  const sorted = [...issues];
  sorted.sort((a, b) => {
    const sa = STATUS_ORDER.indexOf(a.status);
    const sb = STATUS_ORDER.indexOf(b.status);
    if (sa !== sb) return sa - sb;
    return compareBySortKey(a, b, sortKey);
  });
  return sorted;
}

export function sortGroup(issues: Issue[], sortKey: SortKey): Issue[] {
  if (sortKey === 'manual') {
    const keyMap = getEffectiveKeys(issues);
    return [...issues].sort((a, b) =>
      compareKeys(keyMap.get(a.id) || '', keyMap.get(b.id) || '')
    );
  }
  return [...issues].sort((a, b) => compareBySortKey(a, b, sortKey));
}
