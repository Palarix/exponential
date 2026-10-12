import { describe, expect, it } from "vitest";
import { makeIssue } from "../../test-utils";
import { addPill, chatContextFor, loadPills, loadPrefs, removePill, savePills, savePrefs } from "./agent-state-utils";

function memoryStorage(initial: Record<string, string> = {}): Storage {
  const data = new Map(Object.entries(initial));
  return {
    get length() { return data.size; },
    clear: () => data.clear(),
    getItem: (k) => data.get(k) ?? null,
    key: (i) => [...data.keys()][i] ?? null,
    removeItem: (k) => { data.delete(k); },
    setItem: (k, v) => { data.set(k, v); },
  };
}

const throwing: Storage = {
  length: 0,
  clear: () => { throw new Error("blocked"); },
  getItem: () => { throw new Error("blocked"); },
  key: () => null,
  removeItem: () => { throw new Error("blocked"); },
  setItem: () => { throw new Error("blocked"); },
};

describe("pills", () => {
  it("round-trips through storage", () => {
    const s = memoryStorage();
    savePills(s, ["a", "b"]);
    expect(loadPills(s)).toEqual(["a", "b"]);
  });

  it("returns [] for missing, malformed, or blocked storage", () => {
    expect(loadPills(memoryStorage())).toEqual([]);
    expect(loadPills(memoryStorage({ "xpo.agent.pills": "{nope" }))).toEqual([]);
    expect(loadPills(memoryStorage({ "xpo.agent.pills": '[1, "a"]' }))).toEqual(["a"]);
    expect(loadPills(throwing)).toEqual([]);
    expect(loadPills(null)).toEqual([]);
    expect(() => savePills(throwing, ["a"])).not.toThrow();
  });

  it("adds without duplicates and removes", () => {
    expect(addPill(["a"], "b")).toEqual(["a", "b"]);
    expect(addPill(["a", "b"], "a")).toEqual(["a", "b"]);
    expect(removePill(["a", "b"], "a")).toEqual(["b"]);
  });
});

describe("prefs", () => {
  it("round-trips valid values", () => {
    const s = memoryStorage();
    savePrefs(s, { model: "opus", effort: "high" });
    expect(loadPrefs(s)).toEqual({ model: "opus", effort: "high" });
  });

  it("falls back to defaults for unknown values", () => {
    const s = memoryStorage({ "xpo.agent.prefs": '{"model":"gpt-5","effort":"ultra"}' });
    expect(loadPrefs(s)).toEqual({ model: "", effort: "" });
    expect(loadPrefs(throwing)).toEqual({ model: "", effort: "" });
  });
});

describe("chatContextFor", () => {
  const issues = [makeIssue({ id: "xpo-abc123", title: "Login times out" })];

  it("describes the open issue", () => {
    expect(chatContextFor("board", "xpo-abc123", issues)).toEqual({
      view: "Board",
      issue_id: "xpo-abc123",
      issue_title: "Login times out",
    });
  });

  it("keeps an unknown issue ID without a title", () => {
    expect(chatContextFor("backlog", "xpo-ffffff", issues)).toEqual({ view: "Issues", issue_id: "xpo-ffffff" });
  });

  it("describes only the view without an issue", () => {
    expect(chatContextFor("timeline", null, issues)).toEqual({ view: "Timeline" });
  });
});
