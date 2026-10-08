import { describe, expect, it } from 'vitest';
import { buildBacklogRows, type RowItem } from './useBacklogRows';
import { makeIssue } from '../../test-utils';

// Epic in DOING with one child in DOING and two in PLANNED. On the Active
// tab, the PLANNED group shows the epic as a ghost parent over its two
// PLANNED children.
const epic = makeIssue({ id: 'xpo-epic01', status: 'DOING', sort_order: 'a0' });
const doingChild = makeIssue({
  id: 'xpo-kid001',
  status: 'DOING',
  parent_id: epic.id,
  sort_order: 'a1',
});
const plannedA = makeIssue({
  id: 'xpo-kid002',
  status: 'PLANNED',
  parent_id: epic.id,
  sort_order: 'a2',
});
const plannedB = makeIssue({
  id: 'xpo-kid003',
  status: 'PLANNED',
  parent_id: epic.id,
  sort_order: 'a3',
});
const issues = [epic, doingChild, plannedA, plannedB];

const build = (expandedNodes: Set<string>, filteredIssues = issues) =>
  buildBacklogRows(
    issues,
    filteredIssues,
    'active',
    new Set(['PLANNED', 'DOING', 'BLOCKED']),
    expandedNodes,
    'manual',
  );

const issueRows = (rows: RowItem[], status: string) => {
  const start = rows.findIndex((r) => r.kind === 'group' && r.status === status);
  const out: Extract<RowItem, { kind: 'issue' }>[] = [];
  for (let i = start + 1; i < rows.length; i++) {
    const r = rows[i];
    if (r.kind === 'group') break;
    out.push(r);
  }
  return out;
};

describe('buildBacklogRows ghost parents', () => {
  it('shows ghost-parent children when the parent is expanded', () => {
    const planned = issueRows(build(new Set([epic.id])), 'PLANNED');
    expect(planned.map((r) => r.issue.id)).toEqual([
      epic.id,
      plannedA.id,
      plannedB.id,
    ]);
    expect(planned[0].isGhostParent).toBe(true);
    expect(planned[0].hasVisibleChildren).toBe(true);
  });

  it('hides ghost-parent children when the parent is collapsed', () => {
    const planned = issueRows(build(new Set()), 'PLANNED');
    expect(planned.map((r) => r.issue.id)).toEqual([epic.id]);
    expect(planned[0].isGhostParent).toBe(true);
    expect(planned[0].hasVisibleChildren).toBe(true);
  });

  it('collapses the real parent and its ghost together (shared per issue id)', () => {
    const rows = build(new Set());
    expect(issueRows(rows, 'DOING').map((r) => r.issue.id)).toEqual([epic.id]);
    expect(issueRows(rows, 'PLANNED').map((r) => r.issue.id)).toEqual([epic.id]);
  });

  it('renders a search hit under an expanded ghost parent', () => {
    // Search matched only plannedA: the epic is filtered out and becomes a
    // ghost parent in PLANNED.
    const planned = issueRows(build(new Set([epic.id]), [plannedA]), 'PLANNED');
    expect(planned.map((r) => r.issue.id)).toEqual([epic.id, plannedA.id]);
    expect(planned[0].isGhostParent).toBe(true);
  });
});
