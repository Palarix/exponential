import { describe, it, expect } from "vitest";
import type { InboxItem } from "../../api/types";
import { groupByIssue, summaryActor, buildChangeSummary } from "./inbox-utils";

function makeItem(overrides: Partial<InboxItem> = {}): InboxItem {
  return {
    issue_id: "xpo-abc123",
    issue_title: "Test issue",
    type: "UPDATE",
    payload: {},
    created_at: "2026-03-15T10:00:00Z",
    created_by: "Alice <alice@example.com>",
    ...overrides,
  };
}

describe("groupByIssue", () => {
  it("groups items by issue_id", () => {
    const items = [
      makeItem({ issue_id: "a", created_at: "2026-03-15T10:00:00Z" }),
      makeItem({ issue_id: "b", created_at: "2026-03-15T11:00:00Z" }),
      makeItem({ issue_id: "a", created_at: "2026-03-15T12:00:00Z" }),
    ];
    const groups = groupByIssue(items, 0);
    expect(groups).toHaveLength(2);
    expect(groups.find((g) => g.issueId === "a")?.events).toHaveLength(2);
    expect(groups.find((g) => g.issueId === "b")?.events).toHaveLength(1);
  });

  it("sorts groups by latest timestamp descending", () => {
    const items = [
      makeItem({ issue_id: "old", created_at: "2026-03-14T10:00:00Z" }),
      makeItem({ issue_id: "new", created_at: "2026-03-15T10:00:00Z" }),
    ];
    const groups = groupByIssue(items, 0);
    expect(groups[0].issueId).toBe("new");
    expect(groups[1].issueId).toBe("old");
  });

  it("marks groups with events after lastReadTime as unread", () => {
    const lastRead = new Date("2026-03-15T09:00:00Z").getTime();
    const items = [
      makeItem({ issue_id: "a", created_at: "2026-03-15T10:00:00Z" }),
      makeItem({ issue_id: "b", created_at: "2026-03-15T08:00:00Z" }),
    ];
    const groups = groupByIssue(items, lastRead);
    expect(groups.find((g) => g.issueId === "a")?.hasUnread).toBe(true);
    expect(groups.find((g) => g.issueId === "b")?.hasUnread).toBe(false);
  });

  it("uses issue_id as title fallback", () => {
    const items = [makeItem({ issue_id: "xpo-abc", issue_title: "" })];
    const groups = groupByIssue(items, 0);
    expect(groups[0].issueTitle).toBe("xpo-abc");
  });

  it("returns empty for no items", () => {
    expect(groupByIssue([], 0)).toEqual([]);
  });
});

describe("summaryActor", () => {
  it("returns null when no on_behalf_of", () => {
    expect(summaryActor(makeItem(), "alice@example.com")).toBeNull();
  });

  it("returns the principal with the agent as via, marking the viewer", () => {
    const item = makeItem({
      created_by: "claude-code/2.1.294 <agent@mcp>",
      on_behalf_of: "Alice <alice@example.com>",
    });
    expect(summaryActor(item, "alice@example.com")).toEqual({
      principal: "Alice",
      via: "claude-code/2.1.294 <agent@mcp>",
      isYou: true,
    });
  });

  it("does not mark other principals as the viewer", () => {
    const item = makeItem({
      created_by: "claude-code/2.1.294 <agent@mcp>",
      on_behalf_of: "Bob <bob@example.com>",
    });
    expect(summaryActor(item, "alice@example.com")).toEqual({
      principal: "Bob",
      via: "claude-code/2.1.294 <agent@mcp>",
      isYou: false,
    });
  });
});

describe("buildChangeSummary", () => {
  it("counts comments", () => {
    const events = [
      makeItem({ type: "COMMENT" }),
      makeItem({ type: "COMMENT" }),
    ];
    const { parts } = buildChangeSummary(events, "me@test.com");
    expect(parts).toContain("2 new comments");
  });

  it("reports single comment without plural", () => {
    const { parts } = buildChangeSummary([makeItem({ type: "COMMENT" })], "me@test.com");
    expect(parts).toContain("1 new comment");
  });

  it("reports issue creation", () => {
    const { parts } = buildChangeSummary([makeItem({ type: "CREATE" })], "me@test.com");
    expect(parts).toContain("Issue created");
  });

  it("reports merge", () => {
    const { parts } = buildChangeSummary([makeItem({ type: "MERGE" })], "me@test.com");
    expect(parts).toContain("Merged");
  });

  it("reports status changes", () => {
    const events = [
      makeItem({ type: "UPDATE", payload: { status: "DONE" } }),
    ];
    const { parts } = buildChangeSummary(events, "me@test.com");
    expect(parts).toContain("Status completed");
  });

  it("reports assignee changes", () => {
    const events = [
      makeItem({ type: "UPDATE", payload: { assignee: "Bob <bob@example.com>" } }),
    ];
    const { parts } = buildChangeSummary(events, "me@test.com");
    expect(parts).toContain("Reassigned to Bob");
  });

  it("reports assignee removal", () => {
    const events = [
      makeItem({ type: "UPDATE", payload: { assignee: "" } }),
    ];
    const { parts } = buildChangeSummary(events, "me@test.com");
    expect(parts).toContain("Assignee removed");
  });

  it("reports artifact actions", () => {
    const events = [
      makeItem({ type: "ARTIFACT", payload: { filename: "spec.md", action: "created" } }),
    ];
    const { parts } = buildChangeSummary(events, "me@test.com");
    expect(parts).toContain("spec.md created");
  });

  it("counts other updates", () => {
    const events = [
      makeItem({ type: "UPDATE", payload: { title: "changed" } }),
      makeItem({ type: "UPDATE", payload: { description: "changed" } }),
    ];
    const { parts } = buildChangeSummary(events, "me@test.com");
    expect(parts).toContain("2 other updates");
  });

  it("returns empty for no events", () => {
    expect(buildChangeSummary([], "me@test.com")).toEqual({ actor: null, parts: [] });
  });

  it("returns the first agent-on-behalf actor separately from the parts", () => {
    const events = [
      makeItem({ type: "COMMENT" }),
      makeItem({ type: "COMMENT", created_by: "claude-code/2.1.294 <agent@mcp>", on_behalf_of: "Me <me@test.com>" }),
    ];
    const { actor, parts } = buildChangeSummary(events, "me@test.com");
    expect(actor).toEqual({ principal: "Me", via: "claude-code/2.1.294 <agent@mcp>", isYou: true });
    expect(parts).toEqual(["2 new comments"]);
  });

  it("has no actor when nobody acted through an agent", () => {
    expect(buildChangeSummary([makeItem({ type: "COMMENT" })], "me@test.com").actor).toBeNull();
  });
});
