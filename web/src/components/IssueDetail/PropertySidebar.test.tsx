import { describe, expect, it } from "vitest";
import { makeIssue } from "../../test-utils";
import { renderMarkup } from "../../render-markup";
import PropertySidebar from "./PropertySidebar";
import { popoverTriggerProps } from "./popover-trigger";

const issue = makeIssue({
  estimate: 2,
  priority: 1,
  assignee: "Alice <alice@x.io>",
  labels: ["bug"],
});

function renderSidebar(): string {
  return renderMarkup(
    <PropertySidebar
      issue={issue}
      issues={[issue]}
      openPopover={null}
      setOpenPopover={() => {}}
      saveUpdate={() => {}}
      onClose={() => {}}
      contributors={[]}
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

describe("popoverTriggerProps", () => {
  it("reports expanded only for the open popover", () => {
    expect(popoverTriggerProps("status", "status")).toEqual({ "aria-haspopup": "menu", "aria-expanded": true });
    expect(popoverTriggerProps("status", "priority")).toEqual({ "aria-haspopup": "menu", "aria-expanded": false });
    expect(popoverTriggerProps("status", null)).toEqual({ "aria-haspopup": "menu", "aria-expanded": false });
  });
});
