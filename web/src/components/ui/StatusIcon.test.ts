import { describe, expect, it } from 'vitest';
import { statusIconClass } from './status-icon-utils';

describe('statusIconClass', () => {
  it('never lets the icon shrink inside a flex row', () => {
    expect(statusIconClass()).toContain('shrink-0');
  });

  it('merges a custom className while keeping shrink-0', () => {
    const cls = statusIconClass('inline-block align-[-1px] mx-0.5');
    expect(cls).toContain('shrink-0');
    expect(cls).toContain('inline-block');
    expect(cls).toContain('mx-0.5');
  });
});
