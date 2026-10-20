import { describe, expect, it } from "vitest";
import { renderMarkup } from "../../render-markup";
import type { Release } from "../../api/types";
import { makeIssue } from "../../test-utils";
import { MoveChildrenBody } from "./ReleaseAssign";

const noop = () => {};
const target: Release = { id: "rel-2", number: 2, name: "1.2", label: "1.2", state: "OPEN", counts: {}, total: 0, done: 0 };
const epic = makeIssue({ id: "xpo-ep1", title: "Courier app" });
const kids = [makeIssue({ id: "xpo-k1", title: "Map view" }), makeIssue({ id: "xpo-k2", title: "Push" })];

function body(doneCount: number, t: Release | null = target) {
  return renderMarkup(
    <MoveChildrenBody prompt={{ epics: [epic], children: kids, doneCount }} target={t} onAbort={noop} onJustThis={noop} onMoveAll={noop} />,
  );
}

describe("MoveChildrenBody", () => {
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

  it("offers the three choices", () => {
    const html = body(0);
    expect(html).toContain("Abort");
    expect(html).toContain("Just this issue");
    expect(html).toContain("Move all");
  });

  it("speaks of removing for No release", () => {
    expect(body(0, null)).toContain("Remove them from their release too?");
  });
});
