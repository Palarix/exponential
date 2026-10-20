import { describe, expect, it } from "vitest";
import css from "./index.css?raw";

/** The selector list of the first rule whose selector starts with `prefix`. */
function rule(prefix: string): string {
  const start = css.indexOf(prefix);
  expect(start).toBeGreaterThanOrEqual(0);
  return css.slice(start, css.indexOf("}", start));
}

const MOVEMENT = ["animate-step-in-forward", "animate-step-in-back", "animate-pop-in", "animate-slide-up", "animate-scale-in"];

describe("motion tiers in index.css", () => {
  it("swaps movement for a fade under reduced motion", () => {
    const reduced = rule('[data-motion="reduced"] :is(');
    for (const cls of MOVEMENT) expect(reduced).toContain(`.${cls}`);
    expect(reduced).not.toContain(".animate-fade-in");
    expect(reduced).toMatch(/animation-name:\s*fade-in/);
  });

  it("stops movement and fades when motion is off", () => {
    const off = rule('[data-motion="off"] :is(');
    for (const cls of [...MOVEMENT, "animate-fade-in"]) expect(off).toContain(`.${cls}`);
    expect(off).toMatch(/animation:\s*none/);
  });

  it("never stops status animations", () => {
    for (const status of ["animate-spin", "animate-pulse", "text-shimmer"]) {
      expect(css).not.toMatch(new RegExp(`\\[data-motion[^{]*\\.${status}`));
    }
  });

  it("points Tailwind's motion variants at the preference", () => {
    expect(css).toMatch(/@custom-variant motion-safe \(&:where\(\[data-motion="full"\]/);
    expect(css).toMatch(/@custom-variant motion-reduce \(&:where\(\[data-motion="reduced"\]/);
  });
});
