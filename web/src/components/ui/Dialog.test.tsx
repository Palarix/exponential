import { describe, expect, it } from "vitest";
import { renderMarkup } from "../../render-markup";
import { DialogPanel } from "./Dialog";

describe("DialogPanel", () => {
  const html = renderMarkup(
    <DialogPanel id="d" title="Edit it" footer={<button>Save</button>}>
      <div>the body</div>
    </DialogPanel>,
  );

  it("is a dialog labelled by its title", () => {
    expect(html).toMatch(/role="dialog"[^>]*aria-labelledby="d-title"/);
    expect(html).toMatch(/id="d-title"[^>]*>Edit it</);
  });

  it("puts the body before a footer bar with the esc hint and the buttons", () => {
    expect(html.indexOf("the body")).toBeGreaterThan(html.indexOf("Edit it"));
    const footer = html.match(/<div class="[^"]*border-t[^"]*bg-\[var\(--color-surface-2\)\][^"]*">.*$/s)?.[0];
    expect(footer).toMatch(/<kbd[^>]*>esc<\/kbd>.*to cancel.*>Save</s);
  });

  it("replaces the esc hint with footerLeft", () => {
    const withLeft = renderMarkup(
      <DialogPanel id="d" title="Edit it" footerLeft={<button>Delete</button>} footer={<button>Save</button>}>body</DialogPanel>,
    );
    expect(withLeft).not.toContain("to cancel");
    expect(withLeft).toMatch(/>Delete<.*>Save</s);
  });

  it("takes an alertdialog role", () => {
    expect(renderMarkup(<DialogPanel id="d" role="alertdialog" title="t" footer={null}>b</DialogPanel>)).toContain('role="alertdialog"');
  });
});
