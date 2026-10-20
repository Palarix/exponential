import { describe, expect, it } from "vitest";
import type { Release } from "../../api/types";
import { renderMarkup as render, tagsWithRole as byRole } from "../../render-markup";
import ReleasePicker from "./ReleasePicker";
import { pickerOptions } from "./release-assign-utils";

const noop = () => {};
const rel = (number: number, o: Partial<Release> = {}): Release => ({
  id: `rel-${number}`, number, name: o.label ?? `#${number}`, state: "OPEN", counts: {}, total: 0, done: 0, ...o,
});
const options = pickerOptions({
  current: rel(1, { label: "1.1", goal: "Courier app", slot: "current" }),
  next: rel(2, { id: undefined, placeholder: true, locked: false, slot: "next" }),
  upcoming: [rel(3, { label: "Team Mode", slot: "upcoming" })],
  past: [rel(9, { label: "1.0", state: "RELEASED" })],
});

describe("ReleasePicker", () => {
  it("offers the open releases as radios, never past ones", () => {
    const html = render(<ReleasePicker options={options} current="rel-1" onSelect={noop} />);
    const radios = byRole(html, "menuitemradio");
    expect(radios).toHaveLength(3);
    expect(radios.filter((r) => r.includes('aria-checked="true"'))).toHaveLength(1);
    expect(html).toContain("#2");
    expect(html).toContain("Team Mode");
    expect(html).not.toContain("1.0");
  });

  it("shows each release as the Releases view does: name – goal and the slot badge", () => {
    const html = render(<ReleasePicker options={options} onSelect={noop} />);
    expect(html).toContain(" – Courier app");
    expect(html).toMatch(/rounded-full[^>]*>Current</);
    expect(html).toMatch(/rounded-full[^>]*>Next</);
    expect(html).not.toContain("· Current");
  });

  it("names the menu like the other property pickers", () => {
    expect(render(<ReleasePicker options={options} onSelect={noop} />)).toContain('role="menu" aria-label="Change release"');
  });

  it("names each item for assistive tech", () => {
    const html = render(<ReleasePicker options={options} onSelect={noop} />);
    expect(byRole(html, "menuitemradio")[0]).toContain('aria-label="1.1, Current"');
  });

  it("offers No release only while the issue has one", () => {
    const withRelease = render(<ReleasePicker options={options} current="rel-1" onSelect={noop} />);
    expect(byRole(withRelease, "menuitem")).toHaveLength(1);
    expect(withRelease).toContain("No release");
    expect(byRole(render(<ReleasePicker options={options} onSelect={noop} />), "menuitem")).toHaveLength(0);
  });

  it("shows the issue's released release checked and disabled, with a Released pill", () => {
    const board = {
      current: rel(1, { label: "1.1", slot: "current" as const }),
      next: rel(2, { slot: "next" as const }),
      upcoming: [],
      past: [rel(9, { label: "1.0", state: "RELEASED" as const, slot: "past" as const })],
    };
    const html = render(<ReleasePicker options={pickerOptions(board, "rel-9")} current="rel-9" onSelect={noop} />);
    const [first] = byRole(html, "menuitemradio");
    expect(first).toContain('aria-label="1.0, Released"');
    expect(first).toContain('aria-checked="true"');
    expect(first).toContain('aria-disabled="true"');
    expect(html).toMatch(/rounded-full[^>]*>Released</);
  });

  it("disables a locked Next", () => {
    const locked = options.map((o) => (o.slotLabel === "Next" ? { ...o, disabled: true } : o));
    const html = render(<ReleasePicker options={locked} onSelect={noop} />);
    expect(byRole(html, "menuitemradio").filter((r) => r.includes('aria-disabled="true"'))).toHaveLength(1);
    expect(html).toContain("Save Current first");
  });
});
