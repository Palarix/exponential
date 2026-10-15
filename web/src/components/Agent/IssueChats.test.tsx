import type { ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";
import type { ChatMeta } from "../../api/chat";
import { AppNavContext, type AppNav } from "../../app/contexts";
import { renderMarkup } from "../../render-markup";
import { AgentChatContext, type AgentChat } from "./agent-context";
import { IssueChatIcon, IssueChatsList } from "./IssueChats";

function chat(c: Partial<ChatMeta> & { id: string }): ChatMeta {
  return { title: c.id, created_at: "", updated_at: "2026-10-13T10:00:00Z", agent: "claude", state: "idle", ...c };
}

function agentWith(chats: ChatMeta[], available = true): AgentChat {
  return { status: { available, default_provider: "claude", providers: [] }, chats: chats.filter((c) => !c.archived_at), allChats: chats } as unknown as AgentChat;
}

function render(node: ReactNode, agent: AgentChat | null): string {
  return renderMarkup(
    <QueryClientProvider client={new QueryClient()}>
      <AppNavContext.Provider value={{ openChatHistory: () => {} } as unknown as AppNav}>
        <AgentChatContext.Provider value={agent}>{node}</AgentChatContext.Provider>
      </AppNavContext.Provider>
    </QueryClientProvider>,
  );
}

const past = chat({ id: "a", title: "upe-1 First try", worked_issues: ["upe-1"] });
const archived = chat({ id: "c", title: "upe-1 Old try", worked_issues: ["upe-1"], archived_at: "2026-10-01T00:00:00Z" });
const holder = chat({ id: "b", title: "upe-1 Second try", worked_issues: ["upe-1"], mode: "work", work: { issue_id: "upe-1", worktree: "/wt" } });

describe("IssueChatIcon", () => {
  it("renders nothing for an issue without chats, or without an agent", () => {
    expect(render(<IssueChatIcon issueId="upe-2" />, agentWith([past]))).toBe("");
    expect(render(<IssueChatIcon issueId="upe-1" />, agentWith([past], false))).toBe("");
    expect(render(<IssueChatIcon issueId="upe-1" />, null)).toBe("");
  });

  it("shows the count, also for one chat, and opens the chats popover", () => {
    const html = render(<IssueChatIcon issueId="upe-1" />, agentWith([past]));
    expect(html).toContain('aria-label="1 chat on upe-1"');
    expect(html).toContain('aria-haspopup="menu"');
    expect(html).toContain("lucide-message-square");
    expect(html).toContain("color-text-muted");
    expect(html).not.toMatch(/color-(accent|warning|error)/);
    expect(html).toMatch(/<\/svg>1<\/button>/);
    // The shared indicator pill, with a background on hover since it is clickable.
    expect(html).toMatch(/<button[^>]*rounded-full[^>]*border-\[var\(--color-border-subtle\)\]/);
    expect(html).toContain("hover:bg-");
  });

  it("shows the count and opens the work chat directly", () => {
    const html = render(<IssueChatIcon issueId="upe-1" />, agentWith([holder, past]));
    expect(html).toContain('aria-label="Open work chat: upe-1 Second try"');
    expect(html).not.toContain("aria-haspopup");
    // State shows by shape only: the work chat is as muted as any other.
    expect(html).toContain("color-text-muted");
    expect(html).not.toMatch(/lucide[^"]*color-(accent|warning|error|text-secondary)/);
    expect(html).toMatch(/<\/svg>2<\/button>/);
    expect(html).toContain("lucide-hammer");
  });

  it("names an untitled work chat", () => {
    const untitled = { ...holder, title: "" };
    expect(render(<IssueChatIcon issueId="upe-1" />, agentWith([untitled]))).toContain('aria-label="Open work chat: Untitled chat"');
  });

  it("uses the monochrome message-square variants for waiting, running and interrupted", () => {
    for (const [c, icon] of [
      [{ ...past, state: "running", waiting: true }, "lucide-message-square-dot"],
      [{ ...past, state: "running" }, "lucide-message-square-more"],
      [{ ...past, state: "interrupted" }, "lucide-message-square-warning"],
    ] as const) {
      const html = render(<IssueChatIcon issueId="upe-1" />, agentWith([c as ChatMeta]));
      expect(html).toContain(icon);
      expect(html).not.toMatch(/color-(accent|warning|error)/);
    }
  });
});

describe("IssueChatIcon with archived chats", () => {
  it("counts only active chats, and hides when all are archived", () => {
    expect(render(<IssueChatIcon issueId="upe-1" />, agentWith([past, archived]))).toContain('aria-label="1 chat on upe-1"');
    expect(render(<IssueChatIcon issueId="upe-1" />, agentWith([archived]))).toBe("");
  });
});

describe("IssueChatsList", () => {
  it("lists the chats with a search field, and badges the work chat", () => {
    const html = render(<IssueChatsList issueId="upe-1" onClose={() => {}} />, agentWith([holder, past]));
    expect(html).toContain('placeholder="Search chats…"');
    expect(html.indexOf("Second try")).toBeLessThan(html.indexOf("First try"));
    expect(html.match(/>Work</g)).toHaveLength(1);
    // A chat holds the issue, so the rows offer no actions.
    expect(html).not.toContain('aria-label="Chat actions"');
  });

  it("hides archived chats and links to them on the history page", () => {
    const html = render(<IssueChatsList issueId="upe-1" onClose={() => {}} />, agentWith([past, archived]));
    expect(html).not.toContain("Old try");
    expect(html).toContain("1 archived chat");
  });

  it("shows no archived link without archived chats", () => {
    expect(render(<IssueChatsList issueId="upe-1" onClose={() => {}} />, agentWith([past]))).not.toContain("archived");
  });

  it("offers row actions when no chat is in Work Mode, without showing them yet", () => {
    const html = render(<IssueChatsList issueId="upe-1" onClose={() => {}} />, agentWith([past]));
    expect(html).toContain('aria-label="Chat actions"');
    expect(html).not.toContain("Enter work mode");
  });
});
