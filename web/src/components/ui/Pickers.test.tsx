import { describe, expect, it } from "vitest";
import { MenuItem, Menu } from "./Menu";
import StatusPicker from "./StatusPicker";
import PriorityPicker from "./PriorityPicker";
import EstimatePicker from "./EstimatePicker";
import CyclePicker from "./CyclePicker";
import AssigneePicker from "./AssigneePicker";
import LabelPicker from "./LabelPicker";
import { STATUS_OPTIONS, PRIORITY_OPTIONS, ESTIMATE_OPTIONS } from "../../constants";
import type { Cycle } from "../../api/client";
import { renderMarkup as render, tagsWithRole as byRole } from "../../render-markup";

const noop = () => {};

describe("MenuItem roles", () => {
  it("renders a plain menuitem without a checked state", () => {
    const html = render(<Menu><MenuItem label="Open" /></Menu>);
    expect(byRole(html, "menuitem")).toHaveLength(1);
    expect(html).not.toContain("aria-checked");
  });

  it("renders menuitemcheckbox for checkable items", () => {
    const html = render(<Menu><MenuItem label="Show" checked /></Menu>);
    const [item] = byRole(html, "menuitemcheckbox");
    expect(item).toContain('aria-checked="true"');
  });

  it("renders menuitemradio for radio items", () => {
    const html = render(<Menu><MenuItem label="High" radio checked={false} /></Menu>);
    const [item] = byRole(html, "menuitemradio");
    expect(item).toContain('aria-checked="false"');
    expect(byRole(html, "menuitemcheckbox")).toHaveLength(0);
  });

  it("shows the checked item in primary text, like a focused one", () => {
    const html = render(
      <Menu autoFocus={false}>
        <MenuItem label="Low" radio checked={false} />
        <MenuItem label="High" radio checked />
      </Menu>,
    );
    const [low, high] = byRole(html, "menuitemradio");
    expect(low).toContain("text-[var(--color-text-secondary)]");
    expect(high).toContain("text-[var(--color-text-primary)]");
  });

  it("passes aria-label to the menu container", () => {
    const html = render(<Menu aria-label="Change status"><MenuItem label="x" /></Menu>);
    expect(byRole(html, "menu")[0]).toContain('aria-label="Change status"');
  });
});

function expectRadioMenu(html: string, label: string, optionCount: number) {
  const [menu] = byRole(html, "menu");
  expect(menu).toContain(`aria-label="${label}"`);
  const radios = byRole(html, "menuitemradio");
  expect(radios).toHaveLength(optionCount);
  expect(radios.filter((r) => r.includes('aria-checked="true"'))).toHaveLength(1);
  expect(byRole(html, "menuitemcheckbox")).toHaveLength(0);
}

describe("single-choice pickers", () => {
  it("StatusPicker exposes radio options with the current status checked", () => {
    const html = render(<StatusPicker current={STATUS_OPTIONS[1].value} onSelect={noop} />);
    expectRadioMenu(html, "Change status", STATUS_OPTIONS.length);
  });

  it("PriorityPicker exposes radio options with the current priority checked", () => {
    const html = render(<PriorityPicker current={PRIORITY_OPTIONS[2].value} onSelect={noop} />);
    expectRadioMenu(html, "Set priority", PRIORITY_OPTIONS.length);
  });

  it("EstimatePicker exposes radio options with the current estimate checked", () => {
    const html = render(<EstimatePicker current={ESTIMATE_OPTIONS[1]} onSelect={noop} />);
    expectRadioMenu(html, "Change estimate", ESTIMATE_OPTIONS.length);
  });

  it("CyclePicker exposes radio options for cycles", () => {
    const cycle = (id: string, number: number): Cycle => ({
      id, number, start: id, end: id, status: "upcoming", done: 0, total: 0,
    });
    const cycles = [cycle("2026-10-01", 1), cycle("2026-10-15", 2)];
    const html = render(<CyclePicker cycles={cycles} current="2026-10-15" onSelect={noop} />);
    const radios = byRole(html, "menuitemradio");
    expect(radios).toHaveLength(2);
    expect(radios.filter((r) => r.includes('aria-checked="true"'))).toHaveLength(1);
    // "No cycle" is an action, not a choice with a checked state.
    expect(byRole(html, "menuitem")).toHaveLength(1);
  });
});

describe("AssigneePicker", () => {
  const people = ["Alice <alice@x.io>", "Bob <bob@x.io>"];

  it("exposes radio options with the current assignee checked", () => {
    const html = render(<AssigneePicker people={people} current="Bob <bob@x.io>" onSelect={noop} />);
    // "No assignee" plus one option per person.
    expectRadioMenu(html, "Change assignee", people.length + 1);
  });

  it("offers No assignee as a choice, checked when nobody is assigned", () => {
    const assigned = render(<AssigneePicker people={people} current="Bob <bob@x.io>" onSelect={noop} />);
    const [noneWhenAssigned] = byRole(assigned, "menuitemradio");
    expect(noneWhenAssigned).toContain('aria-checked="false"');
    expect(assigned).toContain("No assignee");
    expect(assigned).not.toContain("Remove assignee");

    const unassigned = render(<AssigneePicker people={people} onSelect={noop} />);
    const [none, ...rest] = byRole(unassigned, "menuitemradio");
    expect(none).toContain('aria-checked="true"');
    expect(rest.every((r) => r.includes('aria-checked="false"'))).toBe(true);
  });

  it("shows a dashed placeholder avatar for No assignee", () => {
    const html = render(<AssigneePicker people={people} onSelect={noop} />);
    const none = html.slice(html.indexOf('role="menuitemradio"'), html.indexOf("No assignee"));
    expect(none).toContain("border-dashed");
  });

  it("shows an empty state when nobody is known", () => {
    const html = render(<AssigneePicker people={[]} onSelect={noop} />);
    expect(html).toContain("No matching people");
  });
});

describe("LabelPicker", () => {
  it("stays multi-select with menuitemcheckbox options", () => {
    const html = render(<LabelPicker allLabels={["bug", "ux"]} selected={["ux"]} onToggle={noop} />);
    expect(byRole(html, "menuitemradio")).toHaveLength(0);
    const boxes = byRole(html, "menuitemcheckbox");
    expect(boxes.length).toBeGreaterThanOrEqual(2);
    expect(boxes.filter((b) => b.includes('aria-checked="true"'))).toHaveLength(1);
  });
});
