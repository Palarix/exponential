import { describe, expect, it } from "vitest";
import { makeIssue } from "../../test-utils";
import type { AgentStatus } from "../../api/chat";
import {
  addPill,
  clampPopupSize,
  loadPopupSize,
  savePopupSize,
  DEFAULT_POPUP_SIZE,
  popupLeft,
  chatListPollMs,
  chatContextFor,
  adoptNewChatContext,
  consumeContext,
  consumedFor,
  nextViewed,
  pendingContextIssue,
  chatPrefs,
  composerOptions,
  formatContextWindow,
  loadPrefs,
  loadPills,
  modelKey,
  normalizePrefs,
  pickModel,
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
        { id: "", label: "Default model", name: "Claude Opus 5.5", context_window: 1000000, efforts: ["low", "medium", "max"] },
        { id: "opus", label: "Opus", efforts: ["low", "medium", "max"] },
      ],
    },
    {
      id: "codex",
      name: "Codex",
      available: true,
      models: [
        { id: "", label: "Default model", efforts: ["low", "ultra"] },
        { id: "gpt-x", label: "GPT-X", efforts: ["medium"] },
      ],
    },
    { id: "gemini", name: "Gemini", available: false, reason: "not installed", models: [] },
  ],
};

describe("normalizePrefs", () => {
  it("keeps valid prefs", () => {
    expect(normalizePrefs({ provider: "codex", model: "gpt-x", effort: "medium" }, status)).toEqual({ provider: "codex", model: "gpt-x", effort: "medium" });
  });

  it("falls back to the default provider and model, and to Medium", () => {
    expect(normalizePrefs({ provider: "", model: "", effort: "" }, status)).toEqual({ provider: "claude", model: "", effort: "medium" });
    expect(normalizePrefs({ provider: "gemini", model: "x", effort: "low" }, status)).toEqual({ provider: "claude", model: "", effort: "low" });
    expect(normalizePrefs({ provider: "codex", model: "gpt-x", effort: "ultra" }, status)).toEqual({ provider: "codex", model: "gpt-x", effort: "medium" });
  });

  it("falls back to the model's first effort when it has no Medium", () => {
    expect(normalizePrefs({ provider: "codex", model: "opus", effort: "max" }, status)).toEqual({ provider: "codex", model: "", effort: "low" });
  });

  it("leaves prefs alone until the status has loaded", () => {
    const prefs = { provider: "codex", model: "gpt-x", effort: "medium" };
    expect(normalizePrefs(prefs, undefined)).toBe(prefs);
  });
});

describe("withProvider and withModel", () => {
  it("resets the model when the provider changes", () => {
    expect(withProvider({ provider: "claude", model: "opus", effort: "max" }, "codex")).toEqual({ provider: "codex", model: "", effort: "max" });
    const same = { provider: "claude", model: "opus", effort: "max" };
    expect(withProvider(same, "claude")).toBe(same);
  });

  it("keeps the effort only when the new model supports it", () => {
    expect(withModel({ provider: "codex", model: "", effort: "low" }, "gpt-x", status)).toEqual({ provider: "codex", model: "gpt-x", effort: "medium" });
    expect(withModel({ provider: "codex", model: "gpt-x", effort: "medium" }, "", status)).toEqual({ provider: "codex", model: "", effort: "low" });
    expect(withModel({ provider: "claude", model: "", effort: "max" }, "opus", status)).toEqual({ provider: "claude", model: "opus", effort: "max" });
  });
});

