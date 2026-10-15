import { describe, expect, it } from 'vitest';
import css from './index.css?raw';

type RGB = [number, number, number];

function hex(value: string): RGB {
  const m = value.match(/^#([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$/i);
  if (!m) throw new Error(`not a hex color: ${value}`);
  return [parseInt(m[1], 16), parseInt(m[2], 16), parseInt(m[3], 16)];
}

function luminance(rgb: RGB): number {
  const [r, g, b] = rgb.map((c) => {
    const s = c / 255;
    return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function contrast(a: RGB, b: RGB): number {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
}

function rule(selector: string): string {
  const escaped = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  return css.match(new RegExp(`(?:^|\\n)${escaped}\\s*{([^}]*)}`))?.[1] ?? '';
}

const link = css.match(/--color-link:\s*(#[0-9a-f]{6});/i)?.[1] ?? '';
const surfaces = [...css.matchAll(/--color-(surface[\w-]*):\s*(#[0-9a-f]{6})/gi)].map((m) => [m[1], m[2]] as const);

describe('--color-link', () => {
  it('is defined as a hex color', () => {
    expect(link).toMatch(/^#[0-9a-f]{6}$/i);
  });

  it.each(surfaces)('has at least 4.5:1 contrast on --color-%s', (_, bg) => {
    expect(contrast(hex(link), hex(bg))).toBeGreaterThanOrEqual(4.5);
  });
});

describe('prose links', () => {
  it('take their color from the link token, without an underline at rest', () => {
    expect(rule('.prose-exponential a')).toMatch(/color:\s*var\(--color-link\);/);
    expect(rule('.prose-exponential a')).toMatch(/text-decoration:\s*none;/);
  });

  it('underline on hover', () => {
    expect(rule('.prose-exponential a:hover')).toMatch(/text-decoration:\s*underline;/);
  });

  it('leave issue-ref chips without an underline on hover', () => {
    expect(rule('.issue-ref:hover,\n.prose-exponential .issue-ref:hover')).toMatch(/text-decoration:\s*none;/);
  });
});
