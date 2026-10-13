import { describe, expect, it } from "vitest";
import type { AgentStatus } from "../../api/chat";
import { renderMarkup } from "../../render-markup";
import ChatComposer from "./ChatComposer";
import type { ChatPrefs } from "./agent-state-utils";

const claude = {
  id: "claude",
  name: "Claude Code",
  available: true,
  models: [{ id: "", label: "Default model", efforts: ["", "high"] }],
};
const codex = {
  id: "codex",
  name: "Codex",
  available: true,
  models: [
    { id: "", label: "Default model", efforts: [""] },
    { id: "gpt-x", label: "GPT-X", efforts: ["", "ultra"] },
  ],
};

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

describe("ChatComposer provider picker", () => {
  const both: AgentStatus = { available: true, default_provider: "claude", providers: [claude, codex] };

  it("shows the provider with the provider's models and efforts", () => {
    const html = renderComposer(both, { provider: "codex", model: "gpt-x", effort: "ultra" });
    expect(trigger(html, "Codex")).toBeDefined();
    expect(trigger(html, "GPT-X")).toBeDefined();
    expect(trigger(html, "Ultra")).toBeDefined();
  });

  it("locks the provider once the chat has started", () => {
    const html = renderComposer(both, { provider: "claude", model: "", effort: "" }, true);
    const button = trigger(html, "Claude Code");
    expect(button).toMatch(disabledAttr);
    expect(button).toContain('title="A chat stays with the agent it started with. Start a new chat to switch."');
    expect(trigger(html, "Default model")).not.toMatch(disabledAttr);
  });

  it("hides the picker when only one provider is available", () => {
    const one: AgentStatus = { available: true, default_provider: "claude", providers: [claude, { ...codex, available: false, reason: "not installed" }] };
    const html = renderComposer(one, { provider: "claude", model: "", effort: "" });
    expect(html).not.toContain("Claude Code");
    expect(html).not.toContain("Codex");
    expect(trigger(html, "Default model")).toBeDefined();
  });
});
