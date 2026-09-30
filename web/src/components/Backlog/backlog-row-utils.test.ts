import { describe, expect, it } from 'vitest';
import { nodeIndicator } from './backlog-row-utils';

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

  it('returns expanded for a ghost parent with visible children', () => {
    expect(nodeIndicator({ ...base, isGhostParent: true })).toBe('expanded');
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
