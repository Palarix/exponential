import { describe, expect, it } from "vitest";
import { renderMarkup } from "../../render-markup";
import { Dialog } from "./Dialog";

/** The markup of one slot, from its opening tag to the end of the dialog. */
const slot = (html: string, name: string) => html.match(new RegExp(`<div[^>]*data-slot="${name}"[^>]*>.*$`, "s"))?.[0];

describe("Dialog", () => {
  const html = renderMarkup(
    <Dialog id="d" title="Edit it" footer={<button>Save</button>}>
      <div>the body</div>
    </Dialog>,
  );

  it("is a modal dialog labelled by its title", () => {
    expect(html).toMatch(/role="dialog"[^>]*aria-modal="true"[^>]*aria-labelledby="d-title"/);
    expect(html).toMatch(/<h2 id="d-title"[^>]*>Edit it</);
  });

  it("takes an alertdialog role and a description", () => {
    const alert = renderMarkup(<Dialog id="d" role="alertdialog" describedBy="d-desc" title="t">b</Dialog>);
    expect(alert).toMatch(/role="alertdialog"[^>]*aria-describedby="d-desc"/);
  });

  it("orders header, body and footer, with no dividers on one surface", () => {
    const header = html.indexOf('data-slot="header"');
    const body = html.indexOf('data-slot="body"');
    const footer = html.indexOf('data-slot="footer"');
    expect(header).toBeGreaterThan(-1);
    expect(body).toBeGreaterThan(header);
    expect(footer).toBeGreaterThan(body);
    expect(html).not.toMatch(/border-[tb] /);
    expect(html).not.toContain("surface-2");
  });

  it("pads 16px on every edge without doubling it between slots", () => {
    expect(slot(html, "header")).toMatch(/^<div[^>]*class="[^"]*\bp-4 pb-0\b/);
    expect(slot(html, "body")).toMatch(/^<div[^>]*class="[^"]*\bp-4\b/);
    expect(slot(html, "footer")).toMatch(/^<div[^>]*class="[^"]*\bp-4 pt-0\b/);
  });

  it("scrolls only the body, inside an 85vh cap", () => {
    const card = html.match(/role="dialog"[^>]*class="([^"]*)"/)?.[1];
    expect(card).toMatch(/\bmax-h-\[85vh\]/);
    expect(card).toMatch(/\bflex-col\b/);
    expect(slot(html, "body")).toMatch(/^<div[^>]*class="[^"]*overflow-y-auto/);
  });

  it("shows the esc hint, then the buttons, in the footer", () => {
    expect(slot(html, "footer")).toMatch(/<kbd[^>]*>esc<\/kbd>.*to cancel.*>Save</s);
  });

  it("replaces the esc hint with footerLeft", () => {
    const withLeft = renderMarkup(
      <Dialog id="d" title="Edit it" footerLeft={<button>Delete</button>} footer={<button>Save</button>}>body</Dialog>,
    );
    expect(withLeft).not.toContain("to cancel");
    expect(slot(withLeft, "footer")).toMatch(/>Delete<.*>Save</s);
  });

  it("has no footer without footer or footerLeft", () => {
    expect(renderMarkup(<Dialog id="d" title="t">b</Dialog>)).not.toContain('data-slot="footer"');
  });

  it("shows an icon before the title and header actions after it", () => {
    const withIcon = renderMarkup(<Dialog id="d" title="Merge" icon={<svg data-icon />} headerActions={<button>More</button>}>b</Dialog>);
    const header = slot(withIcon, "header")!;
    expect(header.indexOf("data-icon")).toBeLessThan(header.indexOf(">Merge<"));
    expect(header.indexOf(">More<")).toBeGreaterThan(header.indexOf(">Merge<"));
  });

  it("shows a close button only with onClose", () => {
    expect(html).not.toContain('aria-label="Close"');
    const closable = renderMarkup(<Dialog id="d" title="t" onClose={() => {}}>b</Dialog>);
    expect(slot(closable, "header")).toContain('aria-label="Close"');
  });

  it.each([
    ["sm", "max-w-96"],
    ["md", "max-w-112"],
    ["lg", "max-w-128"],
    ["xl", "max-w-168"],
    ["2xl", "max-w-215"],
  ] as const)("sizes %s to %s", (size, cls) => {
    expect(renderMarkup(<Dialog id="d" title="t" size={size}>b</Dialog>)).toMatch(new RegExp(`role="dialog"[^>]*class="[^"]*\\b${cls}\\b`));
  });

  it("defaults to md", () => {
    expect(html).toMatch(/role="dialog"[^>]*class="[^"]*\bmax-w-112\b/);
  });
});
