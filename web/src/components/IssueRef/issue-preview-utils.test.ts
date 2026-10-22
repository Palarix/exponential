import { describe, expect, it } from "vitest";
import { makeIssue } from "../../test-utils";
import { previewCounts } from "./issue-preview-utils";

const comment = { id: "c", text: "t", created_by: "a", created_at: "2026-10-01T00:00:00Z" };
const artifact = (artifact_type: string, filename = `${artifact_type}.md`) => ({ artifact_type, filename, updated_at: "", updated_by: "" });
const stats = { branch: "b", head_sha: "abc", commits: 4, files_changed: 3, insertions: 120, deletions: 30, has_uncommitted: false };

describe("previewCounts", () => {
  it("is all empty for a bare issue", () => {
    const issue = makeIssue();
    expect(previewCounts(issue, [issue])).toEqual({
      comments: 0, attachments: 0, spec: false, walkthrough: false, children: null, commits: 0, blockers: 0,
    });
  });

  it("counts comments and commits", () => {
    const issue = makeIssue({ comments: [comment, comment], branch_stats: stats });
    const counts = previewCounts(issue, [issue]);
    expect(counts.comments).toBe(2);
    expect(counts.commits).toBe(4);
  });

  it("tells the spec and walkthrough apart from other attachments", () => {
    const issue = makeIssue({ artifacts: [artifact("spec"), artifact("walkthrough"), artifact("file", "a.png"), artifact("file", "b.txt")] });
    expect(previewCounts(issue, [issue])).toMatchObject({ attachments: 2, spec: true, walkthrough: true });
  });

  it("counts sub-issues done of all", () => {
    const parent = makeIssue({ id: "xpo-p00001" });
    const done = makeIssue({ id: "xpo-c00001", parent_id: parent.id, status: "DONE" });
    const open = makeIssue({ id: "xpo-c00002", parent_id: parent.id, status: "DOING" });
    expect(previewCounts(parent, [parent, done, open]).children).toEqual({ done: 1, total: 2 });
  });

  it("counts blockers that are not done", () => {
    const open = makeIssue({ id: "xpo-b00001", status: "DOING" });
    const done = makeIssue({ id: "xpo-b00002", status: "DONE" });
    const issue = makeIssue({ dependencies: [
      { source_id: "xpo-aaa111", target_id: open.id, kind: "blocked_by" },
      { source_id: "xpo-aaa111", target_id: done.id, kind: "blocked_by" },
    ] });
    expect(previewCounts(issue, [issue, open, done]).blockers).toBe(1);
  });
});
