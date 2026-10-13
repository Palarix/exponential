import { describe, expect, it } from "vitest";
import type { AgentStatus } from "../../api/chat";
import { renderMarkup } from "../../render-markup";
import ChatComposer from "./ChatComposer";
import type { ChatPrefs } from "./agent-state-utils";

const claude = {
  id: "claude",
  name: "Claude Code",
  available: true,
  models: [
    { id: "", label: "Default model", name: "Claude Opus 5.5", context_window: 1000000, efforts: ["low", "medium"] },
    { id: "haiku", label: "Haiku", efforts: ["low", "medium"] },
  ],
};
const codex = {
  id: "codex",
  name: "Codex",
  available: true,
  models: [
    { id: "", label: "Default model", efforts: ["medium"] },
    { id: "gpt-x", label: "GPT-X", name: "GPT-X", efforts: ["medium", "ultra"] },
  ],
};
const both: AgentStatus = { available: true, default_provider: "claude", providers: [claude, codex] };

function renderComposer(status: AgentStatus, prefs: ChatPrefs, providerLocked = false): string {
  return renderMarkup(
    <ChatComposer
      running={false}
      placeholder="Ask"
      onDismissContext={() => {}}
      status={status}
      prefs={prefs}
      providerLocked={providerLocked}
      onPrefsChange={() => {}}
      onSend={async () => true}
      onStop={() => {}}
    />,
  );
}

const disabledAttr = /\sdisabled(=""|\s|>)/;

/** The trigger button whose label is text. */
function trigger(html: string, text: string): string | undefined {
  return html.match(new RegExp(`<button[^>]*>(?:(?!</button>).)*<span[^>]*>${text}</span>`))?.[0];
}

describe("ChatComposer pickers", () => {
  it("shows the full model name with the provider icon, and effort with the context window", () => {
    const html = renderComposer(both, { provider: "claude", model: "", effort: "medium" });
    const model = trigger(html, "Claude Opus 5.5");
    expect(model).toBeDefined();
    expect(model).toContain('data-provider-icon="claude"');
    expect(model).toContain("text-[#D97857]");
    expect(trigger(html, "Medium · 1M")).toBeDefined();
  });

  it("falls back to the alias until the name is known", () => {
    const html = renderComposer(both, { provider: "claude", model: "haiku", effort: "low" });
    expect(trigger(html, "Haiku")).toBeDefined();
    expect(trigger(html, "Low")).toBeDefined();
  });

  it("uses the provider's icon", () => {
    const html = renderComposer(both, { provider: "codex", model: "gpt-x", effort: "ultra" });
    expect(trigger(html, "GPT-X")).toContain('data-provider-icon="codex"');
    expect(trigger(html, "Ultra")).toBeDefined();
  });

  it("keeps the model picker usable once the chat has started", () => {
    const html = renderComposer(both, { provider: "claude", model: "", effort: "" }, true);
    expect(trigger(html, "Claude Opus 5.5")).not.toMatch(disabledAttr);
  });

  it("shows a read-only badge that explains where code changes happen", () => {
    const html = renderComposer(both, { provider: "claude", model: "", effort: "" });
    const badge = html.match(/<span[^>]*title="[^"]*"[^>]*>(?:(?!<\/span>\s*<\/span>).)*Read-only/)?.[0];
    expect(badge).toBeDefined();
    expect(badge).toContain("lucide-lock");
    expect(badge).toMatch(/title="Chat can read code and change issues\. Code changes happen in work sessions\."/);
  });

  it("separates the segments with rules", () => {
    const html = renderComposer(both, { provider: "claude", model: "", effort: "" });
    expect(html.match(/data-segment-rule/g)).toHaveLength(2);
  });
});
