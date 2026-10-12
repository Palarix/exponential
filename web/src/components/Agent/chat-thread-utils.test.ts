import { describe, expect, it } from "vitest";
import type { ChatEvent } from "../../api/chat";
import { chatThreadReducer, emptyThread, failedToolCount, isRunning, splitTurn, toolSummary, turnSeconds, type ChatThread } from "./chat-thread-utils";

const T0 = "2026-10-12T10:00:00Z";

function ev(e: Partial<ChatEvent> & { type: ChatEvent["type"] }): ChatEvent {
  return { time: T0, ...e };
}

function reduce(events: ChatEvent[], start: ChatThread = emptyThread): ChatThread {
  return events.reduce((s, e) => chatThreadReducer(s, { type: "event", event: e }), start);
}

describe("chatThreadReducer", () => {
  it("starts a running turn on a user message", () => {
    const s = reduce([ev({ type: "user_message", seq: 1, text: "hi", context: { issue_id: "xpo-1" } })]);
    expect(s.turns).toHaveLength(1);
    expect(s.turns[0].user.text).toBe("hi");
    expect(s.turns[0].user.context?.issue_id).toBe("xpo-1");
    expect(s.turns[0].status).toBe("running");
    expect(isRunning(s)).toBe(true);
    expect(s.lastSeq).toBe(1);
  });

  it("appends text deltas to one live text step", () => {
    const s = reduce([
      ev({ type: "user_message", seq: 1, text: "hi" }),
      ev({ type: "text_delta", text: "Hel" }),
      ev({ type: "text_delta", text: "lo" }),
    ]);
    expect(s.turns[0].steps).toEqual([{ kind: "text", text: "Hello", live: true }]);
  });

  it("replaces the live text with the settled text", () => {
    const s = reduce([
      ev({ type: "user_message", seq: 1, text: "hi" }),
      ev({ type: "text_delta", text: "lo" }), // joined mid-block after a reload
      ev({ type: "assistant_text", seq: 2, text: "Hello" }),
    ]);
    expect(s.turns[0].steps).toEqual([{ kind: "text", text: "Hello", live: false }]);
  });

  it("tracks a tool from start to settled call", () => {
    let s = reduce([
      ev({ type: "user_message", seq: 1, text: "hi" }),
      ev({ type: "tool_start", name: "mcp__xpo__list" }),
      ev({ type: "tool_start", name: "mcp__xpo__list", arg: "log" }),
      ev({ type: "tool_start", name: "mcp__xpo__list", arg: "login" }),
    ]);
    expect(s.turns[0].steps).toEqual([{ kind: "tool", name: "mcp__xpo__list", arg: "login", done: false }]);

    s = reduce([ev({ type: "tool_call", seq: 2, name: "mcp__xpo__list", arg: "login", status: "done" })], s);
    expect(s.turns[0].steps).toEqual([{ kind: "tool", name: "mcp__xpo__list", arg: "login", done: true, status: "done" }]);
  });

  it("matches parallel calls of one tool by ID", () => {
    let s = reduce([
      ev({ type: "user_message", seq: 1, text: "hi" }),
      ev({ type: "tool_start", name: "Read", tool_id: "t-a", arg: "a.go" }),
      ev({ type: "tool_start", name: "Read", tool_id: "t-b" }),
      ev({ type: "tool_start", name: "Read", tool_id: "t-b", arg: "/etc/hosts" }),
    ]);
    expect(s.turns[0].steps.map((x) => x.kind === "tool" && x.arg)).toEqual(["a.go", "/etc/hosts"]);

    s = reduce([ev({ type: "tool_call", seq: 2, name: "Read", tool_id: "t-b", arg: "/etc/hosts", status: "denied", message: "not granted" })], s);
    expect(s.turns[0].steps).toEqual([
      { kind: "tool", id: "t-a", name: "Read", arg: "a.go", done: false },
      { kind: "tool", id: "t-b", name: "Read", arg: "/etc/hosts", done: true, status: "denied", message: "not granted" },
    ]);
  });

  it("counts failed and denied tool calls", () => {
    const s = reduce([
      ev({ type: "user_message", seq: 1, text: "hi" }),
      ev({ type: "tool_call", seq: 2, name: "Read", tool_id: "a", status: "denied", message: "no" }),
      ev({ type: "tool_call", seq: 3, name: "Bash", tool_id: "b", status: "error", message: "exit 1" }),
      ev({ type: "tool_call", seq: 4, name: "Grep", tool_id: "c", status: "done" }),
      ev({ type: "tool_call", seq: 5, name: "Glob", status: "done" }), // saved before tool results
    ]);
    expect(failedToolCount(s.turns[0])).toBe(2);
  });

  it("adds a settled tool call without a live start (replay)", () => {
    const s = reduce([
      ev({ type: "user_message", seq: 1, text: "hi" }),
      ev({ type: "tool_call", seq: 2, name: "Read", arg: "a.go", status: "done" }),
      ev({ type: "tool_call", seq: 3, name: "Read", arg: "b.go", status: "done" }),
    ]);
    expect(s.turns[0].steps.map((x) => x.kind === "tool" && x.arg)).toEqual(["a.go", "b.go"]);
  });

  it("ends turns as done, stopped, or error", () => {
    const done = reduce([ev({ type: "user_message", seq: 1, text: "a" }), ev({ type: "turn_done", seq: 2, duration_ms: 8200 })]);
    expect(done.turns[0].status).toBe("done");
    expect(done.turns[0].durationMs).toBe(8200);
    expect(isRunning(done)).toBe(false);

    const stopped = reduce([ev({ type: "user_message", seq: 1, text: "a" }), ev({ type: "turn_stopped", seq: 2 })]);
    expect(stopped.turns[0].status).toBe("stopped");

    const failed = reduce([ev({ type: "user_message", seq: 1, text: "a" }), ev({ type: "error", seq: 2, message: "boom" })]);
    expect(failed.turns[0].status).toBe("error");
    expect(failed.turns[0].error).toBe("boom");
  });

  it("marks unfinished tools done when the turn ends", () => {
    const s = reduce([
      ev({ type: "user_message", seq: 1, text: "a" }),
      ev({ type: "tool_start", name: "Grep" }),
      ev({ type: "turn_stopped", seq: 2 }),
    ]);
    expect(s.turns[0].steps).toEqual([{ kind: "tool", name: "Grep", arg: "", done: true }]);
  });

  it("ignores settled events it has already seen", () => {
    const first = [ev({ type: "user_message", seq: 1, text: "a" }), ev({ type: "assistant_text", seq: 2, text: "b" })];
    const s = reduce([...first, ...first]);
    expect(s.turns).toHaveLength(1);
    expect(s.turns[0].steps).toHaveLength(1);
  });

  it("thinking flags the turn until output arrives", () => {
    let s = reduce([ev({ type: "user_message", seq: 1, text: "a" }), ev({ type: "thinking" })]);
    expect(s.turns[0].thinking).toBe(true);
    s = reduce([ev({ type: "text_delta", text: "x" })], s);
    expect(s.turns[0].thinking).toBe(false);
  });

  it("marks a running turn interrupted", () => {
    const s = chatThreadReducer(reduce([ev({ type: "user_message", seq: 1, text: "a" })]), { type: "interrupted" });
    expect(s.turns[0].status).toBe("interrupted");
    expect(isRunning(s)).toBe(false);
  });

  it("ignores events before any user message", () => {
    const s = reduce([ev({ type: "text_delta", text: "x" }), ev({ type: "turn_done", seq: 1 })]);
    expect(s.turns).toHaveLength(0);
    expect(s.lastSeq).toBe(1);
  });

  it("resets", () => {
    const s = chatThreadReducer(reduce([ev({ type: "user_message", seq: 1, text: "a" })]), { type: "reset" });
    expect(s).toEqual(emptyThread);
  });
});

