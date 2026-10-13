import { describe, expect, it } from "vitest";
import { makeIssue } from "../../test-utils";
import { renderMarkup } from "../../render-markup";
import { AgentChatContext, type AgentChat } from "../Agent/agent-context";
import PropertySidebar from "./PropertySidebar";

const issue = makeIssue({
  estimate: 2,
  priority: 1,
  assignee: "Alice <alice@x.io>",
  labels: ["bug"],
});

// An agent chat whose agent CLI is available, with no chats.
const agentAvailable = { status: { available: true, default_provider: "claude", providers: [] }, chats: [] } as unknown as AgentChat;

function renderSidebar(target = issue, gitActions?: boolean, agent: AgentChat | null = null): string {
  return renderMarkup(
    <AgentChatContext.Provider value={agent}>
    <PropertySidebar
      issue={target}
      issues={[target]}
      openPopover={null}
      setOpenPopover={() => {}}
      saveUpdate={() => {}}
      onClose={() => {}}
      contributors={[]}
      onOpenMerge={() => {}}
      gitActions={gitActions}
    />
    </AgentChatContext.Provider>,
  );
}

describe("PropertySidebar triggers", () => {
  it("marks every property trigger as a collapsed menu button", () => {
    const html = renderSidebar();
    const triggers = html.match(/<button[^>]*aria-haspopup="menu"[^>]*>/g) ?? [];
    // status, estimate, priority, relations, assignee, labels (cycle hides without cycles)
    expect(triggers).toHaveLength(6);
    for (const t of triggers) expect(t).toContain('aria-expanded="false"');
  });

  it("gives the icon-only labels trigger an accessible name", () => {
    expect(renderSidebar()).toMatch(/<button[^>]*aria-label="Add label"[^>]*aria-haspopup="menu"|<button[^>]*aria-haspopup="menu"[^>]*aria-label="Add label"/);
  });
});

describe("PropertySidebar git actions", () => {
  const started = makeIssue({
    status: "PLANNED",
    branch_stats: { branch: "xpo-aaa111-x", head_sha: "abc", commits: 2, files_changed: 1, insertions: 3, deletions: 0, has_uncommitted: false },
  });

  it("shows Start Work and Review Changes in local mode", () => {
    const html = renderSidebar(started, true);
    expect(html).toContain("Start Work");
    expect(html).toContain("Review Changes");
  });

  it("hides them in remote mode and while the mode is unknown", () => {
    for (const mode of [false, undefined]) {
      const html = renderSidebar(started, mode);
      expect(html).not.toContain("Start Work");
      expect(html).not.toContain("Review Changes");
    }
  });

  const doing = makeIssue({ ...started, status: "DOING" });

  it("offers Continue Work on a DOING issue with a branch when the agent is available", () => {
    const html = renderSidebar(doing, true, agentAvailable);
    expect(html).toContain("Continue Work");
    expect(html).not.toContain("Start Work");
  });

  it("hides Continue Work without an agent, in remote mode, or without a branch", () => {
    expect(renderSidebar(doing, true)).not.toContain("Continue Work");
    expect(renderSidebar(doing, false, agentAvailable)).not.toContain("Continue Work");
    expect(renderSidebar(makeIssue({ status: "DOING" }), true, agentAvailable)).not.toContain("Continue Work");
  });
});

describe("PropertySidebar chats", () => {
  const withChats = (n: number) =>
    ({
      ...agentAvailable,
      chats: Array.from({ length: n }, (_, i) => ({ id: `c${i}`, title: "t", created_at: "", updated_at: "", agent: "claude", state: "idle", worked_issues: [issue.id] })),
    }) as unknown as AgentChat;

  it("shows a chats button below Relations when chats worked on the issue", () => {
    const html = renderSidebar(issue, true, withChats(2));
    const button = html.match(/<button[^>]*>(?:(?!<\/button>).)*2 chats(?:(?!<\/button>).)*<\/button>/)?.[0];
    expect(button).toBeDefined();
    expect(button).toContain('aria-haspopup="menu"');
    expect(renderSidebar(issue, true, withChats(1))).toContain("1 chat<");
  });

  it("hides it without chats", () => {
    expect(renderSidebar(issue, true, agentAvailable)).not.toMatch(/\d chats?</);
  });
});
