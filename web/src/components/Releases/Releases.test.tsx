import { describe, expect, it } from "vitest";
import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import { renderMarkup } from "../../render-markup";
import type { Release, ReleasesResponse } from "../../api/types";
import { makeIssue } from "../../test-utils";
import { ReleaseStack } from "./Releases";
import { EMPTY_FILTERS, type BacklogFilters } from "../Backlog/filters";

const noop = () => {};
const rel = (number: number, o: Partial<Release> = {}): Release => ({
  id: `rel-${number}`, number, name: o.label ?? `#${number}`, state: "OPEN", counts: {}, total: 0, done: 0, ...o,
});
const past = (number: number) =>
  rel(number, { label: `0.${number}`, state: "RELEASED", slot: "past", released_at: "2026-09-01T10:00:00Z" });

const delivered = makeIssue({ id: "xpo-dl1", title: "Delivered thing", status: "DONE", release_id: "rel-1" });
const epic = makeIssue({ id: "xpo-ep1", title: "Courier app", release_id: "rel-3", release: "1.2" });
const epicKids = [
  makeIssue({ id: "xpo-ck1", title: "Kid in 1.1", parent_id: "xpo-ep1", release_id: "rel-2" }),
  makeIssue({ id: "xpo-ck2", title: "Kid without a release", parent_id: "xpo-ep1" }),
  makeIssue({ id: "xpo-ck3", title: "Kid in 1.2", parent_id: "xpo-ep1", release_id: "rel-3" }),
];

function stack(board: ReleasesResponse, query = "", filters: BacklogFilters = EMPTY_FILTERS) {
  return renderMarkup(
    <TooltipPrimitive.Provider>
      <ReleaseStack
        board={board}
        issues={[makeIssue({ id: "xpo-un1", title: "Loose one" }), makeIssue({ id: "xpo-in1", title: "In 1.1", release_id: "rel-2" }), delivered, epic, ...epicKids]}
        today="2026-10-20"
        query={query}
        filters={filters}
        onOpen={noop}
        onIssueClick={noop}
        onReorder={noop}
        onAssign={noop}
      />
    </TooltipPrimitive.Provider>,
  );
}

const board: ReleasesResponse = {
  current: rel(2, { label: "1.1", slot: "current" }),
  next: rel(3, { label: "1.2", slot: "next" }),
  upcoming: [rel(4, { label: "Team Mode", slot: "upcoming" })],
  past: [past(1)],
};

describe("ReleaseStack", () => {
  it("is one column: the future on top, Current, No release, then Past", () => {
    const html = stack(board);
    const at = (s: string) => html.indexOf(s);
    expect(html).not.toContain("Create release");
    expect(at(">Team Mode<")).toBeLessThan(at(">1.2<"));
    expect(at(">1.2<")).toBeLessThan(at(">1.1<"));
    expect(at(">1.1<")).toBeLessThan(at(">No release<"));
    expect(at(">No release<")).toBeLessThan(at(">0.1<"));
    expect(html).not.toContain("grid-cols-2");
  });

  it("gives placeholders no row and no badge", () => {
    const html = stack({ ...board, next: rel(3, { id: undefined, slot: "next", placeholder: true }), upcoming: [] });
    expect(html).not.toContain(">#3<");
    expect(html).not.toMatch(/>Next</);
    expect(html).toMatch(/>Current</);
  });

  it("opens No release by default, with its open issues and their count", () => {
    const html = stack(board);
    const noRelease = html.slice(html.indexOf(">No release<"), html.indexOf(">0.1<"));
    expect(noRelease).toContain("Loose one");
    expect(noRelease).toMatch(/aria-expanded="true"[^>]*aria-label="Hide issues"/);
    expect(html).not.toContain("In 1.1");
  });

  it("opens the rows with search matches and collapses No release when nothing there matches", () => {
    const html = stack(board, "in 1.1");
    expect(html).toContain("In 1.1");
    expect(html).toMatch(/aria-expanded="true"[^>]*aria-label="Hide issues"/);
    expect(html).not.toContain("Loose one");
    expect(html.slice(html.indexOf(">No release<"), html.indexOf(">0.1<"))).toMatch(/aria-expanded="false"[^>]*aria-label="Show issues"/);
  });

  it("lists what a past release delivered, read-only, when the search finds it", () => {
    const html = stack(board, "delivered");
    const row = html.match(/<div[^>]*role="button"[^>]*>(?:(?!role="button").)*Delivered thing/s)?.[0] ?? "";
    expect(row).toContain("Delivered thing");
    expect(row).not.toContain('aria-roledescription="draggable"');
  });

  it("keeps the issues of open releases draggable", () => {
    const html = stack(board, "in 1.1");
    expect(html).toMatch(/<div[^>]*aria-roledescription="draggable"[^>]*>(?:(?!role="button").)*In 1\.1/s);
  });

  it("calls the no-release row No release", () => {
    const html = stack(board);
    expect(html).toContain(">No release<");
    expect(html).not.toContain("Unassigned");
  });

  it("lists each issue only where it is, with a parent breadcrumb that says where the parent is", () => {
    const html = stack(board, "kid");
    expect(html).toContain('aria-label="Parent: xpo-ep1 Courier app · in 1.2"');
    // As in the backlog: the parent comes before the title, in the same row.
    const row = html.match(/<div[^>]*role="button"[^>]*>(?:(?!role="button").)*Kid in 1\.1/s)?.[0] ?? "";
    expect(row).toContain('aria-label="Parent: xpo-ep1 Courier app · in 1.2"');
    expect(html).not.toContain("children in this release");
    // The search matches only the children: the epic's own row shows nowhere, not even as a stand-in.
    expect(html).not.toContain(">xpo-ep1<");
  });

  it("shows a parent's child progress on its row", () => {
    const html = stack(board, "courier");
    expect(html).toMatch(/Courier app(?:(?!role="button").)*0\/3/s);
  });

  it("keeps issue rows compact: ID, status and title in the backlog's order, no priority or labels", () => {
    const html = stack(board, "in 1.1");
    const row = html.match(/<div[^>]*role="button"[^>]*>(?:(?!role="button").)*In 1\.1.*?<\/div>/s)?.[0] ?? "";
    expect(row).toMatch(/title="Copy issue ID"[^>]*>xpo-in1</);
    expect(row.indexOf("xpo-in1")).toBeLessThan(row.indexOf("In 1.1"));
    expect(row).not.toMatch(/aria-label="[^"]*[Pp]riority/);
  });

  it("narrows every list with the filters and opens the rows with matches", () => {
    const html = stack(board, "", { ...EMPTY_FILTERS, statuses: ["DONE"] });
    expect(html).toContain("Delivered thing");
    expect(html).not.toContain("Loose one");
    expect(html).not.toContain("In 1.1");
    expect(html.slice(html.indexOf(">No release<"), html.indexOf(">0.1<"))).toMatch(/aria-expanded="false"[^>]*aria-label="Show issues"/);
  });

  it("keeps releases collapsed while there is no search", () => {
    const html = stack(board);
    expect(html).not.toContain("In 1.1");
    expect(html.slice(0, html.indexOf(">No release<"))).not.toMatch(/aria-label="Hide issues"/);
  });

  it("shows the three latest past releases and offers the rest", () => {
    const html = stack({ ...board, past: [past(9), past(8), past(7), past(6), past(5)] });
    expect(html).toContain(">0.7<");
    expect(html).not.toContain(">0.6<");
    expect(html).toContain("Show 2 older releases");
  });
});
