import { describe, expect, it } from "vitest";
import type { ChatEvent } from "../../api/chat";
import { approvalArgs, approvalOutcomeLabel, approvalTitle, backgroundBanner, chatThreadReducer, emptyPicks, formatDuration, groupSummary, isEmptyThread, isQuestion, modeDividerText, failedSteps, failureReason, parseQuestions, pickOption, questionAdvance, questionAnswered, questionAnswerText, questionAnswers, reviewStep, questionKeyAction, questionTabLabel, autoAdvanceTo, setOtherText, stepIcon, stepSummary, emptyThread, isRunning, pendingApprovals, toggleOther, toolLabel, timelineSteps, toolSummary, turnSeconds, turnSegments, type ChatThread, type ToolIconId, type ToolStep } from "./chat-thread-utils";

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

  it("starts a running turn with no user message when the board continues", () => {
    const s = reduce([
      ev({ type: "user_message", seq: 1, text: "start xpo-1" }),
      ev({ type: "turn_done", seq: 2, duration_ms: 1000 }),
      ev({ type: "mode_changed", seq: 3, mode: "work", issue_id: "xpo-1" }),
      ev({ type: "turn_continued", seq: 4, issue_id: "xpo-1" }),
    ]);
    expect(s.turns).toHaveLength(2);
    expect(s.turns[1].continued).toBe(true);
    expect(s.turns[1].status).toBe("running");
    expect(s.dividers).toEqual([{ at: 1, mode: "work", issueId: "xpo-1", message: undefined }]);
    expect(isRunning(s)).toBe(true);
  });

  it("starts a running turn with no user message when the agent starts one", () => {
    const s = reduce([
      ev({ type: "user_message", seq: 1, text: "serve it" }),
      ev({ type: "turn_done", seq: 2, duration_ms: 1000 }),
      ev({ type: "agent_turn", seq: 3, text: "Serve the board finished" }),
      ev({ type: "assistant_text", seq: 4, text: "The server stopped." }),
    ]);
    expect(s.turns).toHaveLength(2);
    expect(s.turns[1].byAgent).toBe(true);
    expect(s.turns[1].user.text).toBe("Serve the board finished");
    expect(s.turns[1].status).toBe("running");
    expect(isRunning(s)).toBe(true);
  });

  it("tracks the background tasks apart from the turns", () => {
    const tasks = [{ id: "bg-1", kind: "monitor" as const, description: "Serve the board" }];
    let s = reduce([
      ev({ type: "user_message", seq: 1, text: "serve it" }),
      ev({ type: "background_tasks", tasks }),
      ev({ type: "turn_done", seq: 2, duration_ms: 1000 }),
    ]);
    expect(s.background).toEqual(tasks);
    expect(isRunning(s)).toBe(false);
    s = reduce([ev({ type: "background_tasks" })], s);
    expect(s.background).toEqual([]);
    s = chatThreadReducer(s, { type: "background", tasks });
    expect(s.background).toEqual(tasks);
    expect(chatThreadReducer(s, { type: "reset" }).background).toEqual([]);
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
    expect(failedSteps(s.turns[0]).length).toBe(2);
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

describe("turnSegments", () => {
  it("splits a turn into messages and groups of tool calls, in order", () => {
    const s = reduce([
      ev({ type: "user_message", seq: 1, text: "a" }),
      ev({ type: "assistant_text", seq: 2, text: "Let me check." }),
      ev({ type: "tool_call", seq: 3, name: "Read", tool_id: "r", arg: "x", status: "done" }),
      ev({ type: "tool_call", seq: 4, name: "Grep", tool_id: "g", arg: "y", status: "done" }),
      ev({ type: "assistant_text", seq: 5, text: "Now the tests." }),
      ev({ type: "tool_call", seq: 6, name: "Bash", tool_id: "b", arg: "z", status: "done" }),
      ev({ type: "assistant_text", seq: 7, text: "Found it." }),
    ]);
    const segs = turnSegments(s.turns[0]);
    expect(segs.map((g) => (g.kind === "text" ? g.text : g.kind === "tools" ? g.steps.map((st) => st.id).join(",") : g.kind))).toEqual(["Let me check.", "r,g", "Now the tests.", "b", "Found it."]);
  });

  it("joins text steps that follow each other into one message", () => {
    const s = reduce([
      ev({ type: "user_message", seq: 1, text: "a" }),
      ev({ type: "assistant_text", seq: 2, text: "One." }),
      ev({ type: "text_delta", text: "Tw" }),
    ]);
    expect(turnSegments(s.turns[0])).toEqual([{ kind: "text", text: "One.\n\nTw", live: true }]);
  });

  it("has no segments before the agent says or does anything", () => {
    expect(turnSegments(reduce([ev({ type: "user_message", seq: 1, text: "a" })]).turns[0])).toEqual([]);
  });
});

describe("groupSummary", () => {
  const call = (status: ToolStep["status"]): ToolStep => ({ kind: "tool", name: "Read", arg: "", done: true, status });

  it("counts the calls, and the failed ones", () => {
    expect(groupSummary([call("done")])).toEqual({ text: "1 action", failed: null });
    expect(groupSummary([call("done"), call("error"), call("denied")])).toEqual({ text: "3 actions", failed: "2 failed" });
  });

  it("does not count a call the user denied as failed", () => {
    expect(groupSummary([{ ...call("denied"), approval: { id: "a", decision: "deny" } }]).failed).toBeNull();
  });
});

describe("timelineSteps", () => {
  const a: ToolStep = { kind: "tool", id: "a", name: "Read", arg: "", done: true, status: "done" };
  const b: ToolStep = { kind: "tool", id: "b", name: "Grep", arg: "", done: false };

  it("lists every call of a finished group", () => {
    expect(timelineSteps([a, b], false)).toEqual([a, b]);
  });

  it("leaves out the call the active line shows", () => {
    expect(timelineSteps([a, b], true)).toEqual([a]);
    expect(timelineSteps([b], true)).toEqual([]);
  });
});

describe("formatDuration", () => {
  it("reads seconds, then minutes and seconds", () => {
    expect(formatDuration(0)).toBe("0s");
    expect(formatDuration(12)).toBe("12s");
    expect(formatDuration(60)).toBe("1m");
    expect(formatDuration(84)).toBe("1m 24s");
    expect(formatDuration(3725)).toBe("62m 5s");
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

  it("names the skill the agent loaded", () => {
    expect(toolSummary("Skill", "xpo")).toEqual({ text: "Loaded the skill", detail: "xpo" });
    expect(approvalTitle("Skill", "xpo")).toEqual({ action: "Load the skill", detail: "xpo" });
  });

  it("describes xpo tools in words", () => {
    expect(toolSummary("mcp__xpo__list", "")).toEqual({ text: "Listed issues", detail: "" });
    expect(toolSummary("mcp__xpo__list", "login")).toEqual({ text: "Listed issues matching", detail: "login" });
    expect(toolSummary("mcp__xpo__show", "xpo-abc123")).toEqual({ text: "Opened", detail: "xpo-abc123" });
    expect(toolSummary("mcp__xpo__add", "Login times out")).toEqual({ text: "Created issue", detail: "Login times out" });
    expect(toolSummary("mcp__xpo__update", "xpo-abc123")).toEqual({ text: "Updated", detail: "xpo-abc123" });
    expect(toolSummary("mcp__xpo__comment", "xpo-abc123")).toEqual({ text: "Commented on", detail: "xpo-abc123" });
  });

  it("describes the git tools in words", () => {
    expect(toolSummary("mcp__xpo__git_status", "xpo-abc123")).toEqual({ text: "Checked the git status of", detail: "xpo-abc123" });
    expect(toolSummary("mcp__xpo__git_diff", "xpo-abc123")).toEqual({ text: "Read the diff of", detail: "xpo-abc123" });
    expect(toolSummary("mcp__xpo__git_log", "")).toEqual({ text: "Listed commits", detail: "" });
    expect(toolSummary("mcp__xpo__git_log", "xpo-abc123")).toEqual({ text: "Listed commits of", detail: "xpo-abc123" });
    expect(toolSummary("mcp__xpo__git_show", "0123456789abcdef0123456789abcdef01234567")).toEqual({ text: "Read commit", detail: "0123456789ab" });
    expect(toolSummary("mcp__xpo__code_history", "internal/server/board.go")).toEqual({ text: "Traced the history of", detail: "internal/server/board.go" });
  });

  it("names the operation of the spec, walkthrough and artifact tools", () => {
    expect(toolSummary("mcp__xpo__spec", "write xpo-1")).toEqual({ text: "Wrote the spec of", detail: "xpo-1" });
    expect(toolSummary("mcp__xpo__spec", "read xpo-1")).toEqual({ text: "Read the spec of", detail: "xpo-1" });
    expect(toolSummary("mcp__xpo__walkthrough", "write xpo-1")).toEqual({ text: "Wrote the walkthrough of", detail: "xpo-1" });
    expect(toolSummary("mcp__xpo__walkthrough", "delete xpo-1")).toEqual({ text: "Deleted the walkthrough of", detail: "xpo-1" });
    expect(toolSummary("mcp__xpo__artifact", "add xpo-1")).toEqual({ text: "Added an artifact to", detail: "xpo-1" });
    expect(toolSummary("mcp__xpo__artifact", "list xpo-1")).toEqual({ text: "Listed the artifacts of", detail: "xpo-1" });
    expect(approvalTitle("mcp__xpo__spec", "write xpo-1")).toEqual({ action: "Write the spec of", detail: "xpo-1" });
    expect(approvalTitle("mcp__xpo__artifact", "delete xpo-1")).toEqual({ action: "Delete an artifact of", detail: "xpo-1" });
    expect(stepSummary({ kind: "tool", name: "mcp__xpo__spec", arg: "write xpo-1", done: true, status: "error" })).toEqual({ text: "Write the spec of", detail: "xpo-1" });
  });

  it("keeps the old wording for chats saved without an operation", () => {
    expect(toolSummary("mcp__xpo__spec", "xpo-1")).toEqual({ text: "Read the spec of", detail: "xpo-1" });
    expect(toolSummary("mcp__xpo__artifact", "")).toEqual({ text: "Read an artifact of", detail: "" });
    // Only these tools carry an operation: another tool's first word stays.
    expect(toolSummary("mcp__xpo__add", "write docs")).toEqual({ text: "Created issue", detail: "write docs" });
  });

  it("falls back to the tool name", () => {
    expect(toolSummary("mcp__other__thing", "x")).toEqual({ text: "other thing", detail: "x" });
    expect(toolSummary("NotebookEdit", "a.ipynb")).toEqual({ text: "NotebookEdit", detail: "a.ipynb" });
  });

  // The web, LSP, task-list and subagent tools (xpo-80bf05).
  it("describes the lookup, task and subagent tools in words", () => {
    expect(toolSummary("WebSearch", "go 1.26 release")).toEqual({ text: "Searched the web for", detail: "go 1.26 release" });
    expect(toolSummary("WebSearch", "")).toEqual({ text: "Searched the web", detail: "" });
    expect(toolSummary("WebFetch", "https://go.dev")).toEqual({ text: "Fetched", detail: "https://go.dev" });
    expect(toolSummary("WebFetch", "")).toEqual({ text: "Fetched a page", detail: "" });
    expect(toolSummary("LSP", "/repo/internal/chat/profile.go")).toEqual({ text: "Looked up symbols in", detail: "profile.go" });
    expect(toolSummary("LSP", "")).toEqual({ text: "Looked up symbols", detail: "" });
    expect(toolSummary("TaskCreate", "Write the tests")).toEqual({ text: "Added the to-do", detail: "Write the tests" });
    expect(toolSummary("TaskCreate", "")).toEqual({ text: "Added a to-do", detail: "" });
    expect(toolSummary("TaskUpdate", "3")).toEqual({ text: "Updated to-do", detail: "3" });
    expect(toolSummary("TaskUpdate", "")).toEqual({ text: "Updated a to-do", detail: "" });
    expect(toolSummary("TaskList", "")).toEqual({ text: "Listed the to-dos", detail: "" });
    expect(toolSummary("TaskGet", "3")).toEqual({ text: "Read to-do", detail: "3" });
    expect(toolSummary("TaskGet", "")).toEqual({ text: "Read a to-do", detail: "" });
    expect(toolSummary("Agent", "Count files")).toEqual({ text: "Delegated", detail: "Count files" });
    expect(toolSummary("Agent", "")).toEqual({ text: "Delegated to a subagent", detail: "" });
    // Older Claude Code versions name the subagent tool Task.
    expect(toolSummary("Task", "Count files")).toEqual({ text: "Delegated", detail: "Count files" });
  });

  it("collapses whitespace in long details", () => {
    expect(toolSummary("Grep", "a\n   b").detail).toBe("a b");
  });
});

describe("stepIcon", () => {
  const tool = (name: string, arg = "", extra: Partial<ToolStep> = {}): ToolStep => ({ kind: "tool", name, arg, done: true, status: "done", ...extra });

  it("gives every known tool an icon, labelled with its verb", () => {
    // Mostly one icon per kind of action (look, change, search); some xpo tools keep their own.
    const icons: [string, ToolIconId][] = [
      ["Read", "read"], ["mcp__xpo__show", "read"], ["mcp__xpo__history", "read"], ["mcp__xpo__git_log", "read"], ["mcp__xpo__git_show", "read"], ["mcp__xpo__code_history", "read"],
      ["Edit", "edit"], ["Write", "edit"], ["mcp__xpo__update", "edit"],
      ["Grep", "search"], ["Glob", "search"], ["mcp__xpo__list", "search"], ["mcp__xpo__rationale", "search"],
      ["Bash", "shell"], ["Skill", "skill"], ["mcp__xpo__start", "start"], ["mcp__xpo__merge", "merge"],
      ["mcp__xpo__add", "add"], ["mcp__xpo__comment", "comment"], ["mcp__xpo__link", "link"], ["mcp__xpo__unlink", "unlink"],
      ["mcp__xpo__artifact", "artifact"], ["mcp__xpo__git_status", "git_status"], ["mcp__xpo__git_diff", "git_diff"],
      ["mcp__xpo__spec", "spec"], ["mcp__xpo__walkthrough", "walkthrough"],
      ["WebSearch", "search"], ["WebFetch", "read"], ["LSP", "search"],
      ["TaskCreate", "add"], ["TaskUpdate", "edit"], ["TaskList", "read"], ["TaskGet", "read"],
      ["Agent", "agent"], ["Task", "agent"], ["AskUserQuestion", "question"],
    ];
    for (const [name, icon] of icons) {
      const got = stepIcon(tool(name, "x"));
      expect(got.icon, name).toBe(icon);
      expect(got.name, name).toBeUndefined();
    }
    expect(stepIcon(tool("Read", "/r/board.go"))).toEqual({ icon: "read", label: "Read", detail: "board.go" });
    expect(stepIcon(tool("Grep", "TODO"))).toEqual({ icon: "search", label: "Searched the code for", detail: "TODO" });
    expect(stepIcon(tool("mcp__xpo__list"))).toEqual({ icon: "search", label: "Listed issues", detail: "" });
  });

  it("labels a shell call with its verb, not the tool name", () => {
    expect(stepIcon(tool("Bash", "Run the tests"))).toEqual({ icon: "shell", label: "Ran a shell command", detail: "Run the tests" });
    expect(toolSummary("Bash", "Run the tests")).toEqual({ text: "Ran a shell command", detail: "Run the tests" });
  });

  it("labels a call that did not run with what was asked", () => {
    expect(stepIcon(tool("mcp__xpo__merge", "upe-1", { status: "error" }))).toEqual({ icon: "merge", label: "Merge", detail: "upe-1" });
    expect(stepIcon(tool("Bash", "rm", { status: "denied" }))).toEqual({ icon: "shell", label: "Run a shell command", detail: "rm" });
    expect(stepIcon(tool("mcp__xpo__merge", "upe-1", { approval: { id: "a", decision: "deny" } })).label).toBe("Merge");
  });

  it("gives unknown tools a generic icon and their name", () => {
    expect(stepIcon(tool("NotebookEdit", "a.ipynb"))).toEqual({ icon: "tool", label: "NotebookEdit", name: "NotebookEdit", detail: "a.ipynb" });
    expect(stepIcon(tool("mcp__other__thing", "x", { status: "error" }))).toEqual({ icon: "tool", label: "other thing", name: "other thing", detail: "x" });
  });
});

describe("approvals", () => {
  const start = [
    ev({ type: "user_message", seq: 1, text: "merge it" }),
    ev({ type: "tool_start", name: "mcp__xpo__merge", tool_id: "t1", arg: "xpo-1" }),
  ];
  const requested = ev({ type: "approval_requested", seq: 2, approval_id: "a1", name: "mcp__xpo__merge", arg: "xpo-1", tool_id: "t1", input: '{"id":"xpo-1"}' });

  it("attaches a request to its tool step and lists it as pending", () => {
    const s = reduce([...start, requested]);
    const turn = s.turns[0];
    expect(turn.steps).toHaveLength(1);
    expect(turn.steps[0]).toMatchObject({ kind: "tool", id: "t1", approval: { id: "a1", input: '{"id":"xpo-1"}' } });
    expect(pendingApprovals(turn).map((st) => st.approval.id)).toEqual(["a1"]);
  });

  it("records the decision and keeps it when the tool call settles", () => {
    const s = reduce([
      ...start,
      requested,
      ev({ type: "approval_resolved", seq: 3, approval_id: "a1", decision: "allow" }),
      ev({ type: "tool_call", seq: 4, name: "mcp__xpo__merge", tool_id: "t1", arg: "xpo-1", status: "done" }),
    ]);
    const turn = s.turns[0];
    expect(turn.steps[0]).toMatchObject({ kind: "tool", done: true, status: "done", approval: { id: "a1", decision: "allow" } });
    expect(pendingApprovals(turn)).toEqual([]);
  });

  it("adds a step of its own for a request without a tool ID", () => {
    const s = reduce([
      ev({ type: "user_message", seq: 1, text: "x" }),
      ev({ type: "approval_requested", seq: 2, approval_id: "a2", name: "mcp__xpo__merge", arg: "xpo-1" }),
    ]);
    expect(s.turns[0].steps).toEqual([
      { kind: "tool", id: undefined, name: "mcp__xpo__merge", arg: "xpo-1", done: false, approval: { id: "a2", input: undefined } },
    ]);
  });

  it("rebuilds a settled request on reload, before its tool call settles", () => {
    // Settled order on reload: the request settles before the tool call.
    const s = reduce([
      ev({ type: "user_message", seq: 1, text: "merge it" }),
      requested,
      ev({ type: "approval_resolved", seq: 3, approval_id: "a1", decision: "deny" }),
      ev({ type: "tool_call", seq: 4, name: "mcp__xpo__merge", tool_id: "t1", arg: "xpo-1", status: "denied", message: "declined" }),
    ]);
    expect(s.turns[0].steps).toHaveLength(1);
    expect(s.turns[0].steps[0]).toMatchObject({ status: "denied", approval: { id: "a1", decision: "deny" } });
  });

  it("expires open requests when the turn ends or is interrupted", () => {
    const done = reduce([...start, requested, ev({ type: "turn_stopped", seq: 3 })]);
    expect(done.turns[0].steps[0]).toMatchObject({ approval: { id: "a1", decision: "expired" } });
    expect(pendingApprovals(done.turns[0])).toEqual([]);

    const interrupted = chatThreadReducer(reduce([...start, requested]), { type: "interrupted" });
    expect(interrupted.turns[0].steps[0]).toMatchObject({ approval: { decision: "expired" } });
  });

  it("keeps a cancelled decision when the turn stops", () => {
    const s = reduce([
      ...start,
      requested,
      ev({ type: "approval_resolved", seq: 3, approval_id: "a1", decision: "cancelled" }),
      ev({ type: "turn_stopped", seq: 4 }),
    ]);
    expect(s.turns[0].steps[0]).toMatchObject({ approval: { decision: "cancelled" } });
  });

  it("marks an approval expired on request", () => {
    const s = chatThreadReducer(reduce([...start, requested]), { type: "approval_expired", id: "a1" });
    expect(pendingApprovals(s.turns[0])).toEqual([]);
    expect(s.turns[0].steps[0]).toMatchObject({ approval: { decision: "expired" } });
  });
});

describe("approval labels", () => {
  it("names the tool as the user knows it", () => {
    expect(toolLabel("mcp__xpo__merge")).toBe("xpo merge");
    expect(toolLabel("Bash")).toBe("Bash");
  });

  it("says how a card ended", () => {
    expect(approvalOutcomeLabel("allow")).toBe("Allowed once");
    expect(approvalOutcomeLabel("allow_session")).toBe("Allowed for this chat");
    expect(approvalOutcomeLabel("deny")).toBe("Denied");
    expect(approvalOutcomeLabel("cancelled")).toBe("Cancelled");
    expect(approvalOutcomeLabel("expired")).toBe("Expired");
  });
});

describe("failedToolCount with approvals", () => {
  it("does not count calls the user denied", () => {
    const s = reduce([
      ev({ type: "user_message", seq: 1, text: "x" }),
      ev({ type: "approval_requested", seq: 2, approval_id: "a1", name: "mcp__xpo__merge", tool_id: "t1" }),
      ev({ type: "approval_resolved", seq: 3, approval_id: "a1", decision: "deny" }),
      ev({ type: "tool_call", seq: 4, name: "mcp__xpo__merge", tool_id: "t1", status: "denied", message: "declined" }),
      ev({ type: "tool_call", seq: 5, name: "Read", tool_id: "t2", status: "error", message: "no such file" }),
    ]);
    expect(failedSteps(s.turns[0]).length).toBe(1);
  });
});

describe("approvalTitle", () => {
  it("says what the call will do", () => {
    expect(approvalTitle("mcp__xpo__merge", "upe-1")).toEqual({ action: "Merge", detail: "upe-1" });
    expect(approvalTitle("Edit", "/repo/main.go")).toEqual({ action: "Edit", detail: "main.go" });
    expect(approvalTitle("Read", "/etc/hosts")).toEqual({ action: "Read", detail: "hosts" });
  });

  it("names a shell command without echoing its description as an object", () => {
    expect(approvalTitle("Bash", "Write hi into out.txt")).toEqual({ action: "Run a shell command", detail: "Write hi into out.txt" });
  });

  it("falls back to the tool name", () => {
    expect(approvalTitle("mcp__xpo__frobnicate", "")).toEqual({ action: "Use xpo frobnicate", detail: "" });
    expect(approvalTitle("NotebookEdit", "a.ipynb")).toEqual({ action: "Use NotebookEdit", detail: "a.ipynb" });
  });

  // WebFetch asks in both modes (xpo-80bf05).
  it("names a page fetch", () => {
    expect(approvalTitle("WebFetch", "https://x.dev")).toEqual({ action: "Fetch", detail: "https://x.dev" });
    expect(approvalTitle("WebFetch", "")).toEqual({ action: "Fetch a page", detail: "" });
  });
});

describe("stepSummary", () => {
  it("reads a finished call in the past tense", () => {
    expect(stepSummary({ kind: "tool", name: "mcp__xpo__merge", arg: "upe-1", done: true, status: "done", approval: { id: "a", decision: "allow" } })).toEqual({ text: "Merged", detail: "upe-1" });
  });

  it("reads a call that never ran as what was asked", () => {
    for (const decision of ["deny", "cancelled", "expired"] as const) {
      expect(stepSummary({ kind: "tool", name: "mcp__xpo__merge", arg: "upe-1", done: true, status: "denied", approval: { id: "a", decision } })).toEqual({ text: "Merge", detail: "upe-1" });
    }
  });
});

describe("approvalArgs", () => {
  it("shows a shell command as code", () => {
    expect(approvalArgs("Bash", '{"command":"rm -rf build","description":"Clean the build"}', "Clean the build")).toEqual({ command: "rm -rf build", fields: [] });
  });

  it("lists only the arguments the headline does not show", () => {
    expect(approvalArgs("mcp__xpo__merge", '{"id":"upe-1"}', "upe-1")).toEqual({ fields: [] });
    expect(approvalArgs("mcp__xpo__update", '{"id":"upe-1","status":"DONE","priority":2,"labels":["bug","ui"]}', "upe-1")).toEqual({
      fields: [
        ["status", "DONE"],
        ["priority", "2"],
        ["labels", "bug, ui"],
      ],
    });
  });

  it("drops empty values and survives input that is not JSON", () => {
    expect(approvalArgs("Edit", '{"file_path":"/r/a.go","old_string":"","replace_all":false}', "/r/a.go")).toEqual({ fields: [["replace_all", "false"]] });
    expect(approvalArgs("Edit", "not json", "x")).toEqual({ fields: [] });
    expect(approvalArgs("Edit", undefined, "x")).toEqual({ fields: [] });
  });
});

describe("stepSummary for failed calls", () => {
  it("reads a failed call as what was asked", () => {
    expect(stepSummary({ kind: "tool", name: "mcp__xpo__merge", arg: "upe-1", done: true, status: "error", approval: { id: "a", decision: "allow" } })).toEqual({ text: "Merge", detail: "upe-1" });
    expect(stepSummary({ kind: "tool", name: "Grep", arg: "TODO", done: true, status: "error" })).toEqual({ text: "Search the code for", detail: "TODO" });
    expect(stepSummary({ kind: "tool", name: "mcp__xpo__list", arg: "", done: true, status: "denied" })).toEqual({ text: "List issues", detail: "" });
    expect(stepSummary({ kind: "tool", name: "Read", arg: "/r/board.go", done: true, status: "error" })).toEqual({ text: "Read", detail: "board.go" });
  });

  it("keeps the tool name for a failed call of an unknown tool", () => {
    expect(stepSummary({ kind: "tool", name: "mcp__xpo__frobnicate", arg: "x", done: true, status: "error" })).toEqual({ text: "xpo frobnicate", detail: "x" });
  });

  it("keeps the past tense for a call that ran", () => {
    expect(stepSummary({ kind: "tool", name: "Grep", arg: "TODO", done: true, status: "done" })).toEqual({ text: "Searched the code for", detail: "TODO" });
  });
});

describe("failedSteps", () => {
  it("lists the calls that failed, in order, without the ones the user denied", () => {
    const s = reduce([
      ev({ type: "user_message", seq: 1, text: "x" }),
      ev({ type: "tool_call", seq: 2, name: "Read", tool_id: "t1", arg: "a.go", status: "error", message: "no such file" }),
      ev({ type: "approval_requested", seq: 3, approval_id: "a1", name: "mcp__xpo__merge", tool_id: "t2" }),
      ev({ type: "approval_resolved", seq: 4, approval_id: "a1", decision: "deny" }),
      ev({ type: "tool_call", seq: 5, name: "mcp__xpo__merge", tool_id: "t2", status: "denied" }),
      ev({ type: "tool_call", seq: 6, name: "Grep", tool_id: "t3", arg: "x", status: "done" }),
      ev({ type: "tool_call", seq: 7, name: "mcp__xpo__show", tool_id: "t4", arg: "upe-1", status: "error", message: "not found" }),
    ]);
    expect(failedSteps(s.turns[0]).map((st) => st.id)).toEqual(["t1", "t4"]);
    expect(failedSteps(s.turns[0]).length).toBe(2);
  });
});

describe("failureReason", () => {
  it("labels the reason by outcome", () => {
    expect(failureReason({ kind: "tool", name: "Read", arg: "", done: true, status: "error", message: "no such file" })).toEqual({ label: "failed", text: "no such file", more: false });
    expect(failureReason({ kind: "tool", name: "Bash", arg: "", done: true, status: "denied", message: "not allowed" })).toEqual({ label: "denied", text: "not allowed", more: false });
  });

  it("keeps the first line and says when there is more", () => {
    const message = "uncommitted changes — commit or discard before merging:\n  - .gitignore\nDo NOT stash";
    expect(failureReason({ kind: "tool", name: "mcp__xpo__merge", arg: "", done: true, status: "error", message })).toEqual({
      label: "failed",
      text: "uncommitted changes — commit or discard before merging:",
      more: true,
    });
  });

  it("skips blank leading lines and handles a missing message", () => {
    expect(failureReason({ kind: "tool", name: "Read", arg: "", done: true, status: "error", message: "\n\n  boom  " })).toEqual({ label: "failed", text: "boom", more: false });
    expect(failureReason({ kind: "tool", name: "Read", arg: "", done: true, status: "error" })).toEqual({ label: "failed", text: "", more: false });
  });
});

describe("mode dividers", () => {
  it("records a divider before the first turn, as Start Work makes it", () => {
    const s = reduce([ev({ type: "mode_changed", seq: 1, mode: "work", issue_id: "xpo-1" })]);
    expect(s.turns).toHaveLength(0);
    expect(s.dividers).toEqual([{ at: 0, mode: "work", issueId: "xpo-1", message: undefined }]);
    expect(isEmptyThread(s)).toBe(false);
    expect(isEmptyThread(emptyThread)).toBe(true);
  });

  it("records a divider after a finished turn without changing the turn", () => {
    const s = reduce([
      ev({ type: "user_message", seq: 1, text: "merge it" }),
      ev({ type: "turn_done", seq: 2, duration_ms: 1000 }),
      ev({ type: "mode_changed", seq: 3, mode: "chat", issue_id: "xpo-1" }),
    ]);
    expect(s.turns[0].status).toBe("done");
    expect(s.dividers).toEqual([{ at: 1, mode: "chat", issueId: "xpo-1", message: undefined }]);
  });

  it("ignores a replayed divider", () => {
    const e = ev({ type: "mode_changed", seq: 1, mode: "work", issue_id: "xpo-1" });
    expect(reduce([e, e]).dividers).toHaveLength(1);
  });

  it("words each divider", () => {
    expect(modeDividerText({ at: 0, mode: "work", issueId: "xpo-1" })).toBe("Entered Work Mode on xpo-1");
    expect(modeDividerText({ at: 0, mode: "chat", issueId: "xpo-1" })).toBe("Entered Chat Mode");
    expect(modeDividerText({ at: 0, mode: "chat", issueId: "xpo-1", message: "The worktree of xpo-1 is gone." })).toBe("The worktree of xpo-1 is gone.");
  });
});

describe("start approval", () => {
  it("asks to enter Work Mode", () => {
    expect(approvalTitle("mcp__xpo__start", "xpo-1")).toEqual({ action: "Enter Work Mode on", detail: "xpo-1" });
  });
});

describe("backgroundBanner", () => {
  const monitor = { id: "m", kind: "monitor" as const };
  const agent = { id: "a", kind: "agent" as const };
  it("counts the watch loops when no agent works", () => {
    expect(backgroundBanner([monitor])).toBe("Monitoring 1 task");
    expect(backgroundBanner([monitor, { ...monitor, id: "n" }])).toBe("Monitoring 2 tasks");
  });
  it("counts agents when any agent works", () => {
    expect(backgroundBanner([agent])).toBe("1 agent working");
    expect(backgroundBanner([agent, monitor, { ...agent, id: "b" }])).toBe("2 agents working");
  });
  it("is empty without tasks", () => {
    expect(backgroundBanner([])).toBeUndefined();
  });
});

// AskUserQuestion cards (xpo-447687).
const QUESTIONS_INPUT = JSON.stringify({
  questions: [
    { question: "Which color?", header: "Color", options: [{ label: "Red", description: "Warm" }, { label: "Blue", description: "Cool" }], multiSelect: false },
    { question: "Which pets?", header: "Pets", options: [{ label: "Cat", description: "" }, { label: "Dog", description: "" }, { label: "Fish", description: "" }], multiSelect: true },
  ],
});

describe("question cards", () => {
  const asked = [
    ev({ type: "user_message", seq: 1, text: "ask me" }),
    ev({ type: "tool_start", name: "AskUserQuestion", tool_id: "t1", arg: "Which color?" }),
    ev({ type: "approval_requested", seq: 2, approval_id: "a1", name: "AskUserQuestion", arg: "Which color?", tool_id: "t1", input: QUESTIONS_INPUT }),
  ];
  const answers = { "Which color?": "Red", "Which pets?": "Cat, Dog" };
  const answered = [
    ...asked,
    ev({ type: "approval_resolved", seq: 3, approval_id: "a1", decision: "answer", answers }),
    ev({ type: "tool_call", seq: 4, name: "AskUserQuestion", tool_id: "t1", arg: "Which color?", status: "done" }),
  ];

  it("records the answers on the step, and keeps them when the call settles", () => {
    const s = reduce(answered);
    expect(s.turns[0].steps[0]).toMatchObject({ approval: { id: "a1", decision: "answer", answers } });
    expect(pendingApprovals(s.turns[0])).toEqual([]);
    expect(isQuestion(s.turns[0].steps[0] as ToolStep)).toBe(true);
  });

  it("gives an answered question a segment of its own, in place", () => {
    const s = reduce([
      ev({ type: "user_message", seq: 1, text: "ask me" }),
      ev({ type: "tool_call", seq: 2, name: "Read", tool_id: "r", arg: "x", status: "done" }),
      ...answered.slice(1).map((e) => (e.seq ? { ...e, seq: e.seq + 1 } : e)),
      ev({ type: "tool_call", seq: 6, name: "Grep", tool_id: "g", arg: "y", status: "done" }),
    ]);
    const kinds = turnSegments(s.turns[0]).map((g) => (g.kind === "tools" ? g.steps.map((st) => st.id).join(",") : g.kind));
    expect(kinds).toEqual(["r", "question", "g"]);
  });

  it("keeps an open or skipped question in its group", () => {
    const open = reduce(asked);
    expect(turnSegments(open.turns[0]).map((g) => g.kind)).toEqual(["tools"]);
    const skipped = reduce([...asked, ev({ type: "approval_resolved", seq: 3, approval_id: "a1", decision: "deny" })]);
    expect(turnSegments(skipped.turns[0]).map((g) => g.kind)).toEqual(["tools"]);
  });

  it("reads as a question asked, answered or skipped", () => {
    const step = (decision?: "answer" | "deny"): ToolStep => ({ kind: "tool", id: "t1", name: "AskUserQuestion", arg: "Which color?", done: true, status: decision === "deny" ? "denied" : "done", approval: { id: "a1", decision } });
    expect(stepSummary(step("answer"))).toEqual({ text: "Asked", detail: "Which color?" });
    expect(stepSummary(step("deny"))).toEqual({ text: "Ask", detail: "Which color?" });
    expect(approvalOutcomeLabel("deny", true)).toBe("Chat instead");
    expect(approvalOutcomeLabel("answer", true)).toBe("Answered");
    expect(approvalOutcomeLabel("deny")).toBe("Denied");
    expect(failedSteps(reduce([...asked, ev({ type: "approval_resolved", seq: 3, approval_id: "a1", decision: "deny" }), ev({ type: "tool_call", seq: 4, name: "AskUserQuestion", tool_id: "t1", status: "denied" })]).turns[0])).toEqual([]);
  });
});

describe("parseQuestions", () => {
  it("reads the questions of an AskUserQuestion input", () => {
    const qs = parseQuestions(QUESTIONS_INPUT);
    expect(qs).toHaveLength(2);
    expect(qs[0]).toEqual({ question: "Which color?", header: "Color", options: [{ label: "Red", description: "Warm" }, { label: "Blue", description: "Cool" }], multiSelect: false });
    expect(qs[1].multiSelect).toBe(true);
  });

  it("tolerates missing fields and bad input", () => {
    expect(parseQuestions(undefined)).toEqual([]);
    expect(parseQuestions("not json")).toEqual([]);
    expect(parseQuestions('{"questions":"x"}')).toEqual([]);
    expect(parseQuestions('{"questions":[{"question":"Why?"},{"header":"no text"}]}')).toEqual([{ question: "Why?", header: "", options: [], multiSelect: false }]);
  });
});

describe("question picks", () => {
  const qs = parseQuestions(QUESTIONS_INPUT);

  it("needs an answer to every question", () => {
    let picks = emptyPicks(qs);
    expect(questionAnswers(qs, picks)).toBeNull();
    picks = pickOption(picks, qs, 0, "Red");
    expect(questionAnswers(qs, picks)).toBeNull();
    picks = pickOption(picks, qs, 1, "Dog");
    expect(questionAnswers(qs, picks)).toEqual({ "Which color?": "Red", "Which pets?": "Dog" });
  });

  it("takes one option for a single-select question", () => {
    let picks = pickOption(emptyPicks(qs), qs, 0, "Red");
    picks = pickOption(picks, qs, 0, "Blue");
    expect(picks[0]).toEqual({ picked: ["Blue"], other: null });
  });

  it("toggles options of a multi-select question, and keeps the option order", () => {
    let picks = emptyPicks(qs);
    for (const label of ["Fish", "Cat", "Dog"]) picks = pickOption(picks, qs, 1, label);
    picks = pickOption(picks, qs, 1, "Dog");
    expect(picks[1].picked).toEqual(["Fish", "Cat"]);
    picks = pickOption(picks, qs, 0, "Red");
    expect(questionAnswers(qs, picks)?.["Which pets?"]).toBe("Cat, Fish");
  });

  it("opens the free-text field only on Other, in place of the option for a single-select question", () => {
    let picks = pickOption(emptyPicks(qs), qs, 0, "Red");
    expect(picks[0].other).toBeNull();
    picks = toggleOther(picks, qs, 0);
    expect(picks[0]).toEqual({ picked: [], other: "" });
    picks = setOtherText(picks, 0, "  Teal ");
    picks = pickOption(picks, qs, 1, "Cat");
    expect(questionAnswers(qs, picks)?.["Which color?"]).toBe("Teal");
    // Picking an option again closes the field.
    picks = pickOption(picks, qs, 0, "Blue");
    expect(picks[0]).toEqual({ picked: ["Blue"], other: null });
  });

  it("adds free text after the options of a multi-select question", () => {
    let picks = pickOption(emptyPicks(qs), qs, 0, "Red");
    picks = pickOption(picks, qs, 1, "Cat");
    picks = toggleOther(picks, qs, 1);
    expect(questionAnswers(qs, picks)?.["Which pets?"]).toBe("Cat");
    picks = setOtherText(picks, 1, "a parrot");
    expect(questionAnswers(qs, picks)?.["Which pets?"]).toBe("Cat, a parrot");
    picks = toggleOther(picks, qs, 1);
    expect(picks[1]).toEqual({ picked: ["Cat"], other: null });
  });

  it("opens the field at once for a question without options", () => {
    const open = parseQuestions('{"questions":[{"question":"Why?"}]}');
    let picks = emptyPicks(open);
    expect(picks[0]).toEqual({ picked: [], other: "" });
    expect(questionAnswers(open, picks)).toBeNull();
    picks = setOtherText(picks, 0, "Because");
    expect(questionAnswers(open, picks)).toEqual({ "Why?": "Because" });
  });

  it("says whether one question has an answer", () => {
    let picks = emptyPicks(qs);
    expect(questionAnswered(qs[0], picks[0])).toBe(false);
    picks = pickOption(picks, qs, 0, "Red");
    expect(questionAnswered(qs[0], picks[0])).toBe(true);
    picks = toggleOther(picks, qs, 1);
    picks = setOtherText(picks, 1, "   ");
    expect(questionAnswered(qs[1], picks[1])).toBe(false);
    picks = setOtherText(picks, 1, "a parrot");
    expect(questionAnswered(qs[1], picks[1])).toBe(true);
    picks = setOtherText(picks, 1, "");
    picks = pickOption(picks, qs, 1, "Cat");
    expect(questionAnswered(qs[1], picks[1])).toBe(true);
  });
});

describe("question navigation", () => {
  const three = parseQuestions(
    JSON.stringify({
      questions: [
        { question: "Which color?", header: "Color", options: [{ label: "Red" }, { label: "Blue" }] },
        { question: "Which pets?", header: "", options: [{ label: "Cat" }], multiSelect: true },
        { question: "Why?", header: "Reason" },
      ],
    }),
  );

  it("labels a tab with the header, or Q and the number", () => {
    expect(questionTabLabel(three[0], 0)).toBe("Color");
    expect(questionTabLabel(three[1], 1)).toBe("Q2");
  });

  const one = parseQuestions('{"questions":[{"question":"A?","options":[{"label":"x"}]}]}');

  it("has a review step after the questions, only with 2 or more questions", () => {
    expect(reviewStep(three)).toBe(3);
    expect(reviewStep(one)).toBeNull();
    expect(reviewStep([])).toBeNull();
  });

  it("moves Next to the next question, whether or not it has an answer, and from the last one to review", () => {
    const picks = emptyPicks(three);
    expect(questionAdvance(three, picks, 0)).toEqual({ kind: "go", index: 1 });
    expect(questionAdvance(three, picks, 1)).toEqual({ kind: "go", index: 2 });
    expect(questionAdvance(three, picks, 2)).toEqual({ kind: "go", index: 3 });
  });

  it("sends only from the review step, once every question has an answer", () => {
    let picks = pickOption(emptyPicks(three), three, 0, "Red");
    picks = pickOption(picks, three, 1, "Cat");
    picks = setOtherText(picks, 2, "Because");
    expect(questionAdvance(three, picks, 3)).toEqual({ kind: "send" });
    expect(questionAdvance(three, picks, 2)).toEqual({ kind: "go", index: 3 });
    expect(questionAdvance(three, picks, 0)).toEqual({ kind: "go", index: 1 });
  });

  it("moves from the review step to the first question without an answer", () => {
    let picks = pickOption(emptyPicks(three), three, 0, "Red");
    picks = setOtherText(picks, 2, "Because");
    expect(questionAdvance(three, picks, 3)).toEqual({ kind: "go", index: 1 });
  });

  it("sends a single question straight away, once it has an answer", () => {
    expect(questionAdvance(one, emptyPicks(one), 0)).toEqual({ kind: "go", index: 0 });
    expect(questionAdvance(one, pickOption(emptyPicks(one), one, 0, "x"), 0)).toEqual({ kind: "send" });
  });

  it("auto-advances after a single-select option, from the last question to review", () => {
    expect(autoAdvanceTo(three, 0)).toBe(1);
    expect(autoAdvanceTo(three, 1)).toBeNull(); // multi-select
    const singles = parseQuestions('{"questions":[{"question":"A?","options":[{"label":"x"}]},{"question":"B?","options":[{"label":"y"}]}]}');
    expect(autoAdvanceTo(singles, 1)).toBe(2);
    expect(autoAdvanceTo(one, 0)).toBeNull();
  });

  it("maps arrow and number keys to a step or an option", () => {
    expect(questionKeyAction("ArrowRight", three, 0)).toEqual({ kind: "go", index: 1 });
    expect(questionKeyAction("ArrowLeft", three, 1)).toEqual({ kind: "go", index: 0 });
    expect(questionKeyAction("ArrowLeft", three, 0)).toBeNull();
    expect(questionKeyAction("ArrowRight", three, 2)).toEqual({ kind: "go", index: 3 });
    expect(questionKeyAction("ArrowRight", three, 3)).toBeNull();
    expect(questionKeyAction("ArrowLeft", three, 3)).toEqual({ kind: "go", index: 2 });
    expect(questionKeyAction("ArrowRight", one, 0)).toBeNull();
    expect(questionKeyAction("2", three, 0)).toEqual({ kind: "pick", label: "Blue" });
    expect(questionKeyAction("1", three, 3)).toBeNull(); // review
    expect(questionKeyAction("3", three, 0)).toBeNull();
    expect(questionKeyAction("0", three, 0)).toBeNull();
    expect(questionKeyAction("x", three, 0)).toBeNull();
  });

  it("writes one question's answer, or nothing", () => {
    let picks = pickOption(emptyPicks(three), three, 1, "Cat");
    picks = toggleOther(picks, three, 1);
    picks = setOtherText(picks, 1, " a parrot ");
    expect(questionAnswerText(three[1], picks[1])).toBe("Cat, a parrot");
    expect(questionAnswerText(three[0], picks[0])).toBe("");
  });
});
