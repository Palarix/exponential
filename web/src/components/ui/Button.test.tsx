import { describe, expect, it } from "vitest";
import { renderMarkup } from "../../render-markup";
import Button from "./Button";

const html = (el: Parameters<typeof renderMarkup>[0]) => renderMarkup(el);

describe("Button", () => {
  it("tints secondary with its own text color, with no border by default", () => {
    const b = html(<Button variant="secondary">Save</Button>);
    expect(b).toContain("bg-[var(--color-tint-rest)]");
    expect(b).toContain("hover:bg-[var(--color-tint-hover)]");
    expect(b).toContain("active:bg-[var(--color-tint-active)]");
    expect(b).toContain("text-[var(--color-text-primary)]");
    expect(b).not.toContain("border border-");
  });

  it("adds a tinted border when bordered", () => {
    expect(html(<Button variant="secondary" bordered>Save</Button>)).toContain("border border-[var(--color-tint-border)]");
  });

  it("colors secondary and ghost text by tone, and the tint follows", () => {
    expect(html(<Button variant="secondary" tone="danger">Delete</Button>)).toContain("text-[var(--color-error)]");
    expect(html(<Button variant="ghost" tone="warning">Retry</Button>)).toContain("text-[var(--color-warning)]");
  });

  it("gives ghost no fill at rest and secondary's resting tint on hover", () => {
    const b = html(<Button variant="ghost">More</Button>);
    expect(b).toContain("bg-transparent");
    expect(b).toContain("hover:bg-[var(--color-tint-rest)]");
    expect(b).toContain("active:bg-[var(--color-tint-hover)]");
  });

  it("darkens primary when pressed", () => {
    expect(html(<Button>Go</Button>)).toContain("active:bg-[var(--color-accent-primary-active)]");
  });

  it("gives danger and warning a solid scale around their base color", () => {
    const danger = html(<Button variant="danger">Delete</Button>);
    expect(danger).toContain("bg-[var(--color-error-solid)]");
    expect(danger).toContain("hover:bg-[var(--color-error-solid-hover)]");
    expect(danger).toContain("active:bg-[var(--color-error-solid-active)]");
    expect(danger).not.toContain("opacity-90");
    const warning = html(<Button variant="warning">Careful</Button>);
    expect(warning).toContain("bg-[var(--color-warning-solid)]");
    expect(warning).toContain("text-[var(--color-text-on-warning)]");
  });

  it("ignores tone on solid variants", () => {
    expect(html(<Button variant="danger" tone="warning">Delete</Button>)).not.toContain("text-[var(--color-warning)]");
  });
});
