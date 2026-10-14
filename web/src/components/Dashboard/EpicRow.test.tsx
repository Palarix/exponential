import { describe, expect, it } from "vitest";
import type { EpicProgress } from "../../api/client";
import { makeIssue } from "../../test-utils";
import { renderMarkup } from "../../render-markup";
import EpicRow from "./EpicRow";

const epic: EpicProgress = {
  issue_id: "xpo-e1e1e1",
  title: "Ship the thing",
  children_done: 2,
  children_total: 5,
  points_done: 3,
  points_total: 8,
  points_remaining: 5,
  stale: false,
};

/** Fails if any <button> opens before the previous one closed. */
function hasNestedButtons(html: string): boolean {
  let depth = 0;
  for (const tag of html.match(/<\/?button\b/g) ?? []) {
    depth += tag.startsWith("</") ? -1 : 1;
    if (depth > 1) return true;
  }
  return false;
}

describe("EpicRow", () => {
  it("does not nest the copyable ID button inside another button", () => {
    const html = renderMarkup(<EpicRow epic={epic} issue={makeIssue({ id: epic.issue_id })} onOpen={() => {}} />);
    expect(html).toContain(epic.issue_id);
    expect(html).toContain('title="Copy issue ID"');
    expect(hasNestedButtons(html)).toBe(false);
  });

  it("keeps a focusable button that opens the epic", () => {
    const html = renderMarkup(<EpicRow epic={epic} issue={makeIssue({ id: epic.issue_id })} onOpen={() => {}} />);
    expect(html).toMatch(/<button[^>]*>.*Ship the thing.*?<\/button>/);
  });

  it("shows the stale badge for stale epics", () => {
    const html = renderMarkup(<EpicRow epic={{ ...epic, stale: true }} onOpen={() => {}} />);
    expect(html).toContain("STALE");
    expect(hasNestedButtons(html)).toBe(false);
  });
});
