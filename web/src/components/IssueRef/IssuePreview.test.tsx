import { describe, expect, it } from "vitest";
import type { Issue } from "../../api/types";
import { createQueryClient, queryKeys } from "../../api/queries";
import { renderMarkup } from "../../render-markup";
import { makeIssue } from "../../test-utils";
import { IssuePreviewCard } from "./IssuePreview";

const issue = makeIssue({
  id: "xpo-a1b2c3", title: "Fix login redirect loop", status: "DOING", priority: 2,
  assignee: "Ada Lovelace <ada@example.com>", labels: ["bug", "frontend"], estimate: 3,
});

/** The card with the issue list loaded, for sub-issues and blockers. */
function render(card: Issue, issues: Issue[] = [card]): string {
  const qc = createQueryClient();
  qc.setQueryData(queryKeys.issues, issues);
  qc.setQueryData(queryKeys.config, { prefix: "xpo" });
  return renderMarkup(<IssuePreviewCard issue={card} />, qc);
}

describe("IssuePreviewCard", () => {
  it("shows the ID, title and assignee avatar, and leaves out priority, estimate and labels", () => {
    const html = render(issue);
    expect(html).toContain("xpo-a1b2c3");
    expect(html).toContain("Fix login redirect loop");
    expect(html).toContain('title="Ada Lovelace &lt;ada@example.com&gt;"');
    expect(html).not.toContain(">Ada Lovelace<");
    expect(html).not.toContain("High");
    expect(html).not.toContain("Frontend");
  });

  it("has no avatar when unassigned", () => {
    expect(render(makeIssue())).not.toContain("title=");
  });

  it("shows the description as a clamped Markdown excerpt", () => {
    const html = render(makeIssue({ description: "Some **bold** text" }));
    expect(html).toContain("<strong>bold</strong>");
    expect(html).toContain('data-preview="excerpt"');
  });

  it("has no excerpt without a description", () => {
    expect(render(makeIssue())).not.toContain('data-preview="excerpt"');
  });

  it("has no footer when there is nothing to count", () => {
    expect(render(makeIssue())).not.toContain('data-preview="footer"');
  });

  it("shows a footer item for each thing the issue has", () => {
    const parent = makeIssue({
      id: "xpo-p00001",
      comments: [{ id: "c", text: "t", created_by: "a", created_at: "2026-10-01T00:00:00Z" }],
      artifacts: [
        { artifact_type: "spec", filename: "spec.md", updated_at: "", updated_by: "" },
        { artifact_type: "walkthrough", filename: "walkthrough.md", updated_at: "", updated_by: "" },
        { artifact_type: "file", filename: "mock.png", updated_at: "", updated_by: "" },
      ],
      branch_stats: { branch: "b", head_sha: "abc", commits: 7, files_changed: 1, insertions: 1, deletions: 0, has_uncommitted: false },
      dependencies: [{ source_id: "xpo-p00001", target_id: "xpo-b00001", kind: "blocked_by" }],
    });
    const child = makeIssue({ id: "xpo-c00001", parent_id: parent.id, status: "DONE" });
    const blocker = makeIssue({ id: "xpo-b00001", status: "DOING" });
    const html = render(parent, [parent, child, blocker]);
    expect(html).toContain('data-preview="footer"');
    for (const label of ["1 comment", "1 attachment", "Spec", "Walkthrough", "1 of 1 sub-issues done", "7 commits", "Blocked by 1 issue"]) {
      expect(html).toContain(`aria-label="${label}"`);
    }
  });

  it("leaves out footer items with nothing to count", () => {
    const html = render(makeIssue({ comments: [{ id: "c", text: "t", created_by: "a", created_at: "2026-10-01T00:00:00Z" }] }));
    expect(html).toContain('aria-label="1 comment"');
    expect(html).not.toMatch(/aria-label="[^"]*(attachment|Spec|Walkthrough|sub-issues|commit|Blocked)/);
  });
});
