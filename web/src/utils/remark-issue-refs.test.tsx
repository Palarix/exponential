import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import Markdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import { remarkIssueRefs } from "./remark-issue-refs";

function render(md: string, prefix = "xpo"): string {
  return renderToStaticMarkup(
    <Markdown remarkPlugins={[remarkGfm, remarkBreaks, [remarkIssueRefs, { prefix }]]}>{md}</Markdown>,
  );
}

const link = (id: string) => `<a href="#/issues/${id}">${id}</a>`;

describe("remarkIssueRefs", () => {
  it("links IDs in text", () => {
    expect(render("see xpo-a1b2c3 for details")).toBe(`<p>see ${link("xpo-a1b2c3")} for details</p>`);
  });

  it("links IDs inside emphasis and list items", () => {
    expect(render("- **xpo-a1b2c3** done")).toContain(`<strong>${link("xpo-a1b2c3")}</strong>`);
  });

  it("links IDs next to punctuation", () => {
    expect(render("(xpo-a1b2c3).")).toBe(`<p>(${link("xpo-a1b2c3")}).</p>`);
  });

  it("links IDs inside inline code and keeps one code span", () => {
    expect(render("run `xpo show xpo-a1b2c3` now")).toBe(
      `<p>run <code>xpo show ${link("xpo-a1b2c3")}</code> now</p>`,
    );
  });

  it("renders inline code that is only an ID as the bare link", () => {
    expect(render("or just `xpo-a1b2c3`.")).toBe(`<p>or just ${link("xpo-a1b2c3")}.</p>`);
  });

  it("leaves inline code without an ID alone", () => {
    expect(render("`plain code`")).toBe("<p><code>plain code</code></p>");
  });

  it("leaves IDs in fenced code blocks as plain text", () => {
    const html = render("```\nxpo show xpo-a1b2c3\n```");
    expect(html).toContain("xpo show xpo-a1b2c3");
    expect(html).not.toContain("<a");
  });

  it("does not wrap IDs that are already inside a link", () => {
    expect(render("[xpo-a1b2c3](#/issues/xpo-a1b2c3)")).toBe(`<p>${link("xpo-a1b2c3")}</p>`);
    expect(render("[about xpo-a1b2c3](https://example.com)")).toBe(
      `<p><a href="https://example.com">about xpo-a1b2c3</a></p>`,
    );
  });

  it("does not wrap IDs inside reference links or autolinks", () => {
    expect(render("[xpo-a1b2c3][ref]\n\n[ref]: https://example.com")).toBe(
      `<p><a href="https://example.com">xpo-a1b2c3</a></p>`,
    );
    expect(render("https://example.com/xpo-a1b2c3")).toBe(
      `<p><a href="https://example.com/xpo-a1b2c3">https://example.com/xpo-a1b2c3</a></p>`,
    );
  });

  it("ignores IDs with another prefix", () => {
    expect(render("abc-a1b2c3")).toBe("<p>abc-a1b2c3</p>");
    expect(render("abc-a1b2c3", "abc")).toBe(`<p>${link("abc-a1b2c3")}</p>`);
  });
});
