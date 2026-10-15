import { describe, expect, it } from "vitest";
import type { ChatMeta, ChatUsage } from "../../api/chat";
import { activeChats, archiveBlocker, filterHistory, formatCost, formatTokens, summarize, type HistoryFilter } from "./chat-history-utils";

const NOW = Date.parse("2026-10-15T12:00:00Z");
const DAY = 24 * 60 * 60 * 1000;

function usage(u: Partial<ChatUsage>): ChatUsage {
  return { turns: 0, tokens_in: 0, tokens_out: 0, cost_usd: 0, cost_known: false, ...u };
}

function chat(c: Partial<ChatMeta> & { id: string }): ChatMeta {
  return { title: c.id, created_at: "", updated_at: new Date(NOW).toISOString(), agent: "claude", state: "idle", ...c };
}

const ALL: HistoryFilter = { query: "", archived: false, agent: "", issueId: "", period: "all" };

describe("activeChats", () => {
  it("drops archived chats and keeps the order", () => {
    const chats = [chat({ id: "a" }), chat({ id: "b", archived_at: "2026-10-01T00:00:00Z" }), chat({ id: "c" })];
    expect(activeChats(chats).map((c) => c.id)).toEqual(["a", "c"]);
  });
});

describe("filterHistory", () => {
  const chats = [
    chat({ id: "a", title: "Fix Login bug", worked_issues: ["xpo-1"] }),
    chat({ id: "b", title: "Triage", agent: "codex", updated_at: new Date(NOW - 10 * DAY).toISOString() }),
    chat({ id: "c", title: "Old login notes", archived_at: "2026-10-01T00:00:00Z", updated_at: new Date(NOW - 40 * DAY).toISOString() }),
  ];
  const ids = (f: Partial<HistoryFilter>) => filterHistory(chats, { ...ALL, ...f }, NOW).map((c) => c.id);

  it("shows active chats, or archived ones", () => {
    expect(ids({})).toEqual(["a", "b"]);
    expect(ids({ archived: true })).toEqual(["c"]);
  });

  it("matches titles case-insensitively", () => {
    expect(ids({ query: "LOGIN" })).toEqual(["a"]);
    expect(ids({ query: "login", archived: true })).toEqual(["c"]);
  });

  it("limits rows to one agent", () => {
    expect(ids({ agent: "codex" })).toEqual(["b"]);
  });

  it("limits rows to chats linked to an issue", () => {
    expect(ids({ issueId: "xpo-1" })).toEqual(["a"]);
  });

  it("limits rows to chats active in the period", () => {
    expect(ids({ period: "7d" })).toEqual(["a"]);
    expect(ids({ period: "30d" })).toEqual(["a", "b"]);
  });
});

describe("summarize", () => {
  it("totals tokens and known costs", () => {
    const got = summarize([
      chat({ id: "a", usage: usage({ turns: 2, tokens_in: 100, tokens_out: 20, cost_usd: 0.5, cost_known: true }) }),
      chat({ id: "b", usage: usage({ turns: 1, tokens_in: 50, tokens_out: 5, cost_usd: 0.25, cost_known: true }) }),
    ]);
    expect(got).toEqual({ count: 2, tokensIn: 150, tokensOut: 25, costUSD: 0.75, costKnown: true, costPartial: false });
  });

  it("marks the cost partial when some chats report none", () => {
    const got = summarize([
      chat({ id: "a", usage: usage({ turns: 1, cost_usd: 0.5, cost_known: true }) }),
      chat({ id: "b", agent: "codex", usage: usage({ turns: 1, tokens_in: 9 }) }),
    ]);
    expect(got.costPartial).toBe(true);
    expect(got.costUSD).toBe(0.5);
  });

  it("knows no cost when no chat reports one", () => {
    const got = summarize([chat({ id: "a", usage: usage({ turns: 1 }) })]);
    expect(got.costKnown).toBe(false);
    expect(got.costPartial).toBe(false);
  });

  it("does not count a chat without turns against the cost", () => {
    const got = summarize([chat({ id: "a", usage: usage({ turns: 1, cost_usd: 0.1, cost_known: true }) }), chat({ id: "b" })]);
    expect(got.costPartial).toBe(false);
  });
});

describe("formatCost", () => {
  it("shows a dash for an unknown cost", () => {
    expect(formatCost(0, false)).toBe("—");
  });

  it("shows dollars with cents, and more digits under a cent", () => {
    expect(formatCost(1.234, true)).toBe("$1.23");
    expect(formatCost(0.0042, true)).toBe("$0.0042");
  });

  it("marks a partial total", () => {
    expect(formatCost(1.5, true, true)).toBe("$1.50+");
  });
});

describe("formatTokens", () => {
  it("abbreviates thousands and millions", () => {
    expect(formatTokens(0)).toBe("0");
    expect(formatTokens(950)).toBe("950");
    expect(formatTokens(12_345)).toBe("12.3K");
    expect(formatTokens(2_500_000)).toBe("2.5M");
  });
});

describe("archiveBlocker", () => {
  it("blocks a running chat and a chat in Work Mode", () => {
    expect(archiveBlocker(chat({ id: "a", state: "running" }))).toBe("Stop the running reply before archiving");
    expect(archiveBlocker(chat({ id: "a", mode: "work", work: { issue_id: "xpo-1", worktree: "/wt" } }))).toBe("Leave Work Mode on xpo-1 before archiving");
    expect(archiveBlocker(chat({ id: "a" }))).toBeNull();
  });
});
