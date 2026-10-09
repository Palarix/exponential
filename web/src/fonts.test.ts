import { describe, expect, it } from 'vitest';
import css from './index.css?raw';

const fontFiles = Object.keys(import.meta.glob('./assets/fonts/*.woff2'));
const licenses = import.meta.glob('../public/licenses/*-OFL.txt', { query: '?raw', import: 'default', eager: true }) as Record<string, string>;
const fontFaces = css.match(/@font-face\s*{[^}]*}/g) ?? [];

function facesFor(family: string): string[] {
  return fontFaces.filter((f) => f.includes(`font-family: "${family}"`));
}

describe.each(['Inter', 'JetBrains Mono'])('@font-face for %s', (family) => {
  it('is declared', () => {
    expect(facesFor(family).length).toBeGreaterThan(0);
  });

  it('pins ascent, descent and line-gap metrics', () => {
    for (const face of facesFor(family)) {
      expect(face).toMatch(/ascent-override:\s*[\d.]+%/);
      expect(face).toMatch(/descent-override:\s*[\d.]+%/);
      expect(face).toMatch(/line-gap-override:\s*[\d.]+%/);
    }
  });

  it('references bundled woff2 files that exist', () => {
    for (const face of facesFor(family)) {
      const url = face.match(/url\("?([^")]+)"?\)/)?.[1];
      expect(url).toBeDefined();
      expect(fontFiles).toContain(url);
    }
  });

  it('ships the SIL Open Font License text (OFL §2)', () => {
    const file = `../public/licenses/${family.replace(/\s+/g, '')}-OFL.txt`;
    expect(licenses[file]).toContain('SIL Open Font License, Version 1.1');
    expect(licenses[file]).toContain(`The ${family} Project Authors`);
  });
});