describe("composerOptions", () => {
  it("groups the available providers' models, named once known", () => {
    const opts = composerOptions(status, { provider: "codex", model: "", effort: "" }, false);
    expect(opts.groups).toEqual([
      {
        provider: "claude",
        label: "Claude Code",
        disabled: false,
        options: [
          { value: modelKey("claude", ""), label: "Claude Opus 5.5 (Default)" },
          { value: modelKey("claude", "opus"), label: "Opus" },
        ],
      },
      {
        provider: "codex",
        label: "Codex",
        disabled: false,
        options: [
          { value: modelKey("codex", ""), label: "Default model" },
          { value: modelKey("codex", "gpt-x"), label: "GPT-X" },
        ],
      },
    ]);
    expect(opts.model).toBe(modelKey("codex", ""));
    expect(opts.modelLabel).toBe("Default model");
    expect(opts.efforts).toEqual([
      { value: "low", label: "Low" },
      { value: "ultra", label: "Ultra" },
    ]);
  });

  it("shows the full name and the context window of the picked model", () => {
    const opts = composerOptions(status, { provider: "claude", model: "", effort: "low" }, false);
    expect(opts.modelLabel).toBe("Claude Opus 5.5");
    expect(opts.effortLabel).toBe("Low · 1M");
    expect(composerOptions(status, { provider: "claude", model: "", effort: "medium" }, false).effortLabel).toBe("Medium · 1M");
  });

  it("disables the other providers once the chat has started", () => {
    const opts = composerOptions(status, { provider: "claude", model: "", effort: "" }, true);
    expect(opts.groups.map((g) => [g.provider, g.disabled])).toEqual([["claude", false], ["codex", true]]);
  });

  it("is empty before the status loads", () => {
    expect(composerOptions(undefined, { provider: "", model: "", effort: "" }, false)).toEqual({
      groups: [],
      model: modelKey("", ""),
      modelLabel: "Model",
      efforts: [],
      effortLabel: "Effort",
    });
  });
});

describe("pickModel", () => {
  it("switches the provider and keeps a supported effort for another provider's model", () => {
    expect(pickModel({ provider: "claude", model: "opus", effort: "max" }, modelKey("codex", "gpt-x"), status)).toEqual({ provider: "codex", model: "gpt-x", effort: "medium" });
  });

  it("keeps the effort within a provider when the model supports it", () => {
    expect(pickModel({ provider: "claude", model: "", effort: "max" }, modelKey("claude", "opus"), status)).toEqual({ provider: "claude", model: "opus", effort: "max" });
  });
});

describe("chatPrefs", () => {
  const saved = { provider: "codex", model: "gpt-x", effort: "medium" };
  const meta = { id: "c1", title: "t", created_at: "", updated_at: "", agent: "claude", model: "opus", effort: "max", state: "idle" as const };

  it("uses the remembered prefs for a new chat", () => {
    expect(chatPrefs(undefined, undefined, saved)).toBe(saved);
  });

  it("restores what an existing chat last used", () => {
    expect(chatPrefs(meta, undefined, saved)).toEqual({ provider: "claude", model: "opus", effort: "max" });
  });

  it("prefers a pick made in the open chat", () => {
    const picked = { provider: "claude", model: "", effort: "low" };
    expect(chatPrefs(meta, picked, saved)).toBe(picked);
  });

  it("uses the remembered prefs for a chat that has no message yet", () => {
    expect(chatPrefs({ ...meta, agent: "", model: undefined, effort: undefined }, undefined, saved)).toBe(saved);
  });
});

