import { describe, expect, it, vi } from "vitest";
import type { ChatMeta } from "../../api/chat";
import { renderMarkup } from "../../render-markup";
import { chatCommandItems } from "./chat-items";

function chat(c: Partial<ChatMeta> & { id: string }): ChatMeta {
  return { title: c.id, created_at: "", updated_at: "2026-10-20T10:00:00Z", agent: "claude", state: "idle", ...c };
}

describe("chatCommandItems", () => {
  const matches = [{ chat: chat({ id: "a", title: "Fix login" }), issueId: "xpo-aaa111" }, { chat: chat({ id: "b", title: "" }) }];

  it("groups under Recent Chats without a query, Chats with one", () => {
    expect(chatCommandItems(matches, "", vi.fn()).map((i) => i.group)).toEqual(["Recent Chats", "Recent Chats"]);
    expect(chatCommandItems(matches, "fix", vi.fn()).map((i) => i.group)).toEqual(["Chats", "Chats"]);
  });

  it("labels rows with the title, or Untitled chat", () => {
    expect(chatCommandItems(matches, "", vi.fn()).map((i) => i.label)).toEqual(["Fix login", "Untitled chat"]);
  });

  it("shows the linked issue ID as meta", () => {
    const [withIssue, without] = chatCommandItems(matches, "", vi.fn());
    expect(renderMarkup(<>{withIssue.meta}</>)).toContain("xpo-aaa111");
    expect(renderMarkup(<>{without.meta}</>)).not.toContain("xpo-");
  });

  it("opens the chosen chat", () => {
    const open = vi.fn();
    chatCommandItems(matches, "", open)[0].action();
    expect(open).toHaveBeenCalledWith("a");
  });
});
