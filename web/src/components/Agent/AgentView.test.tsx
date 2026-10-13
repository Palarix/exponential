import { describe, expect, it } from "vitest";
import type { ChatMeta } from "../../api/chat";
import { AppNavContext, type AppNav } from "../../app/contexts";
import { renderMarkup } from "../../render-markup";
import { AgentChatContext, type AgentChat } from "./agent-context";
import AgentView from "./AgentView";

const chat: ChatMeta = { id: "c1", title: "A chat", created_at: "", updated_at: "", agent: "claude", state: "idle" };

function render(chatId: string | null): string {
  const agent = {
    status: { available: true, default_provider: "claude", providers: [] },
    chats: [chat],
    prefsFor: () => ({ provider: "", model: "", effort: "" }),
    draftFor: () => undefined,
  } as unknown as AgentChat;
  const nav = { view: "agent", chatId } as unknown as AppNav;
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

  it("offers no dock button on the new chat home", () => {
    expect(render(null)).not.toContain('aria-label="Dock chat"');
  });
});
