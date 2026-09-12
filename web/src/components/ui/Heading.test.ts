import { describe, expect, it } from 'vitest';
import { headingClass, headingWrapperClass } from './heading-utils';

describe('headingClass', () => {
  it('returns base classes with no options', () => {
    const cls = headingClass();
    expect(cls).toContain('text-sm');
    expect(cls).toContain('font-medium');
    expect(cls).toContain('text-[var(--color-text-primary)]');
  });

  it('adds leading-snug when leading is snug', () => {
    const cls = headingClass({ leading: 'snug' });
    expect(cls).toContain('leading-snug');
  });

  it('adds truncate and min-w-0 when truncate is true', () => {
    const cls = headingClass({ truncate: true });
    expect(cls).toContain('truncate');
    expect(cls).toContain('min-w-0');
  });

  it('does not add truncate by default', () => {
    const cls = headingClass();
    expect(cls).not.toContain('truncate');
  });

  it('adds line-clamp when clamp is set', () => {
    const cls = headingClass({ clamp: 2 });
    expect(cls).toContain('line-clamp-2');
  });

  it('combines multiple options', () => {
    const cls = headingClass({ leading: 'snug', truncate: true });
    expect(cls).toContain('leading-snug');
    expect(cls).toContain('truncate');
    expect(cls).toContain('text-sm');
  });
});

describe('headingWrapperClass', () => {
  it('returns flex wrapper classes with gap', () => {
    const cls = headingWrapperClass();
    expect(cls).toContain('flex');
    expect(cls).toContain('items-center');
    expect(cls).toContain('gap-3');
  });
});
