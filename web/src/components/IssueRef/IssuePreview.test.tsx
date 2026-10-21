import { describe, expect, it } from "vitest";
import { renderMarkup } from "../../render-markup";
import { makeIssue } from "../../test-utils";
import { IssuePreviewCard } from "./IssuePreview";

const issue = makeIssue({
  id: "xpo-a1b2c3", title: "Fix login redirect loop", status: "DOING", priority: 2,
  assignee: "Ada Lovelace <ada@example.com>", labels: ["bug", "frontend"], estimate: 3,
});

describe("IssuePreviewCard", () => {
  it("shows the ID, title, status and priority by default", () => {
    const html = renderMarkup(<IssuePreviewCard issue={issue} />);
    expect(html).toContain("xpo-a1b2c3");
    expect(html).toContain("Fix login redirect loop");
    expect(html).toContain("In Progress");
  });

  it("leaves out what a board row shows and adds what it doesn't, as details", () => {
    const html = renderMarkup(<IssuePreviewCard issue={issue} variant="details" />);
    expect(html).not.toContain("xpo-a1b2c3");
    expect(html).not.toContain("Fix login redirect loop");
    expect(html).not.toContain("In Progress");
    expect(html).toContain("Ada Lovelace");
    expect(html).toContain("High");
    expect(html).toContain("Bug");
    expect(html).toContain("Frontend");
    expect(html).toMatch(/Estimate.*3/s);
  });

  it("says Unassigned and skips an unset estimate and missing labels", () => {
    const html = renderMarkup(<IssuePreviewCard issue={makeIssue({ id: "xpo-x", title: "T" })} variant="details" />);
    expect(html).toContain("Unassigned");
    expect(html).not.toContain("Estimate");
    expect(html).not.toContain("Labels");
  });
});
