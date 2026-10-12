import { describe, expect, it } from "vitest";
import { makeIssue } from "../../test-utils";
import { renderMarkup } from "../../render-markup";
import PropertySidebar from "./PropertySidebar";

const issue = makeIssue({
  estimate: 2,
  priority: 1,
  assignee: "Alice <alice@x.io>",
  labels: ["bug"],
});

function renderSidebar(target = issue, gitActions?: boolean): string {
  return renderMarkup(
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
    />,
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
});
