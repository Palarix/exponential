import { describe, expect, it } from "vitest";
import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import type { Issue } from "../../api/types";
import { makeIssue } from "../../test-utils";
import { renderMarkup } from "../../render-markup";
import { AgentChatContext, type AgentChat } from "../Agent/agent-context";
import PropertySidebar, { BlockerList } from "./PropertySidebar";

const issue = makeIssue({
  estimate: 2,
  priority: 1,
  assignee: "Alice <alice@x.io>",
  labels: ["bug"],
});

// An agent chat whose agent CLI is available, with no chats.
const agentAvailable = { status: { available: true, default_provider: "claude", providers: [] }, chats: [], allChats: [] } as unknown as AgentChat;

function renderSidebar(target = issue, gitActions?: boolean, agent: AgentChat | null = null, others: Issue[] = []): string {
  // The app wraps everything in a tooltip provider (main.tsx).
  return renderMarkup(
    <TooltipPrimitive.Provider>
    <AgentChatContext.Provider value={agent}>
    <PropertySidebar
      issue={target}
      issues={[target, ...others]}
      openPopover={null}
      setOpenPopover={() => {}}
      saveUpdate={() => {}}
      onClose={() => {}}
      contributors={[]}
      onOpenMerge={() => {}}
      gitActions={gitActions}
    />
    </AgentChatContext.Provider>
    </TooltipPrimitive.Provider>,
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
      allChats: Array.from({ length: n }, (_, i) => ({ id: `c${i}`, title: "t", created_at: "", updated_at: "", agent: "claude", state: "idle", worked_issues: [issue.id] })),
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

describe("PropertySidebar button priority", () => {
  const stats = { branch: "xpo-aaa111-x", head_sha: "abc", commits: 0, files_changed: 1, insertions: 1, deletions: 0, has_uncommitted: true };
  /** The button whose label is text, up to its closing tag. */
  const button = (html: string, text: string) => html.match(new RegExp(`<button[^>]*>(?:(?!</button>).)*${text}`))?.[0];

  it("makes Continue Work secondary when Review Changes shows", () => {
    const html = renderSidebar(makeIssue({ status: "DOING", branch_stats: stats }), true, agentAvailable);
    expect(button(html, "Review Changes")).toContain("bg-[var(--color-accent-primary)]");
    expect(button(html, "Continue Work")).not.toContain("bg-[var(--color-accent-primary)]");
    expect(button(html, "Continue Work")).toContain("border-[var(--color-border-default)]");
    // The primary action is always the top button.
    expect(html.indexOf("Review Changes")).toBeLessThan(html.indexOf("Continue Work"));
  });

  it("offers a secondary View changes button on a merged DONE issue", () => {
    const merged = makeIssue({ status: "DONE", merged: true });
    const html = renderSidebar(merged, true);
    expect(button(html, "View changes")).toBeDefined();
    expect(button(html, "View changes")).not.toContain("bg-[var(--color-accent-primary)]");
    expect(button(html, "View changes")).toContain("border-[var(--color-border-default)]");
    expect(html).not.toContain("Review Changes");
  });

  it("hides View changes in remote mode, and for issues that weren't merged or aren't DONE", () => {
    expect(renderSidebar(makeIssue({ status: "DONE", merged: true }), false)).not.toContain("View changes");
    expect(renderSidebar(makeIssue({ status: "DONE" }), true)).not.toContain("View changes");
    expect(renderSidebar(makeIssue({ status: "DOING", merged: true }), true)).not.toContain("View changes");
  });

  it("keeps the work button primary without Review Changes", () => {
    const html = renderSidebar(makeIssue({ status: "DOING", branch_stats: { ...stats, files_changed: 0, has_uncommitted: false } }), true, agentAvailable);
    expect(html).not.toContain("Review Changes");
    expect(button(html, "Continue Work")).toContain("bg-[var(--color-accent-primary)]");
  });

  describe("Continue Work with a Work Mode chat", () => {
    const doing = makeIssue({ status: "DOING", branch_stats: { ...stats, files_changed: 0, has_uncommitted: false } });
    const withChat = (mode: "work" | undefined) =>
      ({
        ...agentAvailable,
        allChats: [{ id: "c1", title: "t", created_at: "", updated_at: "", agent: "claude", state: "idle", mode, work: mode ? { issue_id: doing.id, worktree: "/w" } : undefined, worked_issues: [doing.id] }],
      }) as unknown as AgentChat;

    it("makes Continue Work secondary while a chat is in Work Mode on the issue", () => {
      const html = renderSidebar(doing, true, withChat("work"));
      expect(button(html, "Continue Work")).not.toContain("bg-[var(--color-accent-primary)]");
      expect(button(html, "Continue Work")).toContain("border-[var(--color-border-default)]");
    });

    it("keeps Continue Work primary once the chat has left Work Mode", () => {
      const html = renderSidebar(doing, true, withChat(undefined));
      expect(button(html, "Continue Work")).toContain("bg-[var(--color-accent-primary)]");
    });
  });
});

describe("PropertySidebar Start Work blockers", () => {
  const blockerWith = (status: string, id = "xpo-bbb222") => makeIssue({ id, title: `blocker ${status}`, status });
  const blockedIssue = (...targets: string[]) =>
    makeIssue({ status: "PLANNED", dependencies: targets.map(t => ({ source_id: "xpo-aaa111", target_id: t, kind: "blocked_by" })) });
  const startButton = (html: string) => html.match(/<button[^>]*>(?:(?!<\/button>).)*Start Work/)?.[0];

  for (const status of ["PLANNED", "DOING", "CANCELED", "DUPLICATE"]) {
    it(`disables Start Work while a ${status} blocker is unresolved`, () => {
      const html = renderSidebar(blockedIssue("xpo-bbb222"), true, null, [blockerWith(status)]);
      expect(startButton(html)).toMatch(/disabled=""/);
    });
  }

  it("enables Start Work once every blocker is DONE", () => {
    const html = renderSidebar(blockedIssue("xpo-bbb222"), true, null, [blockerWith("DONE")]);
    expect(startButton(html)).toBeDefined();
    expect(startButton(html)).not.toMatch(/disabled=""/);
  });

  it("doesn't disable Start Work for informational links", () => {
    const target = makeIssue({
      status: "PLANNED",
      dependencies: ["relates_to", "depends_on"].map(kind => ({ source_id: "xpo-aaa111", target_id: "xpo-bbb222", kind })),
    });
    expect(startButton(renderSidebar(target, true, null, [blockerWith("PLANNED")]))).not.toMatch(/disabled=""/);
  });
});

describe("PropertySidebar delete", () => {
  it("offers Delete issue, with no inline confirmation", () => {
    const html = renderSidebar();
    expect(html).toMatch(/<button[^>]*>(?:(?!<\/button>).)*Delete issue<\/button>/);
    expect(html).not.toContain("Delete sub-issues too");
    expect(html).not.toContain("Delete this issue?");
  });
});

describe("BlockerList", () => {
  const render = (blockers: Issue[]) => renderMarkup(<BlockerList blockers={blockers} onRemove={() => {}} />);

  it("names an open blocker without a status note or Remove link", () => {
    const html = render([makeIssue({ id: "xpo-bbb222", status: "PLANNED" })]);
    expect(html).toContain("Blocked by xpo-bbb222");
    expect(html).not.toMatch(/\((canceled|duplicate)\)/);
    expect(html).not.toContain("Remove link");
  });

  it("marks CANCELED and DUPLICATE blockers and offers Remove link for each", () => {
    const html = render([
      makeIssue({ id: "xpo-bbb222", status: "CANCELED" }),
      makeIssue({ id: "xpo-ccc333", status: "DUPLICATE" }),
    ]);
    expect(html).toContain("Blocked by xpo-bbb222 (canceled)");
    expect(html).toContain("Blocked by xpo-ccc333 (duplicate)");
    expect(html.match(/Remove link/g)).toHaveLength(2);
  });
});
