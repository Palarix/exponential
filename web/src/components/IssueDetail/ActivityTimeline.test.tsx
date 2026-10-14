import { describe, expect, it } from "vitest";
import { createRef } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { QueryClientProvider } from "@tanstack/react-query";
import { createQueryClient, queryKeys } from "../../api/queries";
import type { HistoryEvent } from "../../api/client";
import { makeIssue } from "../../test-utils";
import ActivityTimeline from "./ActivityTimeline";

const issue = makeIssue({ id: "xpo-a1b2c3" });

function render(history: HistoryEvent[]): string {
  const qc = createQueryClient();
  qc.setQueryData(queryKeys.issues, [issue]);
  qc.setQueryData(queryKeys.config, { prefix: "xpo" });
  qc.setQueryData(queryKeys.issueHistory(issue), history);
  return renderToStaticMarkup(
    <QueryClientProvider client={qc}>
      <ActivityTimeline
        issue={issue}
        newComment=""
        onNewCommentChange={() => {}}
        onAddComment={() => {}}
        saving={false}
        commentRef={createRef<HTMLTextAreaElement>()}
      />
    </QueryClientProvider>,
  );
}

const update: HistoryEvent = {
  type: "UPDATE",
  payload: { priority: 2, estimate: 3 },
  created_at: "2026-10-14T00:00:00Z",
  created_by: "Alice Smith <alice@example.com>",
};

const HOVER = "hover:text-\\[var\\(--color-text-primary\\)\\]";

// The text part of the system row: from its marker to the end of the row.
function systemRow(html: string): string {
  const start = html.indexOf('data-activity="system"');
  expect(start).toBeGreaterThan(-1);
  const tagStart = html.lastIndexOf("<", start);
  return html.slice(tagStart, html.indexOf("</div>", start));
}

// Bug xpo-ca763f: system rows mixed muted and primary text, and the row's flex
// gap put 8px between every part.
describe("ActivityTimeline system rows", () => {
  it("keeps the actor name and values muted, highlighting them on hover", () => {
    const row = systemRow(render([update]));
    // Nothing in a system row is primary until hovered.
    expect(row.replace(/hover:text-\[var\(--color-text-primary\)\]/g, "")).not.toContain(
      "text-[var(--color-text-primary)]",
    );
    expect(row).toMatch(
      /<span class="[^"]*hover:text-\[var\(--color-text-primary\)\][^"]*"><span[^>]*>Alice Smith<\/span><\/span>/,
    );
    // Hoverable parts show a pointer cursor.
    expect(row).toMatch(/<span class="[^"]*cursor-pointer[^"]*"><span[^>]*>Alice Smith<\/span><\/span>/);
    // Each value's chip carries the hover highlight; its wrapped icon comes first.
    const html = row.replace(/<!-- -->/g, "");
    for (const value of ["3 Points", "High"]) {
      expect(html).toMatch(new RegExp(`<span class="[^"]*${HOVER}[^"]*"><span[^>]*><svg[\\s\\S]*?</svg></span>${value}</span>`));
    }
  });

  // Bug xpo-d3578a: inline chips sat off the row's text line, so the row and
  // its chips are flex boxes that center their items.
  it("centers chips and text in a wrapping flex row", () => {
    const row = systemRow(render([update]));
    const containerClass = row.match(/^<div[^>]* class="([^"]*)"/)?.[1] ?? "";
    for (const cls of ["flex", "flex-wrap", "items-center", "gap-x-1"]) {
      expect(containerClass.split(" ")).toContain(cls);
    }
    const html = row.replace(/<!-- -->/g, "");
    for (const value of ["3 Points", "High"]) {
      const chipClass = html.match(new RegExp(`<span class="([^"]*)"><span[^>]*><svg[\\s\\S]*?</svg></span>${value}</span>`))?.[1] ?? "";
      expect(chipClass.split(" ")).toEqual(expect.arrayContaining(["inline-flex", "items-center"]));
    }
  });

  it("reads as one sentence", () => {
    const row = systemRow(render([update]));
    const text = row.replace(/<[^>]+>/g, "").replace(/\s+/g, " ").trim();
    const [sentence, time] = text.split(" · ");
    expect(sentence).toBe("Alice Smith set estimate to 3 Points and set priority to High");
    expect(time).toBeTruthy();
  });
});
