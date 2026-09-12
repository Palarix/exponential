import { describe, expect, it } from 'vitest';
import { textClass } from './text-utils';

describe('textClass', () => {
  it('returns muted text-sm font-normal by default', () => {
    const cls = textClass();
    expect(cls).toBe('text-sm text-[var(--color-text-muted)] font-normal');
  });

  it('maps color to CSS variable', () => {
    expect(textClass({ color: 'primary' })).toContain('text-[var(--color-text-primary)]');
    expect(textClass({ color: 'secondary' })).toContain('text-[var(--color-text-secondary)]');
    expect(textClass({ color: 'success' })).toContain('text-[var(--color-success)]');
    expect(textClass({ color: 'error' })).toContain('text-[var(--color-error)]');
    expect(textClass({ color: 'warning' })).toContain('text-[var(--color-warning)]');
    expect(textClass({ color: 'accent' })).toContain('text-[var(--color-accent-primary)]');
  });

  it('maps size to tailwind class', () => {
    expect(textClass({ size: 'xs' })).toContain('text-xs');
    expect(textClass({ size: 'regular' })).toContain('text-sm');
    expect(textClass({ size: 'base' })).toContain('text-base');
    expect(textClass({ size: 'lg' })).toContain('text-lg');
  });

  it('adds weight class when not regular', () => {
    expect(textClass({ weight: 'medium' })).toContain('font-medium');
    expect(textClass({ weight: 'semibold' })).toContain('font-semibold');
    expect(textClass({ weight: 'bold' })).toContain('font-bold');
  });

  it('adds font-normal for regular weight to prevent inheritance', () => {
    expect(textClass({ weight: 'regular' })).toContain('font-normal');
  });

  it('adds mono class', () => {
    expect(textClass({ mono: true })).toContain('font-mono');
  });

  it('adds tabular-nums class', () => {
    expect(textClass({ tabular: true })).toContain('tabular-nums');
  });

  it('adds truncate and min-w-0', () => {
    const cls = textClass({ truncate: true });
    expect(cls).toContain('truncate');
    expect(cls).toContain('min-w-0');
  });

  it('combines multiple options', () => {
    const cls = textClass({ color: 'primary', size: 'xs', weight: 'medium', mono: true, tabular: true });
    expect(cls).toContain('text-xs');
    expect(cls).toContain('text-[var(--color-text-primary)]');
    expect(cls).toContain('font-medium');
    expect(cls).toContain('font-mono');
    expect(cls).toContain('tabular-nums');
  });
});
