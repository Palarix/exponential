import { describe, expect, it } from "vitest";
import type { ChatMeta } from "../../api/chat";
import { AppNavContext, type AppNav } from "../../app/contexts";
import { renderMarkup } from "../../render-markup";
import { AgentChatContext, type AgentChat } from "./agent-context";
import AgentView from "./AgentView";

const chat: ChatMeta = { id: "c1", title: "A chat", created_at: "", updated_at: "", agent: "claude", state: "idle" };
const archived: ChatMeta = { ...chat, id: "c2", title: "Old chat", archived_at: "2026-10-01T00:00:00Z" };

function render(chatId: string | null, history: AppNav["chatHistory"] = null): string {
  const agent = {
    status: { available: true, default_provider: "claude", providers: [] },
    chats: [chat],
    allChats: [chat, archived],
    seen: {},
    markSeen: () => {},
    prefsFor: () => ({ provider: "", model: "", effort: "" }),
    draftFor: () => undefined,
  } as unknown as AgentChat;
  const nav = { view: "agent", chatId, chatHistory: history } as unknown as AppNav;
  return renderMarkup(
    <AppNavContext.Provider value={nav}>
      <AgentChatContext.Provider value={agent}>
        <AgentView />
      </AgentChatContext.Provider>
    </AppNavContext.Provider>,
  );
}

describe("AgentView", () => {
  it("offers a dock button in the header for a chat", () => {
    const html = render("c1");
    expect(html).toMatch(/<button[^>]*aria-label="Dock chat"[^>]*><svg[^>]*lucide-dock/);
  });

  it("greets with building, not just asking, on the new chat home", () => {
    const html = render(null);
    expect(html).toMatch(/<h1[^>]*>What should we build in this project\?<\/h1>/);
    expect(html).toContain('placeholder="Ask a question, or describe what to build…"');
    expect(html).not.toContain("What do you want to know");
  });

  it("offers a build card between the bug and summary suggestions", () => {
    const labels = [...render(null).matchAll(/<span class="block text-sm font-medium[^"]*">([^<]*)<\/span>/g)].map((m) => m[1]);
    expect(labels).toEqual(["File a bug", "Plan and build a feature", "Summarize my open issues"]);
    expect(render(null)).toMatch(/lucide-hammer.*Plan and build a feature/s);
  });

  it("offers no dock button on the new chat home", () => {
    expect(render(null)).not.toContain('aria-label="Dock chat"');
  });

  it("links to All chats from the header", () => {
    expect(render(null)).toMatch(/<button[^>]*>.*All chats<\/button>/s);
    expect(render("c1")).toContain("All chats");
  });

  it("shows an archived chat read only, with an Unarchive button", () => {
    const html = render("c2");
    expect(html).toContain("This chat is archived");
    expect(html).toContain("Unarchive");
    expect(html).not.toContain("<textarea");
  });

  it("shows the composer for an active chat", () => {
    const html = render("c1");
    expect(html).toContain("<textarea");
    expect(html).not.toContain("This chat is archived");
  });

  it("shows the chat history page", () => {
    const html = render(null, { issueId: null });
    expect(html).toContain('placeholder="Search chats…"');
    expect(html).toContain("A chat");
    expect(html).not.toContain("Old chat");
  });

  it("keeps usage out of the table and hides row checkboxes until hover", () => {
    const html = render(null, { issueId: null });
    expect(html).not.toMatch(/<th[^>]*>(Turns|Cost|Tokens)/);
    // No checkbox column: the checkbox stacks on the state icon and shows on hover.
    expect(html).not.toContain('class="w-8 px-3');
    expect(html).toMatch(/<input[^>]*aria-label="Select A chat"[^>]*class="peer [^"]*opacity-0 group-hover:opacity-100[^"]*"[^>]*\/><span[^>]*><span title="Idle"[^>]*>.*?<\/span><\/span><\/span><\/span><span class="truncate[^"]*">A chat</s);
    // The icon hides for keyboard focus only, like the checkbox shows: a clicked
    // (mouse-focused) checkbox must not leave the slot empty after unselecting.
    expect(html).toContain("group-hover:invisible peer-focus-visible:invisible");
    expect(html).not.toContain("group-focus-within:invisible");
    // Select-all sits in the same slot of the Chat header.
    expect(html).toMatch(/<th[^>]*><span[^>]*><span[^>]*><input[^>]*aria-label="Select all chats"[^>]*\/><span[^>]*><\/span><\/span>Chat<\/span><\/th>/);
    // An idle chat's row has no message icon.
    expect(html.slice(html.indexOf("<tbody"))).not.toContain("lucide-message-square");
    // Checked boxes take the accent color.
    expect(html).toMatch(/<input[^>]*aria-label="Select all chats"[^>]*accent-\[var\(--color-accent-primary\)\]/);
    expect(html).toMatch(/<input[^>]*aria-label="Select A chat"[^>]*accent-\[var\(--color-accent-primary\)\]/);
    // The bulk bar keeps its slot, hidden, so selecting a row does not move the table.
    expect(html).toMatch(/<div[^>]*aria-hidden="true"[^>]*invisible[^>]*>.*0 selected/s);
    // "Clear selection" ends the bar, flush right: the bar has no right padding.
    expect(html).toMatch(/<div aria-hidden="true" class="[^"]*\bpl-3 pr-0\b[^"]*">.*Clear selection<\/button><\/div>/s);
    // Full width: no centered max-width column.
    expect(html).not.toContain("max-w-6xl");
  });

  // Same symmetric gutter as the thread scroller, so nothing shifts sideways
  // when the pane starts overflowing or the landing pane gives way to a thread.
  it("reserves a stable scrollbar gutter on the landing and history panes", () => {
    const scrollerClass = (html: string, attr: string) => html.match(new RegExp(`<div ${attr}="true" class="([^"]*)"`))?.[1] ?? "";
    for (const scroller of [scrollerClass(render(null), "data-landing-scroller"), scrollerClass(render(null, { issueId: null }), "data-history-scroller")]) {
      expect(scroller).toContain("scroll-stable");
      expect(scroller).not.toMatch(/overflow-y-/);
    }
  });
});
