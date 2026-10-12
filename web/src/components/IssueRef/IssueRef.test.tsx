import { describe, expect, it } from "vitest";
import type { ReactElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { QueryClientProvider } from "@tanstack/react-query";
import { createQueryClient, queryKeys } from "../../api/queries";
import { makeIssue } from "../../test-utils";
import { IssueRef, IssueRefText, MarkdownLink } from "./IssueRef";

const known = makeIssue({ id: "xpo-a1b2c3", title: "Known issue" });

function render(el: ReactElement, issues: unknown[] | null = [known]): string {
  const qc = createQueryClient();
  if (issues) qc.setQueryData(queryKeys.issues, issues);
  qc.setQueryData(queryKeys.config, { prefix: "xpo" });
  return renderToStaticMarkup(<QueryClientProvider client={qc}>{el}</QueryClientProvider>);
}

describe("IssueRef", () => {
  it("renders a known issue as a chip link", () => {
    const html = render(<IssueRef id="xpo-a1b2c3" />);
    expect(html).toMatch(/^<a [^>]*href="#\/issues\/xpo-a1b2c3"[^>]*data-issue-ref[^>]*>xpo-a1b2c3<\/a>$/);
  });

  it("renders an unknown issue as muted text with a not found hint", () => {
    const html = render(<IssueRef id="xpo-ffffff" />);
    expect(html).not.toContain("<a");
    expect(html).toContain('title="Issue not found"');
    expect(html).toContain("xpo-ffffff");
  });

  it("stays a link while the issues are still loading", () => {
    const html = render(<IssueRef id="xpo-ffffff" />, null);
    expect(html).toContain('href="#/issues/xpo-ffffff"');
  });
});

describe("MarkdownLink", () => {
  it("renders issue hrefs as chips", () => {
    const html = render(<MarkdownLink href="#/issues/xpo-a1b2c3">xpo-a1b2c3</MarkdownLink>);
    expect(html).toContain("data-issue-ref");
  });

  it("keeps custom link text for issue hrefs in a label chip", () => {
    const html = render(<MarkdownLink href="#/issues/xpo-a1b2c3">the epic</MarkdownLink>);
    expect(html).toMatch(/^<a [^>]*class="issue-ref issue-ref-label"[^>]*>the epic<\/a>$/);
  });

  it("renders other hrefs as plain links", () => {
    const html = render(<MarkdownLink href="https://example.com">site</MarkdownLink>);
    expect(html).toBe('<a href="https://example.com">site</a>');
  });
});

describe("IssueRefText", () => {
  it("renders the IDs in plain text as chips", () => {
    const html = render(<IssueRefText text="see xpo-a1b2c3 & more" />);
    expect(html).toMatch(/^see <a [^>]*>xpo-a1b2c3<\/a> &amp; more$/);
  });
});
