import { describe, expect, it } from "vitest";
import type { ReactNode } from "react";
import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import type { ChatEvent } from "../../api/chat";
import { renderMarkup } from "../../render-markup";
import ChatThread, { ToolTimeline } from "./ChatThread";
import { chatThreadReducer, emptyThread, type ToolStep } from "./chat-thread-utils";

const T0 = "2026-10-13T10:00:00Z";

// The app wraps everything in a tooltip provider (main.tsx).
function render(node: ReactNode): string {
  return renderMarkup(<TooltipPrimitive.Provider>{node}</TooltipPrimitive.Provider>);
}

function thread(events: Partial<ChatEvent>[]) {
  return events.reduce((s, e) => chatThreadReducer(s, { type: "event", event: { time: T0, ...e } as ChatEvent }), emptyThread);
}

describe("ChatThread turn layout", () => {
  const finished = thread([
    { type: "user_message", seq: 1, text: "go" },
    { type: "assistant_text", seq: 2, text: "Let me look." },
    { type: "tool_call", seq: 3, name: "Read", tool_id: "t1", arg: "/r/board.go", status: "done" },
    { type: "tool_call", seq: 4, name: "mcp__xpo__merge", tool_id: "t2", arg: "upe-1", status: "error", message: "no branch" },
    { type: "assistant_text", seq: 5, text: "Now the tests." },
    { type: "tool_call", seq: 6, name: "Bash", tool_id: "t3", arg: "Run the tests", status: "done" },
    { type: "assistant_text", seq: 7, text: "All done." },
    { type: "turn_done", seq: 8, duration_ms: 84_000 },
  ]);

  it("shows narration as messages with a line for each group of calls between them", () => {
    const html = render(<ChatThread thread={finished} />);
    const order = ["Let me look.", "2 actions", "Now the tests.", "1 action", "All done."].map((t) => html.indexOf(t));
    expect(order.every((i) => i >= 0)).toBe(true);
    expect([...order].sort((x, y) => x - y)).toEqual(order);
    expect(html).not.toContain("Worked for");
  });

  it("keeps a group with failures collapsed and counts them", () => {
    const html = render(<ChatThread thread={finished} />);
    expect(html.match(/aria-expanded="false"/g)).toHaveLength(2);
    expect(html).toContain("1 failed");
    expect(html).not.toContain("failed: no branch");
    expect(html.match(/lucide-chevron-right/g)).toHaveLength(2);
  });

  it("shows the total time of a finished turn under its last message", () => {
    const html = render(<ChatThread thread={finished} />);
    expect(html).toContain(">1m 24s<");
    expect(html.indexOf(">1m 24s<")).toBeGreaterThan(html.indexOf("All done."));
  });
});

describe("ChatThread live turn", () => {
  const running = thread([
    { type: "user_message", seq: 1, text: "go" },
    { type: "tool_call", seq: 2, name: "Read", tool_id: "t1", arg: "/r/board.go", status: "done" },
    { type: "assistant_text", seq: 3, text: "Searching next." },
    { type: "tool_start", name: "Grep", tool_id: "t2", arg: "TODO" },
  ]);

  it("shows the current call of the active group with the timer", () => {
    const html = render(<ChatThread thread={running} />);
    const line = html.match(/<button[^>]*data-active-group[^>]*>.*?<\/button>/)?.[0];
    expect(line).toBeDefined();
    expect(line).toContain('aria-label="Searched the code for"');
    expect(line).toContain("TODO");
    expect(line).toMatch(/>(\d+m )?\d+s<|>\d+m</);
    // One icon, the call's; the text shimmers instead of a spinner.
    expect(line).not.toContain("animate-spin");
    expect(line).toContain("text-shimmer");
    // The group before the narration is done.
    expect(html).toContain("1 action");
  });

  it("has nothing to expand while the active group has only its current call", () => {
    const first = thread([{ type: "user_message", seq: 1, text: "go" }, { type: "tool_start", name: "Grep", tool_id: "t1", arg: "TODO" }]);
    const line = render(<ChatThread thread={first} />).match(/<button[^>]*data-active-group[^>]*>.*?<\/button>/)?.[0];
    expect(line).not.toContain("aria-expanded");
    expect(line).not.toContain("lucide-chevron-right");
  });

  it("shows the approval wait on the active line", () => {
    const waiting = chatThreadReducer(running, { type: "event", event: { time: T0, type: "approval_requested", seq: 4, tool_id: "t2", approval_id: "a1", name: "Grep" } as ChatEvent });
    const html = render(<ChatThread thread={waiting} chatId="c1" />);
    expect(html.match(/<button[^>]*data-active-group[^>]*>.*?<\/button>/)?.[0]).toContain("Waiting for your approval");
  });

  it("shows Thinking while no group is active and nothing streams", () => {
    const thinking = thread([{ type: "user_message", seq: 1, text: "go" }, { type: "thinking" }]);
    expect(render(<ChatThread thread={thinking} />)).toContain("Thinking");
    const fresh = thread([{ type: "user_message", seq: 1, text: "go" }]);
    expect(render(<ChatThread thread={fresh} />)).toContain("Thinking");
    const streaming = thread([{ type: "user_message", seq: 1, text: "go" }, { type: "thinking" }, { type: "text_delta", text: "Let me" }]);
    expect(render(<ChatThread thread={streaming} />)).not.toContain("Thinking");
    expect(render(<ChatThread thread={running} />)).not.toContain("Thinking");
  });
});

