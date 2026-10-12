import { describe, expect, it } from "vitest";
import type { ComponentProps } from "react";
import { DndContext } from "@dnd-kit/core";
import { makeIssue } from "../../test-utils";
import { renderMarkup } from "../../render-markup";
import { BacklogIssueRow } from "./BacklogIssueRow";

const noop = () => {};

function renderRow(overrides: Partial<ComponentProps<typeof BacklogIssueRow>> = {}): string {
  const props: ComponentProps<typeof BacklogIssueRow> = {
    issue: makeIssue(),
    depth: 0,
    hasChildren: false,
    hasVisibleChildren: false,
    childDone: 0,
    childTotal: 0,
    childPointsDone: 0,
    childPointsTotal: 0,
    treeGuides: [],
    rowIndex: 0,
    hierarchyMode: "flat",
    isFocused: false,
    keyboardNav: false,
    isNodeExpanded: false,
    canDrag: false,
    isDropTarget: false,
    dndId: "xpo-aaa111",
    isDraggedOrBatch: false,
    dragBatchCount: 0,
    popoverType: null,
    allKnownLabels: [],
    knownPeople: ["Alice Smith <alice@x.io>"],
    onToggleNode: noop,
    onMouseEnter: noop,
    onOpenPopover: noop,
    onClosePopover: noop,
    onQuickStatus: noop,
    onQuickPriority: noop,
    onQuickEstimate: noop,
    onQuickAssignee: noop,
    onQuickLabelToggle: noop,
    ...overrides,
  };
  return renderMarkup(
    <DndContext>
      <BacklogIssueRow {...props} />
    </DndContext>,
  );
}

/** The assignee trigger: the only button whose accessible name mentions the assignee. */
function assigneeTrigger(html: string): string {
  const start = html.search(/<button[^>]*aria-label="[^"]*change assignee"/);
  expect(start).toBeGreaterThanOrEqual(0);
  return html.slice(start, html.indexOf("</button>", start));
}

describe("BacklogIssueRow assignee cell", () => {
  it("shows a dashed placeholder trigger when unassigned", () => {
    const trigger = assigneeTrigger(renderRow());
    expect(trigger).toContain('aria-label="No assignee, change assignee"');
    expect(trigger).toContain('aria-haspopup="menu"');
    expect(trigger).toContain('aria-expanded="false"');
    expect(trigger).toContain("border-dashed");
  });

  it("shows the assignee's avatar as the trigger when assigned", () => {
    const trigger = assigneeTrigger(renderRow({ issue: makeIssue({ assignee: "Alice Smith <alice@x.io>" }) }));
    expect(trigger).toContain('aria-label="Alice Smith, change assignee"');
    expect(trigger).not.toContain("border-dashed");
  });
});
