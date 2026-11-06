// @vitest-environment jsdom
import { act } from "react";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { KeyboardNavProvider } from "../../keyboard";
import { mountDom, setupDom } from "../../test-dom";
import { Menu, MenuItem } from "./Menu";
import { SubMenu } from "./SubMenu";

/**
 * jsdom has no layout, so these tests supply it: the top menu is ROOT_WIDTH wide
 * at `rootLeft`, and each flyout is FLYOUT_WIDTH wide at the position SubMenu gave
 * it. Every element measures as its whole menu, so a row's edges are its menu's.
 * The positions and the event order are this file's model of the browser.
 */
const ROOT_WIDTH = 200;
const FLYOUT_WIDTH = 256;
const HEIGHT = 200;
let rootLeft = 600;

const rect = (left: number, top: number, width: number, height: number) =>
  ({ x: left, y: top, left, top, width, height, right: left + width, bottom: top + height, toJSON: () => ({}) }) as DOMRect;

beforeAll(setupDom);

let cleanup: (() => void) | undefined;
const originalWidth = window.innerWidth;

beforeEach(() => {
  vi.useFakeTimers({ toFake: ["setTimeout", "clearTimeout"] });
  vi.spyOn(Element.prototype, "getBoundingClientRect").mockImplementation(function (this: Element) {
    const flyout = this.closest<HTMLElement>("[data-submenu-flyout]");
    if (!flyout) return rect(rootLeft, 100, ROOT_WIDTH, HEIGHT);
    return rect(parseFloat(flyout.style.left) || 0, parseFloat(flyout.style.top) || 100, FLYOUT_WIDTH, HEIGHT);
  });
});

afterEach(() => {
  cleanup?.();
  cleanup = undefined;
  vi.useRealTimers();
  vi.restoreAllMocks();
  window.innerWidth = originalWidth;
});

function mount({ left = 600, viewport = 1024 } = {}) {
  rootLeft = left;
  window.innerWidth = viewport;
  const m = mountDom(
    <KeyboardNavProvider>
      <Menu aria-label="Root">
        <SubMenu label="First">
          <MenuItem label="first-row" />
          <SubMenu label="Nested one"><MenuItem label="nested-one-row" /></SubMenu>
          <SubMenu label="Nested two"><MenuItem label="nested-two-row" /></SubMenu>
          <MenuItem label="first-plain" />
        </SubMenu>
        <SubMenu label="Second"><MenuItem label="second-row" /></SubMenu>
        <MenuItem label="Plain" />
      </Menu>
    </KeyboardNavProvider>,
  );
  cleanup = m.cleanup;
}

const flyouts = () => [...document.querySelectorAll<HTMLElement>("[data-submenu-flyout]")];
const row = (label: string) => {
  const found = [...document.querySelectorAll<HTMLElement>("[role^=menuitem]")].find((r) => r.textContent?.startsWith(label));
  if (!found) throw new Error(`no row "${label}"`);
  return found;
};
/** The row labels in flyout `level`, to tell which submenu it shows. */
const shows = (level: number) => flyouts()[level]?.textContent ?? "";

// Event times are explicit, so a test can pass a pause without waiting for it.
let clock = 0;
const fire = (el: HTMLElement, type: string, x: number, y: number) => {
  const event = new MouseEvent(type, { bubbles: true, clientX: x, clientY: y });
  Object.defineProperty(event, "timeStamp", { value: clock });
  act(() => { el.dispatchEvent(event); });
};
/** One step of the mouse, inside the row it was already on. */
const step = (el: HTMLElement, x: number, y: number) => {
  clock += 16;
  fire(el, "mousemove", x, y);
};
/** The step that enters a row. The browser fires mouseover before that step's mousemove. */
const enter = (el: HTMLElement, x: number, y: number) => {
  clock += 16;
  fire(el, "mouseover", x, y);
  fire(el, "mousemove", x, y);
};
const pause = () => { clock += 1000; };
const wait = (ms: number) => act(() => { vi.advanceTimersByTime(ms); });

describe("a mouse move near an open flyout", () => {
  // At left 600 in a 1024 window, the flyout opens left.
  const openFirst = () => {
    mount();
    enter(row("First"), 700, 120);
    expect(shows(0)).toContain("first-row");
    pause();
  };

  it("waits on a submenu row when the move heads for the flyout", () => {
    openFirst();
    step(row("First"), 690, 127);
    enter(row("Second"), 680, 134);
    expect(shows(0)).toContain("first-row");
  });

  it("switches after the wait if the mouse stays on that row", () => {
    openFirst();
    step(row("First"), 690, 127);
    enter(row("Second"), 680, 134);
    wait(300);
    expect(shows(0)).toContain("second-row");
  });

  it("keeps the flyout when the mouse reaches it before the wait ends", () => {
    openFirst();
    step(row("First"), 690, 127);
    enter(row("Second"), 680, 134);
    enter(flyouts()[0], 590, 150);
    wait(1000);
    expect(shows(0)).toContain("first-row");
  });

  it("switches at once on a move along the rows", () => {
    openFirst();
    step(row("First"), 700, 130);
    enter(row("Second"), 700, 140);
    expect(shows(0)).toContain("second-row");
  });

  it("switches at once on a move away from the flyout", () => {
    openFirst();
    step(row("First"), 710, 127);
    enter(row("Second"), 720, 134);
    expect(shows(0)).toContain("second-row");
  });

  it("ignores positions from before a pause", () => {
    mount();
    // The mouse arrives from the left, moving away from the left-side flyout.
    for (const x of [600, 620, 640, 660]) step(document.querySelector<HTMLElement>('[aria-label="Root"]')!, x, 120);
    enter(row("First"), 700, 120);
    pause();
    // A fresh move, back toward the flyout.
    step(row("First"), 690, 127);
    enter(row("Second"), 680, 134);
    expect(shows(0)).toContain("first-row");
  });

  it("reads the direction against its own flyout when two are open", () => {
    // At left 100, both flyouts open right. The first flyout in the document is the
    // second level itself, which reads as the left side of its own rows.
    mount({ left: 100 });
    enter(row("First"), 200, 120);
    enter(row("Nested one"), 400, 140);
    expect(shows(1)).toContain("nested-one-row");
    pause();
    step(row("Nested one"), 410, 147);
    enter(row("Nested two"), 420, 154);
    expect(shows(1)).toContain("nested-one-row");
  });
});

describe("the side a flyout opens on", () => {
  const openNested = () => {
    enter(row("First"), rootLeft + 100, 120);
    enter(row("Nested one"), parseFloat(flyouts()[0].style.left) + 100, 140);
  };

  it("keeps opening left after the second level flipped left", () => {
    // The third level would fit on the right here, over the top menu.
    mount({ left: 600, viewport: 1024 });
    openNested();
    expect(flyouts().map((f) => f.dataset.submenuSide)).toEqual(["left", "left"]);
    expect(flyouts().map((f) => f.style.left)).toEqual(["340px", "80px"]);
  });

  it("opens right when the top menu has room there", () => {
    mount({ left: 100, viewport: 1024 });
    openNested();
    expect(flyouts().map((f) => f.dataset.submenuSide)).toEqual(["right", "right"]);
    expect(flyouts().map((f) => f.style.left)).toEqual(["304px", "564px"]);
  });

  it("turns back right when the left has no room for the third level", () => {
    mount({ left: 270, viewport: 720 });
    openNested();
    expect(flyouts().map((f) => f.dataset.submenuSide)).toEqual(["left", "right"]);
    expect(flyouts().map((f) => f.style.left)).toEqual(["10px", "270px"]);
  });
});