describe("splitTurn", () => {
  it("shows the text after the last tool as the answer", () => {
    const s = reduce([
      ev({ type: "user_message", seq: 1, text: "a" }),
      ev({ type: "assistant_text", seq: 2, text: "Let me check." }),
      ev({ type: "tool_call", seq: 3, name: "Read", arg: "x", status: "done" }),
      ev({ type: "assistant_text", seq: 4, text: "Found it." }),
    ]);
    const { work, answer } = splitTurn(s.turns[0]);
    expect(work.map((w) => w.kind)).toEqual(["text", "tool"]);
    expect(answer.map((a) => a.kind === "text" && a.text)).toEqual(["Found it."]);
  });

  it("puts everything in the answer when no tool ran", () => {
    const s = reduce([ev({ type: "user_message", seq: 1, text: "a" }), ev({ type: "assistant_text", seq: 2, text: "Hi" })]);
    const { work, answer } = splitTurn(s.turns[0]);
    expect(work).toEqual([]);
    expect(answer).toHaveLength(1);
  });
});

describe("turnSeconds", () => {
  it("uses the reported duration of a finished turn", () => {
    const s = reduce([ev({ type: "user_message", seq: 1, text: "a" }), ev({ type: "turn_done", seq: 2, duration_ms: 8400 })]);
    expect(turnSeconds(s.turns[0], Date.parse(T0) + 99_000)).toBe(8);
  });

  it("counts from the user message while running", () => {
    const s = reduce([ev({ type: "user_message", seq: 1, text: "a" })]);
    expect(turnSeconds(s.turns[0], Date.parse(T0) + 3_600)).toBe(3);
  });
});

describe("toolSummary", () => {
  it("describes code tools in words", () => {
    expect(toolSummary("Read", "/repo/internal/server/board.go")).toEqual({ text: "Read", detail: "board.go" });
    expect(toolSummary("Grep", '"estimate":\\s*\\{')).toEqual({ text: "Searched the code for", detail: '"estimate":\\s*\\{' });
    expect(toolSummary("Glob", "**/*.tsx")).toEqual({ text: "Found files matching", detail: "**/*.tsx" });
  });

  it("describes xpo tools in words", () => {
    expect(toolSummary("mcp__xpo__list", "")).toEqual({ text: "Listed issues", detail: "" });
    expect(toolSummary("mcp__xpo__list", "login")).toEqual({ text: "Listed issues matching", detail: "login" });
    expect(toolSummary("mcp__xpo__show", "xpo-abc123")).toEqual({ text: "Opened", detail: "xpo-abc123" });
    expect(toolSummary("mcp__xpo__add", "Login times out")).toEqual({ text: "Created issue", detail: "Login times out" });
    expect(toolSummary("mcp__xpo__update", "xpo-abc123")).toEqual({ text: "Updated", detail: "xpo-abc123" });
    expect(toolSummary("mcp__xpo__comment", "xpo-abc123")).toEqual({ text: "Commented on", detail: "xpo-abc123" });
  });

  it("falls back to the tool name", () => {
    expect(toolSummary("mcp__other__thing", "x")).toEqual({ text: "other thing", detail: "x" });
    expect(toolSummary("WebFetch", "https://a.b")).toEqual({ text: "WebFetch", detail: "https://a.b" });
  });

  it("collapses whitespace in long details", () => {
    expect(toolSummary("Grep", "a\n   b").detail).toBe("a b");
  });
});
