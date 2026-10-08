import { describe, expect, it } from 'vitest';
import { expandedNodeIds, nodeIndicator } from './backlog-row-utils';

const base = {
  hasChildren: true,
  hasVisibleChildren: true,
  isGhostParent: false,
  hierarchyMode: 'nested' as const,
};

describe('nodeIndicator', () => {
  it('returns toggle for a nested parent with visible children', () => {
    expect(nodeIndicator(base)).toBe('toggle');
  });

  it('returns toggle for a ghost parent with visible children', () => {
    expect(nodeIndicator({ ...base, isGhostParent: true })).toBe('toggle');
  });

  it('returns none for a parent whose children are all hidden', () => {
    expect(nodeIndicator({ ...base, hasVisibleChildren: false })).toBe('none');
  });

  it('returns none for a ghost parent with no visible children', () => {
    expect(
      nodeIndicator({ ...base, hasVisibleChildren: false, isGhostParent: true }),
    ).toBe('none');
  });

  it('returns none for an issue without children', () => {
    expect(
      nodeIndicator({ ...base, hasChildren: false, hasVisibleChildren: false }),
    ).toBe('none');
  });

  it('returns none in flat mode', () => {
    expect(nodeIndicator({ ...base, hierarchyMode: 'flat' })).toBe('none');
  });
});

describe('expandedNodeIds', () => {
  const parentIds = ['xpo-p1', 'xpo-p2', 'xpo-p3'];

  it('expands every parent not in the persisted collapsed set', () => {
    const expanded = expandedNodeIds({
      parentIds,
      persistedCollapsed: new Set(['xpo-p2']),
      transientCollapsed: new Set(),
      filtering: false,
    });
    expect([...expanded].sort()).toEqual(['xpo-p1', 'xpo-p3']);
  });

  it('ignores the persisted collapsed set while filtering', () => {
    const expanded = expandedNodeIds({
      parentIds,
      persistedCollapsed: new Set(['xpo-p1', 'xpo-p2']),
      transientCollapsed: new Set(),
      filtering: true,
    });
    expect([...expanded].sort()).toEqual(['xpo-p1', 'xpo-p2', 'xpo-p3']);
  });

  it('honours the transient collapsed set while filtering', () => {
    const expanded = expandedNodeIds({
      parentIds,
      persistedCollapsed: new Set(),
      transientCollapsed: new Set(['xpo-p3']),
      filtering: true,
    });
    expect([...expanded].sort()).toEqual(['xpo-p1', 'xpo-p2']);
  });

  it('ignores the transient collapsed set when not filtering', () => {
    const expanded = expandedNodeIds({
      parentIds,
      persistedCollapsed: new Set(),
      transientCollapsed: new Set(['xpo-p3']),
      filtering: false,
    });
    expect([...expanded].sort()).toEqual(['xpo-p1', 'xpo-p2', 'xpo-p3']);
  });
});
