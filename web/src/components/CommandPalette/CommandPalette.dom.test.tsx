// @vitest-environment jsdom
import { act, type ComponentProps } from "react";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import type { Issue } from "../../api/client";
import { mountDom, pressKey, setupDom } from "../../test-dom";
import CommandPalette from "./CommandPalette";

beforeAll(setupDom);

const issue = (id: string, title: string): Issue => ({ id, title, status: "BACKLOG", labels: [] }) as unknown as Issue;
const issues = [issue("xpo-aaa111", "Alpha task"), issue("xpo-bbb222", "Beta task")];

type Props = ComponentProps<typeof CommandPalette>;
let props: Props;
let m: ReturnType<typeof mountDom> | undefined;

beforeEach(() => {
  vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] });
  props = {
    isOpen: false,
    onClose: vi.fn(),
    issues,
    onIssueSelect: vi.fn(),
    onViewChange: vi.fn(),
    onNewIssue: vi.fn(),
    cyclesEnabled: false,
  };
});
afterEach(() => {
  m?.cleanup();
  m = undefined;
  vi.useRealTimers();
});

function render(over: Partial<Props> = {}) {
  props = { ...props, ...over };
  if (m) m.rerender(<CommandPalette {...props} />);
  else m = mountDom(<CommandPalette {...props} />);
  act(() => { vi.runAllTimers(); });
}

const input = () => document.querySelector<HTMLInputElement>("input")!;
const rows = () => [...document.querySelectorAll<HTMLButtonElement>("button[data-index]")];
const selected = () => rows().find((b) => b.className.includes("bg-[var(--color-hover-surface-3)]"))?.textContent ?? null;
const selectedIndex = () => rows().findIndex((b) => b.className.includes("bg-[var(--color-hover-surface-3)]"));
const type = (value: string) => act(() => {
  const set = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!;
  set.call(input(), value);
  input().dispatchEvent(new Event("input", { bubbles: true }));
});
const key = (k: string) => pressKey(input(), k);

describe("CommandPalette", () => {
  it("renders nothing while closed", () => {
    render();
    expect(input()).toBeNull();
  });

  it("opens empty, with the first row selected and the input focused", () => {
    render({ isOpen: true });
    expect(input().value).toBe("");
    expect(selectedIndex()).toBe(0);
    expect(selected()).toContain("Create new issue");
    expect(document.activeElement).toBe(input());
  });

  it("moves the selection with the arrows, clamped at both ends", () => {
    render({ isOpen: true });
    key("ArrowUp");
    expect(selectedIndex()).toBe(0);
    key("ArrowDown");
    key("ArrowDown");
    expect(selectedIndex()).toBe(2);
    for (let i = 0; i < 50; i++) key("ArrowDown");
    expect(selectedIndex()).toBe(rows().length - 1);
  });

  it("selects the row under the mouse", () => {
    render({ isOpen: true });
    act(() => { rows()[3].dispatchEvent(new MouseEvent("mouseover", { bubbles: true })); });
    expect(selectedIndex()).toBe(3);
  });

  it("selects the first row again when the query changes", () => {
    render({ isOpen: true });
    key("ArrowDown");
    key("ArrowDown");
    type("task");
    expect(input().value).toBe("task");
    expect(selectedIndex()).toBe(0);
    expect(rows().map((r) => r.textContent).join("|")).toContain("Alpha task");
  });

  it("runs the selected row on Enter", () => {
    render({ isOpen: true });
    type("beta");
    key("Enter");
    expect(props.onClose).toHaveBeenCalled();
    expect(props.onIssueSelect).toHaveBeenCalledWith(issues[1]);
  });

  it("closes on Escape", () => {
    render({ isOpen: true });
    key("Escape");
    expect(props.onClose).toHaveBeenCalledTimes(1);
  });

  it("opens fresh after a close: empty query, first row selected", () => {
    render({ isOpen: true });
    type("beta");
    key("ArrowDown");
    render({ isOpen: false });
    render({ isOpen: true });
    expect(input().value).toBe("");
    expect(selectedIndex()).toBe(0);
    expect(document.activeElement).toBe(input());
  });
});
