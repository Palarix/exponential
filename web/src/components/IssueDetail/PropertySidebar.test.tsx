import { describe, expect, it } from "vitest";
import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import { QueryClient } from "@tanstack/react-query";
import { queryKeys } from "../../api/query-keys";
import type { Issue, Release } from "../../api/types";
import { makeIssue } from "../../test-utils";
import { renderMarkup } from "../../render-markup";
import { AgentChatContext, type AgentChat } from "../Agent/agent-context";
import PropertySidebar, { BlockerList, StartWorkBlockedReasons } from "./PropertySidebar";

const issue = makeIssue({
  estimate: 2,
  priority: 1,
  assignee: "Alice <alice@x.io>",
  labels: ["bug"],
});

// An agent chat whose agent CLI is available, with no chats.
const agentAvailable = { status: { available: true, default_provider: "claude", providers: [] }, chats: [], allChats: [] } as unknown as AgentChat;

function renderSidebar(target = issue, agent: AgentChat | null = null, others: Issue[] = [], client?: QueryClient): string {
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
    />
    </AgentChatContext.Provider>
    </TooltipPrimitive.Provider>,
    client,
  );
}

/** A query client whose board answered /api/local/mode with no git repository. */
function noGitClient(): QueryClient {
  const client = new QueryClient();
  client.setQueryData(queryKeys.localMode, { git: false, root: "/work/proj" });
  return client;
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

  it("shows Start Work and Review Changes", () => {
    const html = renderSidebar(started);
    expect(html).toContain("Start Work");
    expect(html).toContain("Review Changes");
  });

  const doing = makeIssue({ ...started, status: "DOING" });

  it("offers Continue Work on a DOING issue with a branch when the agent is available", () => {
    const html = renderSidebar(doing, agentAvailable);
    expect(html).toContain("Continue Work");
    expect(html).not.toContain("Start Work");
  });

  it("hides Continue Work without an agent or without a branch", () => {
    expect(renderSidebar(doing)).not.toContain("Continue Work");
    expect(renderSidebar(makeIssue({ status: "DOING" }), agentAvailable)).not.toContain("Continue Work");
  });
});

describe("PropertySidebar chats", () => {
  const withChats = (n: number) =>
    ({
      ...agentAvailable,
      allChats: Array.from({ length: n }, (_, i) => ({ id: `c${i}`, title: "t", created_at: "", updated_at: "", agent: "claude", state: "idle", worked_issues: [issue.id] })),
    }) as unknown as AgentChat;

  it("shows a chats button below Relations when chats worked on the issue", () => {
    const html = renderSidebar(issue, withChats(2));
    const button = html.match(/<button[^>]*>(?:(?!<\/button>).)*2 chats(?:(?!<\/button>).)*<\/button>/)?.[0];
    expect(button).toBeDefined();
    expect(button).toContain('aria-haspopup="menu"');
    expect(renderSidebar(issue, withChats(1))).toContain("1 chat<");
  });

  it("hides it without chats", () => {
    expect(renderSidebar(issue, agentAvailable)).not.toMatch(/\d chats?</);
  });
});

describe("PropertySidebar button priority", () => {
  const stats = { branch: "xpo-aaa111-x", head_sha: "abc", commits: 0, files_changed: 1, insertions: 1, deletions: 0, has_uncommitted: true };
  /** The button whose label is text, up to its closing tag. */
  const button = (html: string, text: string) => html.match(new RegExp(`<button[^>]*>(?:(?!</button>).)*${text}`))?.[0];

  it("makes Continue Work secondary when Review Changes shows", () => {
    const html = renderSidebar(makeIssue({ status: "DOING", branch_stats: stats }), agentAvailable);
    expect(button(html, "Review Changes")).toContain("bg-[var(--color-accent-primary)]");
    expect(button(html, "Continue Work")).not.toContain("bg-[var(--color-accent-primary)]");
    expect(button(html, "Continue Work")).toContain("border-[var(--color-border-default)]");
    // The primary action is always the top button.
    expect(html.indexOf("Review Changes")).toBeLessThan(html.indexOf("Continue Work"));
  });

  it("offers a secondary View changes button on a merged DONE issue", () => {
    const merged = makeIssue({ status: "DONE", merged: true });
    const html = renderSidebar(merged);
    expect(button(html, "View changes")).toBeDefined();
    expect(button(html, "View changes")).not.toContain("bg-[var(--color-accent-primary)]");
    expect(button(html, "View changes")).toContain("border-[var(--color-border-default)]");
    expect(html).not.toContain("Review Changes");
  });

  it("hides View changes for issues that weren't merged or aren't DONE", () => {
    expect(renderSidebar(makeIssue({ status: "DONE" }))).not.toContain("View changes");
    expect(renderSidebar(makeIssue({ status: "DOING", merged: true }))).not.toContain("View changes");
  });

  it("keeps the work button primary without Review Changes", () => {
    const html = renderSidebar(makeIssue({ status: "DOING", branch_stats: { ...stats, files_changed: 0, has_uncommitted: false } }), agentAvailable);
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
      const html = renderSidebar(doing, withChat("work"));
      expect(button(html, "Continue Work")).not.toContain("bg-[var(--color-accent-primary)]");
      expect(button(html, "Continue Work")).toContain("border-[var(--color-border-default)]");
    });

    it("keeps Continue Work primary once the chat has left Work Mode", () => {
      const html = renderSidebar(doing, withChat(undefined));
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
      const html = renderSidebar(blockedIssue("xpo-bbb222"), null, [blockerWith(status)]);
      expect(startButton(html)).toMatch(/disabled=""/);
    });
  }

  it("enables Start Work once every blocker is DONE", () => {
    const html = renderSidebar(blockedIssue("xpo-bbb222"), null, [blockerWith("DONE")]);
    expect(startButton(html)).toBeDefined();
    expect(startButton(html)).not.toMatch(/disabled=""/);
  });

  it("doesn't disable Start Work for informational links", () => {
    const target = makeIssue({
      status: "PLANNED",
      dependencies: ["relates_to", "depends_on"].map(kind => ({ source_id: "xpo-aaa111", target_id: "xpo-bbb222", kind })),
    });
    expect(startButton(renderSidebar(target, null, [blockerWith("PLANNED")]))).not.toMatch(/disabled=""/);
  });
});

