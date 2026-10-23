import { describe, expect, it } from "vitest";
import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import { QueryClient } from "@tanstack/react-query";
import type { ReactElement } from "react";
import { renderMarkup } from "../../render-markup";
import { queryKeys } from "../../api/query-keys";
import type { Release } from "../../api/types";
import { ReleaseRowBody, ReleaseSummaryCard } from "./ReleaseRow";

const noop = () => {};
const current: Release = {
  id: "rel-2",
  number: 2,
  name: "1.1",
  label: "1.1",
  goal: "Live courier tracking",
  state: "OPEN",
  slot: "current",
  started_at: "2026-10-03T09:00:00Z",
  target_date: "2026-11-15",
  counts: { DOING: 2, PLANNED: 1, DONE: 3 },
  total: 6,
  done: 3,
  points: { total: 13, done: 5 },
};

const render = (el: ReactElement, client?: QueryClient) =>
  renderMarkup(<TooltipPrimitive.Provider>{el}</TooltipPrimitive.Provider>, client);

function row(release: Release, extra: Partial<Parameters<typeof ReleaseRowBody>[0]> = {}) {
  return render(
    <ReleaseRowBody
      release={release}
      today="2026-10-20"
      handle={<span aria-label="Drag to reorder" />}
      expanded={false}
      onToggle={noop}
      onOpen={noop}
      {...extra}
    />,
  );
}

