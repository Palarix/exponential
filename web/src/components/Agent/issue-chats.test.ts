import { describe, expect, it } from "vitest";
import type { ChatMeta } from "../../api/chat";
import { chatsForIssue, enterWorkBlocker, filterChats } from "./issue-chats";

function chat(c: Partial<ChatMeta> & { id: string }): ChatMeta {
  return { title: c.id, created_at: "", updated_at: "", agent: "claude", state: "idle", ...c };
}

describe("chatsForIssue", () => {
  it("lists the chats that worked on the issue, keeping the newest-first order", () => {
    const chats = [
      chat({ id: "a", worked_issues: ["xpo-1"] }),
      chat({ id: "b", worked_issues: ["xpo-2"] }),
      chat({ id: "c", worked_issues: ["xpo-2", "xpo-1"] }),
    ];
    const got = chatsForIssue(chats, "xpo-1");
    expect(got.chats.map((c) => c.id)).toEqual(["a", "c"]);
    expect(got.holder).toBeUndefined();
    expect(got.state).toBe("idle");
  });

  it("returns nothing for an issue without chats", () => {
    expect(chatsForIssue([chat({ id: "a" })], "xpo-1")).toEqual({ chats: [], holder: undefined, state: null, archived: 0 });
  });

  it("counts archived chats apart and leaves them out of the list and state", () => {
    const chats = [
      chat({ id: "a", worked_issues: ["xpo-1"], archived_at: "2026-10-01T00:00:00Z", state: "interrupted" }),
      chat({ id: "b", worked_issues: ["xpo-1"] }),
      chat({ id: "c", worked_issues: ["xpo-1"], archived_at: "2026-10-01T00:00:00Z" }),
    ];
    const got = chatsForIssue(chats, "xpo-1");
    expect(got.chats.map((c) => c.id)).toEqual(["b"]);
    expect(got.archived).toBe(2);
    expect(got.state).toBe("idle");
  });

  it("has no state when every chat on the issue is archived", () => {
    const got = chatsForIssue([chat({ id: "a", worked_issues: ["xpo-1"], archived_at: "2026-10-01T00:00:00Z" })], "xpo-1");
    expect(got).toEqual({ chats: [], holder: undefined, state: null, archived: 1 });
  });

  it("finds the chat in Work Mode on the issue", () => {
    const chats = [
      chat({ id: "a", worked_issues: ["xpo-1"] }),
      chat({ id: "b", worked_issues: ["xpo-1"], mode: "work", work: { issue_id: "xpo-1", worktree: "/wt" } }),
    ];
    const got = chatsForIssue(chats, "xpo-1");
    expect(got.holder?.id).toBe("b");
    expect(got.state).toBe("work");
  });

  it("ignores a chat in Work Mode on another issue as the holder", () => {
    const chats = [chat({ id: "a", worked_issues: ["xpo-1", "xpo-2"], mode: "work", work: { issue_id: "xpo-2", worktree: "/wt" } })];
    const got = chatsForIssue(chats, "xpo-1");
    expect(got.holder).toBeUndefined();
    expect(got.state).toBe("idle");
  });

  it("takes the most urgent state among the chats", () => {
    const work = chat({ id: "w", worked_issues: ["xpo-1"], mode: "work", work: { issue_id: "xpo-1", worktree: "/wt" } });
    const interrupted = chat({ id: "i", worked_issues: ["xpo-1"], state: "interrupted" });
    const running = chat({ id: "r", worked_issues: ["xpo-1"], state: "running" });
    const waiting = chat({ id: "q", worked_issues: ["xpo-1"], state: "running", waiting: true });
    expect(chatsForIssue([work, interrupted], "xpo-1").state).toBe("interrupted");
    expect(chatsForIssue([work, interrupted, running], "xpo-1").state).toBe("running");
    expect(chatsForIssue([work, running, waiting], "xpo-1").state).toBe("waiting");
  });
});

describe("filterChats", () => {
  it("matches titles case-insensitively", () => {
    const chats = [chat({ id: "a", title: "Fix login" }), chat({ id: "b", title: "Refactor" })];
    expect(filterChats(chats, "LOG").map((c) => c.id)).toEqual(["a"]);
    expect(filterChats(chats, "  ")).toHaveLength(2);
  });
});

describe("enterWorkBlocker", () => {
  it("says why a chat cannot enter Work Mode", () => {
    expect(enterWorkBlocker(chat({ id: "a" }))).toBeNull();
    expect(enterWorkBlocker(chat({ id: "a", state: "running" }))).toMatch(/running/);
    expect(enterWorkBlocker(chat({ id: "a", mode: "work", work: { issue_id: "xpo-2", worktree: "/wt" } }))).toMatch(/xpo-2/);
  });

  it("says Work Mode needs a git repository, and where to run git init", () => {
    const why = enterWorkBlocker(chat({ id: "a" }), "/work/proj");
    expect(why).toMatch(/Work Mode needs a git repository/);
    expect(why).toMatch(/git init.*\/work\/proj/);
  });
});
