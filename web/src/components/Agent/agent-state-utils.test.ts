import { describe, expect, it } from "vitest";
import { makeIssue } from "../../test-utils";
import type { AgentStatus } from "../../api/chat";
import {
  addPill,
  chatContextFor,
  composerOptions,
  loadPrefs,
  loadPills,
  normalizePrefs,
  removePill,
  savePills,
  savePrefs,
  withModel,
  withProvider,
} from "./agent-state-utils";

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
  it("round-trips through storage", () => {
    const s = memoryStorage();
    savePrefs(s, { provider: "codex", model: "gpt-x", effort: "high" });
    expect(loadPrefs(s)).toEqual({ provider: "codex", model: "gpt-x", effort: "high" });
  });

  it("reads prefs saved before providers as Claude prefs", () => {
    const s = memoryStorage({ "xpo.agent.prefs": '{"model":"opus","effort":"high"}' });
    expect(loadPrefs(s)).toEqual({ provider: "claude", model: "opus", effort: "high" });
  });

  it("returns empty prefs for malformed or blocked storage", () => {
    expect(loadPrefs(memoryStorage({ "xpo.agent.prefs": '{"model":1}' }))).toEqual({ provider: "", model: "", effort: "" });
    expect(loadPrefs(throwing)).toEqual({ provider: "", model: "", effort: "" });
  });
});

const status: AgentStatus = {
  available: true,
  default_provider: "claude",
  providers: [
    {
      id: "claude",
      name: "Claude Code",
      available: true,
      models: [
        { id: "", label: "Default model", efforts: ["", "low", "max"] },
        { id: "opus", label: "Opus", efforts: ["", "low", "max"] },
      ],
    },
    {
      id: "codex",
      name: "Codex",
      available: true,
      models: [
        { id: "", label: "Default model", efforts: ["", "low", "ultra"] },
        { id: "gpt-x", label: "GPT-X", efforts: ["", "medium"] },
      ],
    },
    { id: "gemini", name: "Gemini", available: false, reason: "not installed", models: [] },
  ],
};

describe("normalizePrefs", () => {
  it("keeps valid prefs", () => {
    expect(normalizePrefs({ provider: "codex", model: "gpt-x", effort: "medium" }, status)).toEqual({ provider: "codex", model: "gpt-x", effort: "medium" });
  });

  it("falls back to the default provider, model and effort", () => {
    expect(normalizePrefs({ provider: "", model: "", effort: "" }, status)).toEqual({ provider: "claude", model: "", effort: "" });
    expect(normalizePrefs({ provider: "gemini", model: "x", effort: "low" }, status)).toEqual({ provider: "claude", model: "", effort: "low" });
    expect(normalizePrefs({ provider: "codex", model: "opus", effort: "max" }, status)).toEqual({ provider: "codex", model: "", effort: "" });
    expect(normalizePrefs({ provider: "codex", model: "gpt-x", effort: "ultra" }, status)).toEqual({ provider: "codex", model: "gpt-x", effort: "" });
  });

  it("leaves prefs alone until the status has loaded", () => {
    const prefs = { provider: "codex", model: "gpt-x", effort: "medium" };
    expect(normalizePrefs(prefs, undefined)).toBe(prefs);
  });
});

describe("withProvider and withModel", () => {
  it("resets model and effort when the provider changes", () => {
    expect(withProvider({ provider: "claude", model: "opus", effort: "max" }, "codex")).toEqual({ provider: "codex", model: "", effort: "" });
    const same = { provider: "claude", model: "opus", effort: "max" };
    expect(withProvider(same, "claude")).toBe(same);
  });

  it("keeps the effort only when the new model supports it", () => {
    expect(withModel({ provider: "codex", model: "", effort: "low" }, "gpt-x", status)).toEqual({ provider: "codex", model: "gpt-x", effort: "" });
    expect(withModel({ provider: "codex", model: "gpt-x", effort: "" }, "", status)).toEqual({ provider: "codex", model: "", effort: "" });
    expect(withModel({ provider: "claude", model: "", effort: "max" }, "opus", status)).toEqual({ provider: "claude", model: "opus", effort: "max" });
  });
});

describe("composerOptions", () => {
  it("lists available providers and the picked provider's models and efforts", () => {
    const opts = composerOptions(status, { provider: "codex", model: "", effort: "" });
    expect(opts.providers).toEqual([
      { value: "claude", label: "Claude Code" },
      { value: "codex", label: "Codex" },
    ]);
    expect(opts.models).toEqual([
      { value: "", label: "Default model" },
      { value: "gpt-x", label: "GPT-X" },
    ]);
    expect(opts.efforts).toEqual([
      { value: "", label: "Default effort" },
      { value: "low", label: "Low" },
      { value: "ultra", label: "Ultra" },
    ]);
  });

  it("is empty before the status loads", () => {
    expect(composerOptions(undefined, { provider: "", model: "", effort: "" })).toEqual({ providers: [], models: [], efforts: [] });
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
