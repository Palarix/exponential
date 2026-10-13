import { describe, expect, it } from "vitest";
import type { ReactNode } from "react";
import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import type { ChatEvent } from "../../api/chat";
import { renderMarkup } from "../../render-markup";
import ChatThread from "./ChatThread";
import { chatThreadReducer, emptyThread } from "./chat-thread-utils";

const T0 = "2026-10-13T10:00:00Z";

// The app wraps everything in a tooltip provider (main.tsx).
function render(node: ReactNode): string {
  return renderMarkup(<TooltipPrimitive.Provider>{node}</TooltipPrimitive.Provider>);
}

function thread(events: Partial<ChatEvent>[]) {
  return events.reduce((s, e) => chatThreadReducer(s, { type: "event", event: { time: T0, ...e } as ChatEvent }), emptyThread);
}

describe("ChatThread work list", () => {
  it("opens the work list of a turn with a failed call", () => {
    const html = render(
      <ChatThread
        thread={thread([
          { type: "user_message", seq: 1, text: "merge it" },
          { type: "tool_call", seq: 2, name: "mcp__xpo__merge", tool_id: "t1", arg: "upe-1", status: "error", message: "no branch found" },
          { type: "tool_call", seq: 3, name: "mcp__xpo__show", tool_id: "t2", arg: "upe-1", status: "done" },
          { type: "assistant_text", seq: 4, text: "It failed." },
          { type: "turn_done", seq: 5, duration_ms: 3000 },
        ])}
      />,
    );
    expect(html).toContain("failed: no branch found");
    expect(html).toContain("Opened");
    expect(html).toContain('aria-expanded="true"');
  });

  it("keeps the work list of a clean turn collapsed", () => {
    const html = render(
      <ChatThread
        thread={thread([
          { type: "user_message", seq: 1, text: "show it" },
          { type: "tool_call", seq: 2, name: "mcp__xpo__show", tool_id: "t2", arg: "upe-1", status: "done" },
          { type: "assistant_text", seq: 3, text: "Here." },
          { type: "turn_done", seq: 4, duration_ms: 1000 },
        ])}
      />,
    );
    expect(html).not.toContain("Opened");
    expect(html).toContain('aria-expanded="false"');
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
    const work = html.indexOf("Work Mode on");
    const message = html.indexOf("Let&#x27;s implement xpo-1.");
    const back = html.indexOf("Back to Chat Mode");
    expect(work).toBeGreaterThan(-1);
    expect(html.match(/role="separator"/g)).toHaveLength(2);
    expect(work).toBeLessThan(message);
    expect(message).toBeLessThan(back);
  });
});
