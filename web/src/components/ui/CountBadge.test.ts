import { describe, expect, it } from 'vitest';
import { formatCount, countBadgeClass } from './count-badge-utils';

describe('formatCount', () => {
  it('returns count with plural label for 0', () => {
    expect(formatCount(0)).toBe('0 issues');
  });

  it('returns count with singular label for 1', () => {
    expect(formatCount(1)).toBe('1 issue');
  });

  it('returns count with plural label for many', () => {
    expect(formatCount(5)).toBe('5 issues');
  });

  it('uses custom label singular', () => {
    expect(formatCount(1, 'file')).toBe('1 file');
  });

  it('uses custom label plural', () => {
    expect(formatCount(3, 'file')).toBe('3 files');
  });

  it('returns count only when short is true', () => {
    expect(formatCount(7, 'issue', true)).toBe('7');
  });

  it('returns count only for short even with custom label', () => {
    expect(formatCount(1, 'file', true)).toBe('1');
  });
});

describe('countBadgeClass', () => {
  it('returns base classes by default', () => {
    const cls = countBadgeClass();
    expect(cls).toContain('text-xs');
    expect(cls).toContain('tabular-nums');
    expect(cls).toContain('text-[var(--color-text-muted)]');
    expect(cls).not.toContain('rounded-full');
  });

  it('adds pill classes when rounded is true', () => {
    const cls = countBadgeClass(true);
    expect(cls).toContain('rounded-full');
    expect(cls).toContain('bg-[var(--color-hover-surface)]');
    expect(cls).toContain('inline-flex');
  });

  it('keeps base classes when rounded is true', () => {
    const cls = countBadgeClass(true);
    expect(cls).toContain('text-xs');
    expect(cls).toContain('tabular-nums');
  });
});
