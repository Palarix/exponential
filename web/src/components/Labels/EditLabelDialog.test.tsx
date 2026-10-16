import { describe, expect, it } from "vitest";
import { renderMarkup } from "../../render-markup";
import { EditLabelPanel } from "./EditLabelDialog";
import { LABEL_PRESET_COLORS } from "../../constants";

const noop = () => {};
const issues = [
  { id: "upe-4f2a", title: "Retry failed courier webhooks", status: "DOING" },
  { id: "upe-91c0", title: "Menu sync from POS", status: "PLANNED" },
];

function panel(props: { name?: string; color?: string; sample?: typeof issues }) {
  return renderMarkup(
    <EditLabelPanel
      id="e"
      name={props.name ?? "ux"}
      color={props.color ?? LABEL_PRESET_COLORS[0]}
      sampleIssues={props.sample ?? issues}
      onNameChange={noop}
      onColorChange={noop}
      onSave={noop}
      onCancel={noop}
      onDelete={noop}
    />,
  );
}

const saveDisabled = /<button[^>]*disabled=""[^>]*>Save</;

describe("EditLabelPanel", () => {
  it("is titled Edit label", () => {
    expect(panel({})).toMatch(/id="e-title"[^>]*>Edit label</);
  });

  it("has an editable name, with Delete label in place of the esc hint", () => {
    const html = panel({});
    expect(html).not.toMatch(/<input[^>]*readOnly/i);
    expect(html).toContain("Delete label");
    expect(html).not.toContain("to cancel");
  });

  it("disables Save for an uncolored label or a blank name", () => {
    expect(panel({ color: "" })).toMatch(saveDisabled);
    expect(panel({ name: "  " })).toMatch(saveDisabled);
    expect(panel({})).not.toMatch(saveDisabled);
  });

  it("previews the label on the given issues", () => {
    const html = panel({});
    expect(html).toContain("upe-4f2a");
    expect(html).toContain("Retry failed courier webhooks");
    expect(html).toContain("Menu sync from POS");
  });

  it("previews on a placeholder row when no issue has the label", () => {
    expect(panel({ sample: [] })).toContain("Issues with this label look like this");
  });

  it("puts the name and color fields before the preview", () => {
    const html = panel({});
    expect(html.indexOf('aria-label="Label name"')).toBeLessThan(html.indexOf("upe-4f2a"));
    expect(html.indexOf('aria-label="Color #')).toBeLessThan(html.indexOf("upe-4f2a"));
  });

  it("captions the preview", () => {
    expect(panel({})).toMatch(/>Preview<.*upe-4f2a/s);
  });

  it("offers one swatch per preset color", () => {
    expect(panel({}).match(/aria-label="Color #/g)).toHaveLength(LABEL_PRESET_COLORS.length);
  });
});
