import { describe, expect, it } from 'vitest';
import { nextIndex, matchShortcut, pointInTriangle } from './menu-utils';

describe('nextIndex', () => {
  const skip = new Set<number>();

  it('moves forward by 1', () => {
    expect(nextIndex(0, 5, 1, skip)).toBe(1);
  });

  it('moves backward by 1', () => {
    expect(nextIndex(2, 5, -1, skip)).toBe(1);
  });

  it('wraps forward from last to first', () => {
    expect(nextIndex(4, 5, 1, skip)).toBe(0);
  });

  it('wraps backward from first to last', () => {
    expect(nextIndex(0, 5, -1, skip)).toBe(4);
  });

  it('skips non-navigable indices going forward', () => {
    const s = new Set([2, 3]);
    expect(nextIndex(1, 5, 1, s)).toBe(4);
  });

  it('skips non-navigable indices going backward', () => {
    const s = new Set([1, 2]);
    expect(nextIndex(3, 5, -1, s)).toBe(0);
  });

  it('wraps and skips combined', () => {
    const s = new Set([0, 1]);
    expect(nextIndex(4, 5, 1, s)).toBe(2);
  });

  it('returns -1 when all others skipped', () => {
    const s = new Set([0, 1, 3, 4]);
    expect(nextIndex(2, 5, 1, s)).toBe(-1);
  });

  it('returns -1 when all items skipped', () => {
    const s = new Set([0, 1, 2]);
    expect(nextIndex(-1, 3, 1, s)).toBe(-1);
  });

  it('handles count of 0', () => {
    expect(nextIndex(0, 0, 1, skip)).toBe(-1);
  });

  it('moves to first from -1 (no focus yet)', () => {
    expect(nextIndex(-1, 5, 1, skip)).toBe(0);
  });

  it('moves to last from -1 going backward', () => {
    expect(nextIndex(-1, 5, -1, skip)).toBe(4);
  });

  it('skips from -1 going forward', () => {
    const s = new Set([0, 1]);
    expect(nextIndex(-1, 5, 1, s)).toBe(2);
  });
});

describe('matchShortcut', () => {
  const shortcuts = new Map<number, string>([
    [0, 'S'],
    [1, 'P'],
    [2, 'A'],
    [4, 'E'],
  ]);

  it('returns the index for a matching shortcut (case-insensitive)', () => {
    expect(matchShortcut('s', shortcuts)).toBe(0);
    expect(matchShortcut('P', shortcuts)).toBe(1);
    expect(matchShortcut('e', shortcuts)).toBe(4);
  });

  it('returns -1 for no match', () => {
    expect(matchShortcut('x', shortcuts)).toBe(-1);
  });

  it('returns -1 for empty shortcuts', () => {
    expect(matchShortcut('s', new Map())).toBe(-1);
  });
});

describe('pointInTriangle', () => {
  it('returns true for a point inside the triangle', () => {
    expect(pointInTriangle(1, 1, 0, 0, 3, 0, 0, 3)).toBe(true);
  });

  it('returns false for a point outside the triangle', () => {
    expect(pointInTriangle(3, 3, 0, 0, 3, 0, 0, 3)).toBe(false);
  });

  it('returns true for a point on the edge', () => {
    expect(pointInTriangle(1.5, 0, 0, 0, 3, 0, 0, 3)).toBe(true);
  });

  it('returns true for a vertex', () => {
    expect(pointInTriangle(0, 0, 0, 0, 3, 0, 0, 3)).toBe(true);
  });

  it('returns false for degenerate triangle (collinear)', () => {
    expect(pointInTriangle(1, 1, 0, 0, 2, 2, 4, 4)).toBe(false);
  });
});
