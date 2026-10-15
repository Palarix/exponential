import { describe, expect, it } from 'vitest';
import css from './index.css?raw';

function rule(selector: string): string {
  const escaped = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  return css.match(new RegExp(`(?:^|\\n)${escaped}\\s*{([^}]*)}`))?.[1] ?? '';
}

// xpo-b47189: a chip never changes the height of the line it sits in.
describe('issue-ref chips', () => {
  const chip = rule('.issue-ref,\n.prose-exponential .issue-ref');

  it('are inline boxes, so their padding paints without growing the line', () => {
    expect(chip).toMatch(/display:\s*inline;/);
    expect(chip).not.toMatch(/inline-flex/);
  });

  it('keep a line height below every parent line height', () => {
    expect(chip).toMatch(/line-height:\s*1;/);
  });

  it('share the text baseline and keep their tint when they wrap', () => {
    expect(chip).toMatch(/vertical-align:\s*baseline;/);
    expect(chip).toMatch(/(?:^|[^-])box-decoration-break:\s*clone;/);
  });

  it('center the status icon on the text', () => {
    expect(rule('.issue-ref > svg')).toMatch(/vertical-align:\s*middle;/);
  });
});
