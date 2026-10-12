import { describe, it, expect } from "vitest";
import {
  groupRelations,
  addLink,
  removeLink,
  changeLinkKind,
  linkCandidates,
  subIssueOfCandidates,
  parentOfCandidates,
  changeToKinds,
  PRIMARY_LINK_KINDS,
  MORE_LINK_KINDS,
  relationshipsLabel,
} from "./relation-utils";
import { makeIssue } from "../../test-utils";
import type { Dependency } from "../../api/types";

const A = "xpo-aaa111";
const B = "xpo-bbb222";
const C = "xpo-ccc333";
const D = "xpo-ddd444";

function sectionSummary(sections: ReturnType<typeof groupRelations>) {
  return sections.map(s => [s.label, s.items.map(i => i.targetId)]);
}

describe("groupRelations", () => {
  it("returns no sections for an issue without relations", () => {
    expect(groupRelations(makeIssue({ id: A }), [])).toEqual([]);
  });

  it("groups links by kind in section order with this issue's wording", () => {
    const issue = makeIssue({
      id: A,
      dependencies: [
        { source_id: A, target_id: B, kind: "duplicates" },
        { source_id: A, target_id: C, kind: "blocks" },
        { source_id: A, target_id: D, kind: "relates_to" },
      ],
    });
    const issues = [issue, makeIssue({ id: B }), makeIssue({ id: C }), makeIssue({ id: D })];
    expect(sectionSummary(groupRelations(issue, issues))).toEqual([
      ["Related", [D]],
      ["Blocking", [C]],
      ["Duplicate of", [B]],
    ]);
  });

  it("shows a link stored on the other issue under its inverse wording", () => {
    // B blocks A, stored on B; the API returns it on A as a derived blocked_by.
    const a = makeIssue({ id: A, dependencies: [{ source_id: A, target_id: B, kind: "blocked_by", derived: true }] });
    const b = makeIssue({ id: B, dependencies: [{ source_id: B, target_id: A, kind: "blocks" }] });
    expect(sectionSummary(groupRelations(a, [a, b]))).toEqual([["Blocked by", [B]]]);
    expect(sectionSummary(groupRelations(b, [a, b]))).toEqual([["Blocking", [A]]]);
  });

  it("leaves the parent out; the subtitle shows it", () => {
    const issue = makeIssue({ id: A, parent_id: C, dependencies: [{ source_id: A, target_id: B, kind: "relates_to" }] });
    const issues = [issue, makeIssue({ id: B }), makeIssue({ id: C })];
    expect(sectionSummary(groupRelations(issue, issues))).toEqual([["Related", [B]]]);
  });

  it("shows a relates_to stored on both sides once", () => {
    const issue = makeIssue({
      id: A,
      dependencies: [
        { source_id: A, target_id: B, kind: "relates_to" },
        { source_id: A, target_id: B, kind: "relates_to", derived: true },
      ],
    });
    expect(sectionSummary(groupRelations(issue, [issue, makeIssue({ id: B })]))).toEqual([["Related", [B]]]);
  });

  it("shows a legacy multi-link pair in every section it has a link in", () => {
    const issue = makeIssue({
      id: A,
      dependencies: [
        { source_id: A, target_id: B, kind: "blocks" },
        { source_id: A, target_id: B, kind: "relates_to" },
      ],
    });
    expect(sectionSummary(groupRelations(issue, [issue, makeIssue({ id: B })]))).toEqual([
      ["Related", [B]],
      ["Blocking", [B]],
    ]);
  });

  it("resolves targets and leaves deleted ones unresolved", () => {
    const b = makeIssue({ id: B, title: "Bee" });
    const issue = makeIssue({
      id: A,
      dependencies: [
        { source_id: A, target_id: B, kind: "relates_to" },
        { source_id: A, target_id: "xpo-gone99", kind: "relates_to" },
      ],
    });
    const [related] = groupRelations(issue, [issue, b]);
    expect(related.items[0].target).toBe(b);
    expect(related.items[1].target).toBeUndefined();
  });

  it("labels every link kind", () => {
    const kinds = ["relates_to", "blocked_by", "blocks", "depends_on", "dependency_of", "duplicates", "duplicated_by"];
    const issue = makeIssue({
      id: A,
      dependencies: kinds.map((kind, i) => ({ source_id: A, target_id: `xpo-t${i}`, kind })),
    });
    expect(groupRelations(issue, [issue]).map(s => s.label)).toEqual([
      "Related", "Blocked by", "Blocking", "Depends on", "Dependency of", "Duplicate of", "Duplicated by",
    ]);
  });
});

