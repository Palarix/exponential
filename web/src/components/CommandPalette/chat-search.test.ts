import { describe, expect, it } from "vitest";
import type { Issue } from "../../api/client";
import type { ChatMeta } from "../../api/chat";
import { searchChats } from "./chat-search";

function chat(c: Partial<ChatMeta> & { id: string }): ChatMeta {
  return { title: c.id, created_at: "", updated_at: "2026-10-20T10:00:00Z", agent: "claude", state: "idle", ...c };
}

const issues = [{ id: "xpo-aaa111", title: "Login page redesign" }, { id: "xpo-bbb222", title: "Export to CSV" }] as Issue[];

const working = chat({ id: "w", title: "Implement it", mode: "work", work: { issue_id: "xpo-bbb222", worktree: "/wt" }, worked_issues: ["xpo-aaa111", "xpo-bbb222"] });
const left = chat({ id: "l", title: "Talk about tokens", worked_issues: ["xpo-aaa111"] });
const plain = chat({ id: "p", title: "Fix the flaky test" });

const ids = (r: ReturnType<typeof searchChats>) => r.map((m) => m.chat.id);

describe("searchChats", () => {
  it("returns the 5 newest chats with an empty query", () => {
    const chats = Array.from({ length: 7 }, (_, i) => chat({ id: `c${i}` }));
    expect(ids(searchChats(chats, issues, ""))).toEqual(["c0", "c1", "c2", "c3", "c4"]);
    expect(ids(searchChats(chats, issues, "   "))).toEqual(["c0", "c1", "c2", "c3", "c4"]);
  });

  it("matches the chat title", () => {
    expect(ids(searchChats([working, left, plain], issues, "flaky"))).toEqual(["p"]);
  });

  it("matches the current Work Mode issue ID", () => {
    expect(ids(searchChats([working, left, plain], issues, "bbb222"))).toEqual(["w"]);
  });

  it("matches earlier Work Mode issue IDs, newest chat first", () => {
    expect(ids(searchChats([working, left, plain], issues, "xpo-aaa111"))).toEqual(["w", "l"]);
  });

  it("matches the linked issue title", () => {
    expect(ids(searchChats([working, left, plain], issues, "csv"))).toEqual(["w"]);
    expect(ids(searchChats([working, left, plain], issues, "login page"))).toEqual(["l"]);
  });

  it("links the Work Mode issue, else the last worked issue, else none", () => {
    const got = searchChats([working, left, plain], issues, "");
    expect(got.map((m) => m.issueId)).toEqual(["xpo-bbb222", "xpo-aaa111", undefined]);
  });

  it("still matches the ID of an issue that no longer exists", () => {
    const gone = chat({ id: "g", worked_issues: ["xpo-zzz999"] });
    expect(searchChats([gone], issues, "zzz999")).toEqual([{ chat: gone, issueId: "xpo-zzz999" }]);
  });

  it("returns nothing without a match", () => {
    expect(searchChats([working, left, plain], issues, "qqqqq")).toEqual([]);
  });

  it("returns at most 8 matches", () => {
    const chats = Array.from({ length: 10 }, (_, i) => chat({ id: `c${i}`, title: "Same title" }));
    expect(searchChats(chats, issues, "same")).toHaveLength(8);
  });
});
