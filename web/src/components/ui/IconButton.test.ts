import { describe, expect, it } from 'vitest';
import { iconButtonClass } from './icon-button-utils';

describe('iconButtonClass', () => {
  it('returns base classes without active or className', () => {
    const cls = iconButtonClass();
    expect(cls).toContain('w-7');
    expect(cls).toContain('h-7');
    expect(cls).toContain('border');
    expect(cls).toContain('transition-colors');
    expect(cls).not.toContain('relative');
  });

  it('adds relative when active is true', () => {
    const cls = iconButtonClass(true);
    expect(cls).toContain('relative');
  });

  it('does not add relative when active is false', () => {
    const cls = iconButtonClass(false);
    expect(cls).not.toContain('relative');
  });

  it('merges custom className', () => {
    const cls = iconButtonClass(false, 'my-custom');
    expect(cls).toContain('my-custom');
    expect(cls).toContain('w-7');
  });

  it('allows className to override base classes via tailwind-merge', () => {
    const cls = iconButtonClass(false, 'w-9');
    expect(cls).toContain('w-9');
    expect(cls).not.toContain('w-7');
  });

  it('combines active and className', () => {
    const cls = iconButtonClass(true, 'extra');
    expect(cls).toContain('relative');
    expect(cls).toContain('extra');
  });
});