describe("ToolTimeline", () => {
  const steps = thread([
    { type: "user_message", seq: 1, text: "go" },
    { type: "tool_call", seq: 2, name: "Read", tool_id: "t1", arg: "/r/board.go", status: "done" },
    { type: "tool_call", seq: 3, name: "Bash", tool_id: "t2", arg: "Run the tests", status: "done" },
    { type: "tool_call", seq: 4, name: "WebFetch", tool_id: "t3", arg: "https://a.b", status: "done" },
    { type: "tool_call", seq: 5, name: "mcp__xpo__merge", tool_id: "t4", arg: "upe-1", status: "error", message: "no branch" },
  ]).turns[0].steps as ToolStep[];

  it("draws each call as a branch with an icon labelled with its verb", () => {
    const html = render(<ToolTimeline steps={steps} />);
    expect(html.match(/data-step="tool"/g)).toHaveLength(4);
    expect(html).toContain('role="img" aria-label="Read" title="Read"');
    expect(html).toContain('aria-label="Ran a shell command"');
    expect(html).not.toContain(">Ran a shell command<");
    expect(html).toContain("board.go");
    // An unknown tool keeps its name next to the generic icon.
    expect(html).toContain('aria-label="WebFetch"');
    expect(html).toContain(">WebFetch<");
    // A failed call reads as what was asked, with its reason.
    expect(html).toContain('aria-label="Merge"');
    expect(html).toContain("failed: no branch");
  });

  it("marks a failed call with a red icon only; its text and reason read as normal text", () => {
    const failed = steps.filter((st) => st.status === "error");
    const html = render(<ToolTimeline steps={failed} />);
    expect(html.match(/color-error/g)).toHaveLength(1);
    expect(html).toMatch(/<span role="img" aria-label="Merge" title="Merge" class="[^"]*color-error/);
    expect(html).toContain("failed: no branch");
  });

  it("shows a running call with its icon and shimmering text, not a spinner", () => {
    const running: ToolStep = { kind: "tool", id: "r", name: "Grep", arg: "TODO", done: false };
    const html = render(<ToolTimeline steps={[running]} />);
    expect(html).toContain('aria-label="Searched the code for"');
    expect(html).toContain("text-shimmer");
    expect(html).not.toContain("animate-spin");
  });
});

describe("ChatThread mode dividers", () => {
  it("draws dividers between turns, in order", () => {
    const html = render(
      <ChatThread
        thread={thread([
          { type: "mode_changed", seq: 1, mode: "work", issue_id: "xpo-1" },
          { type: "user_message", seq: 2, text: "Let's implement xpo-1." },
          { type: "turn_done", seq: 3, duration_ms: 1000 },
          { type: "mode_changed", seq: 4, mode: "chat", issue_id: "xpo-1" },
        ])}
      />,
    );
    const work = html.indexOf("Entered Work Mode on");
    const message = html.indexOf("Let&#x27;s implement xpo-1.");
    const back = html.indexOf("Entered Chat Mode");
    expect(work).toBeGreaterThan(-1);
    expect(html.match(/role="separator"/g)).toHaveLength(2);
    expect(work).toBeLessThan(message);
    expect(message).toBeLessThan(back);
  });
});

describe("ChatThread continued turn", () => {
  it("shows no user bubble for a turn the board continued", () => {
    const html = render(
      <ChatThread
        thread={thread([
          { type: "user_message", seq: 1, text: "start xpo-1" },
          { type: "turn_done", seq: 2, duration_ms: 1000 },
          { type: "mode_changed", seq: 3, mode: "work", issue_id: "xpo-1" },
          { type: "turn_continued", seq: 4, issue_id: "xpo-1" },
          { type: "assistant_text", seq: 5, text: "Building the scaffold." },
          { type: "turn_done", seq: 6, duration_ms: 2000 },
        ])}
      />,
    );
    expect(html.match(/bg-\[var\(--color-surface-ai-2\)\]/g)).toHaveLength(1);
    const divider = html.indexOf("Entered Work Mode on");
    expect(divider).toBeGreaterThan(-1);
    expect(divider).toBeLessThan(html.indexOf("Building the scaffold."));
  });
});

describe("ChatThread cold cache marker", () => {
  const done = thread([
    { type: "user_message", seq: 1, text: "hi" },
    { type: "assistant_text", seq: 2, text: "Hello." },
    { type: "turn_done", seq: 3, duration_ms: 1000 },
  ]);
  const marker = "Context cache expired";

  it("shows a divider under the last turn once the cache has expired", () => {
    const html = render(<ChatThread thread={done} cacheExpiresAt="2026-10-13T10:05:00Z" />);
    const divider = html.match(/<div role="separator"[^>]*>(?:(?!<div role="separator").)*Context cache expired/)?.[0];
    expect(divider).toBeDefined();
    expect(divider).toContain('title="Your next message re-sends the whole conversation."');
    // Dashed, unlike the solid lines of a mode divider.
    expect(divider).toContain("border-dashed");
    expect(html.indexOf(marker)).toBeGreaterThan(html.indexOf("Hello."));
  });

  it("hides it while the cache is warm, without an expiry, or while a turn runs", () => {
    const future = new Date(Date.now() + 3600_000).toISOString();
    expect(render(<ChatThread thread={done} cacheExpiresAt={future} />)).not.toContain(marker);
    expect(render(<ChatThread thread={done} />)).not.toContain(marker);
    const running = chatThreadReducer(done, { type: "event", event: { time: T0, type: "user_message", seq: 4, text: "again" } as ChatEvent });
    expect(render(<ChatThread thread={running} cacheExpiresAt="2026-10-13T10:05:00Z" />)).not.toContain(marker);
  });
});