describe("formatContextWindow", () => {
  it("rounds to K or M", () => {
    expect(formatContextWindow(200000)).toBe("200K");
    expect(formatContextWindow(258400)).toBe("258K");
    expect(formatContextWindow(1000000)).toBe("1M");
    expect(formatContextWindow(1500000)).toBe("1.5M");
    expect(formatContextWindow(0)).toBe("");
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

describe("chatListPollMs", () => {
  const c = (state: string) => ({ id: state, state });
  it("polls while any chat runs, pill or not, so issue chat icons stay current", () => {
    expect(chatListPollMs([c("idle"), c("running")])).toBe(2000);
    expect(chatListPollMs([c("idle"), c("interrupted")])).toBe(false);
    expect(chatListPollMs(undefined)).toBe(false);
  });
});

describe("popupLeft", () => {
  // A 460px popup in a 1600px window, 12px from the edges.
  it("centers the popup over its anchor", () => {
    expect(popupLeft(800, 460, 1600)).toBe(570);
  });
  it("keeps it inside the window near an edge", () => {
    expect(popupLeft(1580, 460, 1600)).toBe(1128);
    expect(popupLeft(40, 460, 1600)).toBe(12);
  });
  it("pins it to the left margin when the window is narrower than the popup", () => {
    expect(popupLeft(200, 460, 400)).toBe(12);
  });
});

describe("clampPopupSize", () => {
  const big = { width: 3000, height: 2000 };
  it("keeps the size between 360x400 and 600x900", () => {
    expect(clampPopupSize({ width: 100, height: 100 }, big)).toEqual({ width: 360, height: 400 });
    expect(clampPopupSize({ width: 999, height: 999 }, big)).toEqual({ width: 600, height: 900 });
    expect(clampPopupSize({ width: 500, height: 700 }, big)).toEqual({ width: 500, height: 700 });
  });
  it("never exceeds the window, minus its margins", () => {
    // 24px of side margins; 64px above and below for the status bar.
    expect(clampPopupSize({ width: 600, height: 900 }, { width: 500, height: 700 })).toEqual({ width: 476, height: 636 });
  });
});

describe("popup size storage", () => {
  function memory(): Storage {
    const m = new Map<string, string>();
    return { getItem: (k: string) => m.get(k) ?? null, setItem: (k: string, v: string) => void m.set(k, v) } as unknown as Storage;
  }
  it("defaults to 460x600 and round-trips a saved size", () => {
    const s = memory();
    expect(loadPopupSize(s)).toEqual(DEFAULT_POPUP_SIZE);
    expect(DEFAULT_POPUP_SIZE).toEqual({ width: 460, height: 600 });
    savePopupSize(s, { width: 520, height: 800 });
    expect(loadPopupSize(s)).toEqual({ width: 520, height: 800 });
    expect(loadPopupSize(null)).toEqual(DEFAULT_POPUP_SIZE);
  });
  it("ignores a malformed value", () => {
    const s = memory();
    s.setItem("xpo.agent.popupSize", '{"width":"x"}');
    expect(loadPopupSize(s)).toEqual(DEFAULT_POPUP_SIZE);
  });
});

describe("context chip", () => {
  const on = (id?: string) => nextViewed({ seq: 0 }, id);
  it("adds the viewed issue as pending context once it is opened", () => {
    expect(pendingContextIssue(on("xpo-1"), undefined, undefined)).toBe("xpo-1");
    expect(pendingContextIssue(on(undefined), undefined, undefined)).toBeUndefined();
  });
  it("clears it once sent or removed, until another issue is opened", () => {
    const a = on("xpo-1");
    expect(pendingContextIssue(a, a.seq, undefined)).toBeUndefined();
    const b = nextViewed(a, "xpo-2");
    expect(pendingContextIssue(b, a.seq, undefined)).toBe("xpo-2");
    const back = nextViewed(b, "xpo-1");
    expect(pendingContextIssue(back, b.seq, undefined)).toBe("xpo-1");
  });
  it("keeps the last opened issue on views without one", () => {
    const a = on("xpo-1");
    expect(nextViewed(a, undefined)).toBe(a);
    expect(nextViewed(a, "xpo-1")).toBe(a);
  });
  it("never adds the issue a Work Mode chat is bound to", () => {
    expect(pendingContextIssue(on("xpo-1"), undefined, "xpo-1")).toBeUndefined();
    expect(pendingContextIssue(on("xpo-2"), undefined, "xpo-1")).toBe("xpo-2");
  });
  it("records what each chat consumed, with null as the new chat", () => {
    let c = consumeContext({}, "a", 1);
    c = consumeContext(c, null, 2);
    expect(c).toEqual({ a: 1, "": 2 });
    expect(consumedFor(c, null)).toBe(2);
    expect(consumedFor(c, "b")).toBeUndefined();
  });
  it("hands the new chat's record to the chat once it exists", () => {
    expect(adoptNewChatContext({ "": 2, a: 1 }, "b")).toEqual({ a: 1, b: 2 });
    const c = { a: 1 };
    expect(adoptNewChatContext(c, "b")).toBe(c);
  });
});
