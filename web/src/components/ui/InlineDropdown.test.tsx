import { describe, expect, it } from "vitest";
import { renderMarkup } from "../../render-markup";
import InlineDropdown from "./InlineDropdown";
import { fieldTriggerClass } from "./field-trigger";

const options = [{ value: "a", label: "Alpha" }];
const trigger = (html: string) => html.match(/<button[^>]*>/)?.[0] ?? "";

describe("InlineDropdown", () => {
  it("uses the shared field style, so it matches the other field triggers", () => {
    const html = trigger(renderMarkup(<InlineDropdown placeholder="Pick" options={options} value="a" onChange={() => {}} />));
    for (const cls of fieldTriggerClass(false).split(" ")) expect(html).toContain(cls);
  });

  it("marks a required dropdown without a value in red", () => {
    const html = trigger(renderMarkup(<InlineDropdown placeholder="Pick" options={options} value="" onChange={() => {}} required />));
    expect(html).toContain("border-[var(--color-error)]/40");
  });

  it("keeps borderless dropdowns plain", () => {
    const html = trigger(renderMarkup(<InlineDropdown placeholder="Pick" options={options} value="a" onChange={() => {}} borderless />));
    expect(html).not.toContain("bg-[var(--color-field-rest)]");
  });
});