describe("PropertySidebar Start Work in BACKLOG", () => {
  const startButton = (html: string) => html.match(/<button[^>]*>(?:(?!<\/button>).)*Start Work/)?.[0];

  it("disables Start Work while the issue is in BACKLOG", () => {
    const html = renderSidebar(makeIssue({ status: "BACKLOG" }));
    expect(startButton(html)).toMatch(/disabled=""/);
    expect(html).toContain("cursor-not-allowed");
  });

  it("enables Start Work for a PLANNED issue without blockers", () => {
    const html = renderSidebar(makeIssue({ status: "PLANNED" }));
    expect(startButton(html)).toBeDefined();
    expect(startButton(html)).not.toMatch(/disabled=""/);
  });
});

describe("PropertySidebar Start Work without a git repository", () => {
  const startButton = (html: string) => html.match(/<button[^>]*>(?:(?!<\/button>).)*Start Work/)?.[0];
  const planned = makeIssue({ status: "PLANNED" });

  it("disables Start Work when the agent would open Work Mode", () => {
    const html = renderSidebar(planned, agentAvailable, [], noGitClient());
    expect(startButton(html)).toMatch(/disabled=""/);
    expect(html).toContain("cursor-not-allowed");
  });

  it("keeps Start Work enabled without an agent: it only starts the issue", () => {
    const html = renderSidebar(planned, null, [], noGitClient());
    expect(startButton(html)).not.toMatch(/disabled=""/);
  });
});

describe("StartWorkBlockedReasons", () => {
  const render = (inBacklog: boolean, blockers: Issue[], noGitRoot?: string) =>
    renderMarkup(<StartWorkBlockedReasons noGitRoot={noGitRoot} inBacklog={inBacklog} blockers={blockers} onRemoveBlocker={() => {}} />);

  it("says Work Mode needs git, and where to run git init, first", () => {
    const html = render(true, [], "/work/proj");
    expect(html).toContain("Work Mode needs a git repository");
    expect(html).toContain("git init");
    expect(html).toContain("/work/proj");
    expect(html.indexOf("git repository")).toBeLessThan(html.indexOf("Move the issue to Planned"));
  });

  it("says to move a BACKLOG issue to Planned", () => {
    const html = render(true, []);
    expect(html).toContain("Move the issue to Planned to start work");
    expect(html).not.toContain("Blocked by");
  });

  it("lists both reasons, BACKLOG first, when a BACKLOG issue also has blockers", () => {
    const html = render(true, [makeIssue({ id: "xpo-bbb222", status: "PLANNED" })]);
    expect(html).toContain("Blocked by xpo-bbb222");
    expect(html.indexOf("Move the issue to Planned")).toBeLessThan(html.indexOf("Blocked by xpo-bbb222"));
  });

  it("lists only blockers outside BACKLOG", () => {
    const html = render(false, [makeIssue({ id: "xpo-bbb222", status: "PLANNED" })]);
    expect(html).toContain("Blocked by xpo-bbb222");
    expect(html).not.toContain("Move the issue to Planned");
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

describe("PropertySidebar release", () => {
  const rel = (number: number, o: Partial<Release> = {}): Release => ({
    id: `rel-${number}`, number, name: o.label ?? `#${number}`, state: "OPEN", counts: {}, total: 0, done: 0, ...o,
  });
  function releasesClient(): QueryClient {
    const client = new QueryClient();
    client.setQueryData(queryKeys.releases, {
      current: rel(1, { label: "1.1", slot: "current" }),
      next: rel(2, { slot: "next", placeholder: true, id: undefined }),
      upcoming: [],
      past: [rel(9, { label: "1.0", state: "RELEASED", slot: "past" })],
    });
    return client;
  }

  it("shows the issue's release as a picker trigger", () => {
    const html = renderSidebar(makeIssue({ release_id: "rel-1", release: "1.1" }), null, [], releasesClient());
    expect(html).toMatch(/aria-haspopup="menu"[^>]*>(?:(?!<\/button>).)*1\.1/);
  });

  it("says No release without one", () => {
    expect(renderSidebar(makeIssue(), null, [], releasesClient())).toContain("No release");
  });

  it("offers Move to Current for an open issue in a released release", () => {
    const html = renderSidebar(makeIssue({ release_id: "rel-9", release: "1.0" }), null, [], releasesClient());
    expect(html).toContain("In released release 1.0.");
    expect(html).toContain("Move to Current");
  });

  it("gives a done issue in a released release no hint", () => {
    const html = renderSidebar(makeIssue({ release_id: "rel-9", release: "1.0", status: "DONE" }), null, [], releasesClient());
    expect(html).not.toContain("Move to Current");
  });
});
