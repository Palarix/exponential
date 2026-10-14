import { describe, expect, it } from "vitest";
import css from "../../index.css?raw";

// Components that draw on the AI surface: they use the --color-*-ai-* tokens,
// never the app's surface, hover, border or accent tokens.
const sources = import.meta.glob(
  ["./ChatPopup.tsx", "./AgentView.tsx", "./ChatThread.tsx", "./ChatComposer.tsx", "./ComposerPickers.tsx", "./ApprovalCard.tsx", "./ChatTitleBar.tsx"],
  { query: "?raw", import: "default", eager: true },
) as Record<string, string>;

const appTokens = /--color-(chrome|surface(-\d)?|hover-surface(-\d)?|border-(subtle|default|elevated|overlay)|accent-primary(-hover)?)\b(?!-ai)/g;

const aiTokens = [
  "surface-ai-1", "surface-ai-2", "surface-ai-3",
  "hover-surface-ai-1", "hover-surface-ai-2", "hover-surface-ai-3",
  "border-ai", "border-ai-subtle", "button-ai", "button-ai-hover",
];

// App tokens the shared primitives (Menu, PopoverPanel, prose) read, which
// .surface-ai remaps to their AI equivalents.
const remapped = [
  "surface-2", "surface-3", "hover-surface-3", "hover-surface-4",
  "border-elevated", "border-default", "border-overlay", "border-subtle",
];

describe("AI surface tokens", () => {
  it.each(aiTokens)("defines --color-%s", (token) => {
    expect(css).toMatch(new RegExp(`--color-${token}:\\s*(#[0-9A-Fa-f]{6}|var\\(--color-[\\w-]+\\));`));
  });

  it.each(remapped)(".surface-ai remaps --color-%s to an AI token", (token) => {
    const block = css.match(/\.surface-ai\s*{[^}]*}/)?.[0] ?? "";
    expect(block).toMatch(new RegExp(`--color-${token}:\\s*var\\(--color-[\\w-]*-ai[\\w-]*\\);`));
  });
});

describe.each(Object.entries(sources))("%s", (_, source) => {
  it("uses no app surface, border or accent tokens", () => {
    expect(source.match(appTokens) ?? []).toEqual([]);
  });

  it("puts every framed menu and popover panel on the AI surface", () => {
    const opened = [...source.matchAll(/<(Menu|PopoverPanel)\b(?:=>|[^>])*>/g)].map((m) => m[0]).filter((tag) => !/\bbare\b/.test(tag));
    for (const tag of opened) expect(tag).toContain("surface-ai");
  });
});

describe.each(["./ChatPopup.tsx", "./AgentView.tsx"])("%s root", (file) => {
  it("scopes its subtree to the AI surface", () => {
    expect(sources[file]).toContain("surface-ai ");
  });
});

describe("Agent view", () => {
  it("has no canvas color of its own, so the app canvas shows through", () => {
    expect(css).not.toMatch(/--color-(hover-)?surface-ai-0:/);
    const root = sources["./AgentView.tsx"].match(/<div className="surface-ai [^"]*"/)?.[0] ?? "";
    expect(root).not.toMatch(/\bbg-/);
  });
});
