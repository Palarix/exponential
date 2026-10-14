import { describe, expect, it } from 'vitest';
import { FAVICON_URL, faviconFor } from './favicon';

function decode(url: string): string {
  const prefix = 'data:image/svg+xml,';
  expect(url.startsWith(prefix)).toBe(true);
  return decodeURIComponent(url.slice(prefix.length));
}

describe('faviconFor', () => {
  it('keeps the plain favicon when nothing needs attention', () => {
    expect(faviconFor('none')).toBe(FAVICON_URL);
  });
  it('adds an amber dot for an approval and a blue dot for new output', () => {
    expect(decode(faviconFor('approval'))).toContain('fill="#f2c94c"');
    expect(decode(faviconFor('output'))).toContain('fill="#4da6e8"');
  });
  it('puts the dot in the top-right corner', () => {
    expect(decode(faviconFor('approval'))).toMatch(/<circle class="ring" cx="264" cy="86"/);
  });
  it('keeps the logo inside the badged icon', () => {
    const svg = decode(faviconFor('output'));
    expect(svg).toMatch(/^<svg[\s\S]*<\/svg>\s*$/);
    expect(svg).toContain('class="bg"');
  });
});
