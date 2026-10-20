import { describe, expect, it } from "vitest";
import type { ReactElement } from "react";
import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import { DndContext } from "@dnd-kit/core";
import { SortableContext } from "@dnd-kit/sortable";
import type { Issue } from "../../api/client";
import { makeIssue } from "../../test-utils";
import { renderMarkup } from "../../render-markup";
import { AgentChatContext, type AgentChat } from "../Agent/agent-context";
import { BoardCard, SortableBoardCard } from "./BoardCard";

const meta = { childDone: 0, childTotal: 0 };

function inDnd(el: ReactElement, id: string): string {
  return renderMarkup(
    <DndContext>
      <SortableContext items={[id]}>{el}</SortableContext>
    </DndContext>,
  );
}

function renderCard(issue: Issue, assigneeOpen = false): string {
  return inDnd(
    <SortableBoardCard issue={issue} meta={meta} assigneeOpen={assigneeOpen} onAssigneeOpen={() => {}} />,
    issue.id,
  );
}

/** The assignee trigger button's markup, up to its closing tag. */
function assigneeTrigger(html: string): string {
  const start = html.search(/<button[^>]*data-assignee-trigger/);
  expect(start).toBeGreaterThanOrEqual(0);
  return html.slice(start, html.indexOf("</button>", start));
}

describe("Board card assignee trigger", () => {
  it("shows a dashed placeholder trigger when unassigned", () => {
    const trigger = assigneeTrigger(renderCard(makeIssue()));
    expect(trigger).toContain('aria-label="No assignee, change assignee"');
    expect(trigger).toContain('aria-haspopup="menu"');
    expect(trigger).toContain('aria-expanded="false"');
    expect(trigger).toContain("border-dashed");
  });

  it("names the trigger after the assignee", () => {
    const trigger = assigneeTrigger(renderCard(makeIssue({ assignee: "Alice Smith <alice@x.io>" })));
    expect(trigger).toContain('aria-label="Alice Smith, change assignee"');
    expect(trigger).not.toContain("border-dashed");
  });

  it("marks the trigger expanded while its picker is open", () => {
    expect(assigneeTrigger(renderCard(makeIssue(), true))).toContain('aria-expanded="true"');
  });

  it("renders a static avatar on the drag overlay", () => {
    const issue = makeIssue();
    const html = inDnd(<BoardCard issue={issue} meta={meta} isOverlay />, issue.id);
    expect(html).not.toContain("data-assignee-trigger");
    expect(html).toContain("border-dashed");
  });
});

describe("Board card chat icon", () => {
  it("shows the chat icon on an issue a chat worked on, even with nothing else in the footer", () => {
    const issue = makeIssue({ labels: [] });
    const agent = {
      status: { available: true, default_provider: "claude", providers: [] },
      allChats: [
        { id: "c1", title: "t", created_at: "", updated_at: "", agent: "claude", state: "idle", worked_issues: [issue.id] },
        { id: "c2", title: "t", created_at: "", updated_at: "", agent: "claude", state: "idle", worked_issues: [issue.id] },
      ],
    } as unknown as AgentChat;
    const html = renderMarkup(
      <AgentChatContext.Provider value={agent}>
        <BoardCard issue={issue} meta={meta} />
      </AgentChatContext.Provider>,
    );
    expect(html).toContain(`aria-label="2 chats on ${issue.id}"`);
  });
});

describe("Board card release chip", () => {
  it("shows the release chip, even with nothing else in the footer", () => {
    const issue = makeIssue({ labels: [], release_id: "rel-2", release: "1.2" });
    const html = renderMarkup(<TooltipPrimitive.Provider><BoardCard issue={issue} meta={meta} /></TooltipPrimitive.Provider>);
    expect(html).toContain('aria-label="Release 1.2, change release"');
  });

  it("shows no chip without a release", () => {
    expect(renderMarkup(<BoardCard issue={makeIssue()} meta={meta} />)).not.toContain("Release ");
  });
});
