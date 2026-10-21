import { describe, expect, it } from "vitest";
import { renderMarkup } from "../../render-markup";
import { ConfirmDialogPanel } from "./ConfirmDialog";

describe("ConfirmDialog", () => {
  const html = renderMarkup(
    <ConfirmDialogPanel id="d" title="Delete it?" description="This can't be undone." body={<div>the card</div>}>
      <button>Cancel</button>
      <button>Delete</button>
    </ConfirmDialogPanel>,
  );

  it("is an alertdialog labelled by its title and described by its text", () => {
    expect(html).toMatch(/role="alertdialog"[^>]*aria-labelledby="d-title"[^>]*aria-describedby="d-desc"/);
    expect(html).toMatch(/id="d-title"[^>]*>Delete it\?</);
    expect(html).toMatch(/id="d-desc"[^>]*>This can&#x27;t be undone\.</);
  });

  it("puts the body between the title and the description", () => {
    expect(html.indexOf("the card")).toBeGreaterThan(html.indexOf("Delete it?"));
    expect(html.indexOf("the card")).toBeLessThan(html.indexOf("This can&#x27;t be undone."));
  });

  it("puts the buttons in the footer after an esc hint", () => {
    const footer = html.match(/<div[^>]*data-slot="footer"[^>]*>.*$/s)?.[0];
    expect(footer).toBeDefined();
    expect(footer).toMatch(/<kbd[^>]*>esc<\/kbd>.*to cancel.*>Cancel<.*>Delete</s);
  });
});
