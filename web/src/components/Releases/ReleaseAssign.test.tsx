import { describe, expect, it } from "vitest";
import { renderMarkup } from "../../render-markup";
import type { Release } from "../../api/types";
import { makeIssue } from "../../test-utils";
import { MoveChildrenPanel } from "./ReleaseAssign";

const noop = () => {};
const target: Release = { id: "rel-2", number: 2, name: "1.2", label: "1.2", state: "OPEN", counts: {}, total: 0, done: 0 };
const epic = makeIssue({ id: "xpo-ep1", title: "Courier app" });
const kids = [makeIssue({ id: "xpo-k1", title: "Map view" }), makeIssue({ id: "xpo-k2", title: "Push" })];

function body(doneCount: number, t: Release | null = target) {
  return renderMarkup(
    <MoveChildrenPanel id="m" prompt={{ epics: [epic], children: kids, doneCount }} target={t} onAbort={noop} onJustThis={noop} onMoveAll={noop} />,
  );
}

describe("MoveChildrenPanel", () => {
  it("names the epic, the count and the target, and lists the children", () => {
    const html = body(0);
    expect(html).toContain("Courier app");
    expect(html).toContain("2 open sub-issues");
    expect(html).toContain("1.2");
    expect(html).toContain("Map view");
    expect(html).toContain("Push");
    expect(html).not.toContain("completed");
  });

  it("says done children stay", () => {
    expect(body(1)).toContain("1 completed issue stays where it is");
    expect(body(3)).toContain("3 completed issues stay where they are");
  });

  it("is a dialog titled with the question", () => {
    expect(body(0)).toMatch(/id="m-title"[^>]*>Move sub-issues too\?</);
  });

  it("offers the three choices in the footer, Abort on the left", () => {
    const footer = body(0).match(/<div[^>]*data-slot="footer"[^>]*>.*$/s)?.[0];
    expect(footer).toMatch(/>Abort<.*>Just this issue<.*>Move all</s);
  });

  it("speaks of removing for No release", () => {
    expect(body(0, null)).toContain("Remove them from their release too?");
  });
});
