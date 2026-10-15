import type { ComponentProps } from "react";
import { describe, expect, it } from "vitest";
import type { AgentStatus } from "../../api/chat";
import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import { renderMarkup } from "../../render-markup";
import ChatComposer, { BackgroundTaskList, ComposerAttachments } from "./ChatComposer";
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

function renderComposer(status: AgentStatus, prefs: ChatPrefs, providerLocked = false, extra: Partial<ComponentProps<typeof ChatComposer>> = {}): string {
  // The app wraps everything in a tooltip provider (main.tsx).
  return renderMarkup(
    <TooltipPrimitive.Provider>
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
      {...extra}
    />
    </TooltipPrimitive.Provider>,
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

  it("shows Chat Mode as a label with no menu, and says how to enter Work Mode", () => {
    const html = renderComposer(both, { provider: "claude", model: "", effort: "" });
    const badge = html.match(/<span[^>]*title="[^"]*"[^>]*>(?:(?!<\/span>\s*<\/span>).)*>Chat</)?.[0];
    expect(badge).toBeDefined();
    expect(badge).toMatch(/title="Ask the agent to start an issue, or use Start Work on an issue\."/);
    expect(html).not.toContain("Read-only");
    expect(html).not.toContain('aria-label="Mode"');
  });

  it("shows Work Mode with the issue, its title as a tooltip, and a menu", () => {
    const html = renderComposer(both, { provider: "claude", model: "", effort: "" }, true, {
      work: { issueId: "xpo-1", title: "Fix login" },
      onLeaveWork: () => {},
      onOpenIssue: () => {},
    });
    // The hammer says Work Mode; the label is the issue alone.
    expect(html).not.toContain("Work · ");
    const mode = trigger(html, "xpo-1");
    expect(mode).toBeDefined();
    expect(mode).toContain('aria-label="Work Mode on xpo-1"');
    expect(mode).toContain('title="xpo-1 Fix login"');
    expect(mode).not.toMatch(disabledAttr);
  });

  it("disables the mode segment while a turn runs", () => {
    const html = renderComposer(both, { provider: "claude", model: "", effort: "" }, true, {
      running: true,
      work: { issueId: "xpo-1" },
      onLeaveWork: () => {},
      onOpenIssue: () => {},
    });
    expect(trigger(html, "xpo-1")).toMatch(disabledAttr);
  });

  it("prefills the draft", () => {
    const html = renderComposer(both, { provider: "claude", model: "", effort: "" }, false, { draft: "Let's implement xpo-1." });
    expect(html).toMatch(/<textarea[^>]*>Let&#x27;s implement xpo-1\.<\/textarea>/);
  });

  it("separates the segments with rules", () => {
    const html = renderComposer(both, { provider: "claude", model: "", effort: "" });
    expect(html.match(/data-segment-rule/g)).toHaveLength(2);
  });
});

describe("ChatComposer send button", () => {
  it("is a 24px circle that does not shrink", () => {
    const html = renderComposer(both, { provider: "claude", model: "", effort: "medium" });
    const send = html.match(/<button[^>]*aria-label="Send"[^>]*>/)?.[0];
    expect(send).toMatch(/\bw-6\b/);
    expect(send).toMatch(/\bh-6\b/);
    expect(send).toMatch(/\bshrink-0\b/);
  });
});

describe("ChatComposer background banner", () => {
  const prefs = { provider: "claude", model: "", effort: "" };
  const tasks = [
    { id: "a", kind: "monitor" as const, description: "Start vite dev server" },
    { id: "b", kind: "monitor" as const },
  ];
  it("shows the banner with a Stop button while background tasks run", () => {
    const html = renderComposer(both, prefs, true, { background: { title: "Monitoring 2 tasks", working: false, stopping: false, tasks, onStop: () => {} } });
    expect(html).toContain("Monitoring 2 tasks");
    // The title opens a card with the task descriptions.
    expect(html).toMatch(/<button[^>]*aria-label="Show background tasks"[^>]*>(?:(?!<\/button>).)*Monitoring 2 tasks/);
    expect(html).toContain('aria-label="Stop background tasks"');
    expect(html).toMatch(/aria-label="Send"/);
  });
  it("shows Stopping… once asked to stop", () => {
    const html = renderComposer(both, prefs, true, { background: { title: "Monitoring 2 tasks", working: false, stopping: true, tasks, onStop: () => {} } });
    expect(html).toContain("Stopping…");
    expect(html.match(/<button[^>]*aria-label="Stop background tasks"[^>]*>/)?.[0]).toMatch(disabledAttr);
  });
  it("hides the banner while a turn runs", () => {
    const html = renderComposer(both, prefs, true, { running: true, background: { title: "Monitoring 2 tasks", working: false, stopping: false, tasks, onStop: () => {} } });
    expect(html).not.toContain("Monitoring");
  });
});

describe("BackgroundTaskList", () => {
  it("lists each task's description, and names a task without one", () => {
    const html = renderMarkup(<BackgroundTaskList tasks={[{ id: "a", kind: "monitor", description: "Start vite dev server" }, { id: "b", kind: "agent" }]} />);
    expect(html).toContain("Start vite dev server");
    expect(html).toContain("Agent");
    expect(html.match(/<li/g)).toHaveLength(2);
  });
});

describe("ChatComposer attachments", () => {
  it("has a paperclip button that opens a file picker", () => {
    const html = renderComposer(both, { provider: "claude", model: "", effort: "medium" });
    expect(html).toContain('aria-label="Attach files"');
    expect(html).toMatch(/<input[^>]*type="file"[^>]*multiple/);
  });

  it("shows pending attachments as removable chips, images with a thumbnail", () => {
    const html = renderMarkup(
      <ComposerAttachments
        items={[
          { key: "1", file: new File(["x"], "shot.png", { type: "image/png" }), preview: "blob:shot" },
          { key: "2", file: new File(["x"], "notes.txt", { type: "text/plain" }) },
        ]}
        onRemove={() => {}}
      />,
    );
    expect(html).toContain('src="blob:shot"');
    expect(html).toContain("notes.txt");
    expect(html).toContain('aria-label="Remove shot.png"');
    expect(html).toContain('aria-label="Remove notes.txt"');
  });
});