describe("ReleaseRowBody", () => {
  it("reads as one title: the label, the goal, then the slot badge", () => {
    const html = row(current);
    expect(html).toMatch(/>1\.1<\/button><span[^>]*> – Live courier tracking<\/span><\/span>/);
    expect(html.indexOf("Live courier tracking")).toBeLessThan(html.indexOf(">Current<"));
    expect(html.indexOf("Nov 15")).toBeLessThan(html.indexOf(">Current<"));
  });

  it("shows the number as the name until the release has a label", () => {
    const html = row({ ...current, label: undefined, name: "#2" });
    expect(html).toContain(">#2<");
    expect(html).not.toContain("Unnamed");
  });

  it("puts the target date and the done/total progress on the title line", () => {
    const html = row(current);
    expect(html).toContain("Nov 15");
    expect(html).toMatch(/aria-label="3 of 6 issues done\. Click for story points"/);
    expect(html).toMatch(/aria-expanded="false"[^>]*aria-label="Show issues"/);
  });

  it("shows story points done of total in points mode", () => {
    const html = row(current, { progressMode: "points" });
    expect(html).toMatch(/aria-label="5 of 13 story points done\. Click for issue count"/);
    expect(html).toContain("5/13");
    expect(html).not.toContain("3/6");
  });

  it("shows the ring in issues mode and the triangle instead in points mode", () => {
    const pill = (html: string) => {
      const start = html.indexOf(" done. Click for");
      return html.slice(start, html.indexOf("</button>", start));
    };
    expect(pill(row(current))).toContain("<circle");
    const points = pill(row(current, { progressMode: "points" }));
    expect(points).toContain("lucide-triangle");
    expect(points).not.toContain("<circle");
    expect(pill(row({ ...current, points: { total: 13, done: 13 } }, { progressMode: "points" }))).not.toContain("circle-check");
  });

  it("raises the progress pill, a button, off the row", () => {
    const html = row(current);
    expect(html).toMatch(/<button[^>]*aria-label="3 of 6 issues done[^"]*" class="[^"]*bg-\[var\(--color-surface-2\)\]/);
  });

  it("hides the pill in points mode for a past release without stored points", () => {
    const old = { ...current, state: "RELEASED" as const, slot: "past" as const, released_at: "2026-10-02T10:00:00Z", points: undefined };
    expect(row(old, { handle: null, progressMode: "points" })).not.toMatch(/story points done/);
    expect(row(old, { handle: null })).toMatch(/aria-label="3 of 6 issues done/);
  });

  it("marks a target date that passed with open scope", () => {
    expect(row({ ...current, target_date: "2026-10-10" })).toMatch(/text-\[var\(--color-warning\)\][^>]*>Oct 10</);
  });

  it("shows the order warning with the conflicting releases", () => {
    expect(row(current, { warning: ["1.2"] })).toContain('aria-label="Target date conflicts with 1.2"');
  });

  it("shows a past release faded, with its release date, and lets it expand", () => {
    const html = row({ ...current, state: "RELEASED", slot: "past", released_at: "2026-10-02T10:00:00Z" }, { handle: null });
    expect(html).toContain("opacity-60");
    expect(html).toContain("Oct 2");
    expect(html).toContain("Show issues");
    expect(html).not.toMatch(/>Current</);
  });

  it("makes a past release's name a toggle for its issues, not a link", () => {
    const html = row({ ...current, state: "RELEASED", slot: "past", released_at: "2026-10-02T10:00:00Z" }, { handle: null, expanded: true });
    expect(html).toMatch(/<button[^>]*aria-expanded="true"[^>]*>1\.1<\/button>/);
  });

  it("makes only an open release's name a link, not the goal or the badge", () => {
    const html = row(current);
    expect(html).toMatch(/<button[^>]*title="Show in Issues"[^>]*>1\.1<\/button><span[^>]*> – Live courier tracking/);
    expect(html).not.toMatch(/<button[^>]*><span class="flex items-center gap-2 min-w-0">/);
  });

  it("puts the progress right after the title", () => {
    const html = row(current);
    const progress = html.indexOf('aria-label="3 of 6 issues done. Click for story points"');
    expect(html.indexOf("Live courier tracking")).toBeLessThan(progress);
    expect(progress).toBeLessThan(html.indexOf("Nov 15"));
  });

  it("puts the slot badge at the far right, just before the actions", () => {
    const html = row(current, { actions: <span aria-label="Release actions" /> });
    expect(html.indexOf("Nov 15")).toBeLessThan(html.indexOf(">Current<"));
    expect(html.indexOf(">Current<")).toBeLessThan(html.indexOf('aria-label="Release actions"'));
  });

  it("keeps the chevron rightmost, with the actions just left of it", () => {
    const html = row(current, { actions: <span aria-label="Release actions" /> });
    expect(html.indexOf("Nov 15")).toBeLessThan(html.indexOf('aria-label="Release actions"'));
    expect(html.indexOf('aria-label="Release actions"')).toBeLessThan(html.indexOf('aria-label="Show issues"'));
    const afterChevron = html.slice(html.indexOf('aria-label="Show issues"'));
    expect(afterChevron.slice(afterChevron.indexOf("</button>"))).toMatch(/^<\/button><\/div>/);
  });
});

describe("ReleaseSummaryCard", () => {
  it("breaks the scope down by status, with the target date and the burndown", () => {
    const client = new QueryClient();
    client.setQueryData(queryKeys.releaseProgress("#2"), {
      days: [
        { date: "2026-10-03", scope: 5, open: 5, completed: 0 },
        { date: "2026-10-04", scope: 6, open: 3, completed: 3 },
      ],
    });
    const html = render(<ReleaseSummaryCard release={current} today="2026-10-20" />, client);
    expect(html).toContain("In progress");
    expect(html).toContain("3/6 done · 5/13 pts");
    expect(html).toMatch(/aria-label="Time to target date"/);
    expect(html).toMatch(/aria-label="Burndown"/);
  });

  it("has no burndown for a release that was never Current", () => {
    const html = render(<ReleaseSummaryCard release={{ ...current, slot: "next", started_at: undefined }} today="2026-10-20" />);
    expect(html).not.toMatch(/aria-label="Burndown"/);
    expect(html).toContain("Starts when this release becomes Current.");
  });

  it("shows a past release's final stats; its burndown is in the expanded row", () => {
    const released = { ...current, state: "RELEASED" as const, slot: "past" as const, released_at: "2026-10-09T10:00:00Z" };
    const html = render(<ReleaseSummaryCard release={released} today="2026-10-20" />);
    expect(html).toContain("In progress");
    expect(html).not.toContain("Burndown");
  });

  it("leaves the points out of the header when the release has none", () => {
    const html = render(<ReleaseSummaryCard release={{ ...current, points: undefined }} today="2026-10-20" />);
    expect(html).toContain("3/6 done");
    expect(html).not.toContain("pts");
  });
});
