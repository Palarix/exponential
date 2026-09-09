import { describe, it, expect } from "vitest";
import { buildChildrenByParent, collectKnownPeople } from "./issues";
import { makeIssue } from "../test-utils";

describe("buildChildrenByParent", () => {
  it("returns empty map for issues with no parents", () => {
    const issues = [makeIssue({ id: "a" }), makeIssue({ id: "b" })];
    const map = buildChildrenByParent(issues);
    expect(map.size).toBe(0);
  });

  it("groups children by parent_id", () => {
    const issues = [
      makeIssue({ id: "parent" }),
      makeIssue({ id: "child1", parent_id: "parent" }),
      makeIssue({ id: "child2", parent_id: "parent" }),
      makeIssue({ id: "other-child", parent_id: "other" }),
    ];
    const map = buildChildrenByParent(issues);
    expect(map.size).toBe(2);
    expect(map.get("parent")?.map((i) => i.id)).toEqual(["child1", "child2"]);
    expect(map.get("other")?.map((i) => i.id)).toEqual(["other-child"]);
  });

  it("does not include the parent itself in its children list", () => {
    const issues = [
      makeIssue({ id: "parent" }),
      makeIssue({ id: "child", parent_id: "parent" }),
    ];
    const map = buildChildrenByParent(issues);
    expect(map.get("parent")?.map((i) => i.id)).toEqual(["child"]);
  });

  it("handles empty input", () => {
    expect(buildChildrenByParent([]).size).toBe(0);
  });
});

describe("collectKnownPeople", () => {
  it("collects from contributors", () => {
    const result = collectKnownPeople([], ["Alice <alice@example.com>"]);
    expect(result).toEqual(["Alice <alice@example.com>"]);
  });

  it("collects from issue assignees and creators", () => {
    const issues = [
      makeIssue({ created_by: "Bob <bob@example.com>", assignee: "Carol <carol@example.com>" }),
    ];
    const result = collectKnownPeople(issues, []);
    expect(result).toEqual(["Bob <bob@example.com>", "Carol <carol@example.com>"]);
  });

  it("deduplicates by email", () => {
    const issues = [
      makeIssue({ created_by: "Alice A <alice@example.com>" }),
    ];
    const result = collectKnownPeople(issues, ["Alice <alice@example.com>"]);
    expect(result).toHaveLength(1);
  });

  it("sorts by name", () => {
    const result = collectKnownPeople([], [
      "Zara <zara@example.com>",
      "Alice <alice@example.com>",
      "Mike <mike@example.com>",
    ]);
    expect(result.map(p => p.split(" <")[0])).toEqual(["Alice", "Mike", "Zara"]);
  });

  it("skips undefined assignees", () => {
    const issues = [makeIssue({ assignee: undefined })];
    const result = collectKnownPeople(issues, []);
    expect(result).toHaveLength(1);
  });

  it("excludes entries without valid email", () => {
    const result = collectKnownPeople([], ["No Email", "Agent <bot.local>"]);
    expect(result).toEqual([]);
  });

  it("excludes issue actors without @ in email", () => {
    const issues = [
      makeIssue({ created_by: "Bot <agent>", assignee: "Alice <alice@example.com>" }),
    ];
    const result = collectKnownPeople(issues, []);
    expect(result).toEqual(["Alice <alice@example.com>"]);
  });

  it("returns empty for no input", () => {
    expect(collectKnownPeople([], [])).toEqual([]);
  });
});