describe("link set edits", () => {
  const deps: Dependency[] = [
    { source_id: A, target_id: B, kind: "blocks" },
    { source_id: A, target_id: C, kind: "relates_to", derived: true },
  ];

  it("addLink appends a link owned by this issue", () => {
    expect(addLink(A, deps, D, "duplicates")).toEqual([...deps, { source_id: A, target_id: D, kind: "duplicates" }]);
  });

  it("removeLink drops the link whether stored or derived", () => {
    expect(removeLink(deps, B, "blocks")).toEqual([deps[1]]);
    expect(removeLink(deps, C, "relates_to")).toEqual([deps[0]]);
  });

  it("removeLink drops both rows of a relates_to stored on both sides", () => {
    const both: Dependency[] = [
      { source_id: A, target_id: B, kind: "relates_to" },
      { source_id: A, target_id: B, kind: "relates_to", derived: true },
    ];
    expect(removeLink(both, B, "relates_to")).toEqual([]);
  });

  it("removeLink keeps other kinds to the same target", () => {
    const legacy: Dependency[] = [
      { source_id: A, target_id: B, kind: "blocks" },
      { source_id: A, target_id: B, kind: "relates_to" },
    ];
    expect(removeLink(legacy, B, "blocks")).toEqual([legacy[1]]);
  });

  it("changeLinkKind replaces the link in place", () => {
    expect(changeLinkKind(A, deps, B, "blocked_by")).toEqual([
      { source_id: A, target_id: B, kind: "blocked_by" },
      deps[1],
    ]);
  });

  it("changeLinkKind replaces a derived link with one owned by this issue", () => {
    expect(changeLinkKind(A, deps, C, "duplicates")).toEqual([
      deps[0],
      { source_id: A, target_id: C, kind: "duplicates" },
    ]);
  });

  it("changeLinkKind collapses a legacy multi-link pair into the chosen kind", () => {
    const legacy: Dependency[] = [
      { source_id: A, target_id: B, kind: "blocks" },
      { source_id: A, target_id: C, kind: "relates_to" },
      { source_id: A, target_id: B, kind: "relates_to" },
    ];
    expect(changeLinkKind(A, legacy, B, "duplicates")).toEqual([
      { source_id: A, target_id: B, kind: "duplicates" },
      legacy[1],
    ]);
  });
});

describe("changeToKinds", () => {
  it("leaves out the current kind", () => {
    expect(changeToKinds("blocks")).toEqual({
      primary: ["relates_to", "blocked_by", "duplicates"],
      more: MORE_LINK_KINDS,
    });
    expect(changeToKinds("depends_on")).toEqual({
      primary: PRIMARY_LINK_KINDS,
      more: ["dependency_of", "duplicated_by"],
    });
  });
});

describe("candidates", () => {
  it("linkCandidates excludes self and issues linked in either direction", () => {
    const issue = makeIssue({
      id: A,
      dependencies: [
        { source_id: A, target_id: B, kind: "blocks" },
        { source_id: A, target_id: C, kind: "relates_to", derived: true },
      ],
    });
    const issues = [issue, makeIssue({ id: B }), makeIssue({ id: C }), makeIssue({ id: D })];
    expect(linkCandidates(issue, issues, "").map(i => i.id)).toEqual([D]);
  });

  it("filters by trimmed title or ID", () => {
    const issue = makeIssue({ id: A });
    const issues = [issue, makeIssue({ id: B, title: "Fix login" }), makeIssue({ id: C, title: "Other" })];
    expect(linkCandidates(issue, issues, "  login ").map(i => i.id)).toEqual([B]);
    expect(linkCandidates(issue, issues, "ccc").map(i => i.id)).toEqual([C]);
  });

  it("subIssueOfCandidates excludes self, descendants, sub-issues and done issues", () => {
    const issue = makeIssue({ id: A });
    const issues = [
      issue,
      makeIssue({ id: B, parent_id: A }),
      makeIssue({ id: C, status: "DONE" }),
      makeIssue({ id: D }),
      makeIssue({ id: "xpo-eee555", parent_id: D }),
    ];
    expect(subIssueOfCandidates(issue, issues, "").map(i => i.id)).toEqual([D]);
  });

  it("parentOfCandidates excludes self, ancestors and current sub-issues", () => {
    const issue = makeIssue({ id: B, parent_id: A });
    const issues = [
      makeIssue({ id: A }),
      issue,
      makeIssue({ id: C, parent_id: B }),
      makeIssue({ id: D, parent_id: A }),
    ];
    expect(parentOfCandidates(issue, issues, "").map(i => i.id)).toEqual([D]);
  });
});

describe("relationshipsLabel", () => {
  it("reads No relations without parent or links", () => {
    expect(relationshipsLabel(makeIssue({ id: A }), [])).toBe("No relations");
  });

  it("counts a parent alone as one", () => {
    expect(relationshipsLabel(makeIssue({ id: A, parent_id: D }), [])).toBe("1 Relationship");
  });

  it("counts the parent and each shown link", () => {
    const issue = makeIssue({
      id: A,
      parent_id: D,
      dependencies: [
        { source_id: A, target_id: B, kind: "blocks" },
        { source_id: A, target_id: C, kind: "relates_to" },
        { source_id: A, target_id: C, kind: "relates_to", derived: true },
      ],
    });
    expect(relationshipsLabel(issue, [])).toBe("3 Relationships");
  });

  it("uses the singular for one", () => {
    const issue = makeIssue({ id: A, dependencies: [{ source_id: A, target_id: B, kind: "blocks" }] });
    expect(relationshipsLabel(issue, [])).toBe("1 Relationship");
  });
});
