import { describe, expect, it } from "vitest";
import type { ChatMeta } from "../../api/chat";
import { renderMarkup } from "../../render-markup";
import { chatStateIcon } from "./recent-chats";

const chat = (over: Partial<ChatMeta> = {}): ChatMeta => ({
  id: "c1", title: "", created_at: "2026-10-14T10:00:00Z", updated_at: "2026-10-14T10:00:00Z", agent: "claude", state: "idle", ...over,
});
const render = (c: ChatMeta, seen: Record<string, string> = {}) => renderMarkup(<>{chatStateIcon(c, seen, "w-2.5 h-2.5", <i>idle</i>)}</>);

describe("chatStateIcon", () => {
  it("shows a warning dot while an approval waits", () => {
    const html = render(chat({ waiting: true }));
    expect(html).toContain("--color-warning");
    expect(html).toContain('aria-label="Waiting for your approval"');
    expect(html).not.toContain("lucide");
  });
  it("keeps the spinner while running", () => {
    expect(render(chat({ state: "running" }))).toContain("animate-spin");
  });
  it("shows an error dot for an interrupted chat", () => {
    const html = render(chat({ state: "interrupted" }));
    expect(html).toContain("--color-error");
    expect(html).toContain('aria-label="Interrupted"');
  });
  it("shows a blue dot for a reply not seen yet", () => {
    const html = render(chat({ updated_at: "2026-10-14T10:05:00Z" }), { c1: "2026-10-14T10:00:00Z" });
    expect(html).toContain("--color-info");
    expect(html).toContain('aria-label="New reply"');
  });
  it("shows the idle fallback otherwise", () => {
    expect(render(chat(), { c1: "2026-10-14T10:00:00Z" })).toBe("<i>idle</i>");
  });
});
