// @vitest-environment jsdom
import { act } from "react";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { KeyboardNavProvider } from "../../keyboard";
import { mountDom, pressKey, setupDom } from "../../test-dom";
import type { ChatMeta } from "../../api/chat";
import { ChatRow } from "../Agent/IssueChats";
import { Menu, MenuDivider, MenuItem } from "./Menu";
import { SubMenu } from "./SubMenu";

/** How a menu behaves for the mouse and keyboard: rows, submenus, and custom rows. */

beforeAll(setupDom);

let cleanup: (() => void) | undefined;
afterEach(() => {
  cleanup?.();
  cleanup = undefined;
});

function mount(el: React.ReactElement) {
  const m = mountDom(<KeyboardNavProvider>{el}</KeyboardNavProvider>);
  cleanup = m.cleanup;
  return m;
}

const row = (label: string) => {
  const found = [...document.querySelectorAll<HTMLElement>("[role^=menuitem]")].find((r) => r.textContent?.startsWith(label));
  if (!found) throw new Error(`no row "${label}"`);
  return found;
};
const focused = () => document.querySelector<HTMLElement>('[role^=menuitem][tabindex="0"]')?.textContent ?? null;
const click = (el: HTMLElement) => act(() => { el.click(); });
// jsdom has no layout: every rect is empty at 0,0. Hover far from it, so the
// safe triangle (SubMenu.dom.test.tsx covers it) never holds a submenu open.
const hover = (el: HTMLElement) => act(() => {
  el.dispatchEvent(new MouseEvent("mouseover", { bubbles: true, clientX: 5000, clientY: 5000 }));
});
const key = (k: string) => pressKey(document.body, k);
const flyouts = () => document.querySelectorAll("[data-submenu-flyout]").length;

describe("Menu rows", () => {
  function mountFlat() {
    const onClose = vi.fn();
    const one = vi.fn();
    const two = vi.fn();
    const keep = vi.fn();
    mount(
      <Menu aria-label="Flat" onClose={onClose} autoFocus={false}>
        <MenuItem label="One" shortcut="o" onClick={one} />
        <MenuDivider />
        <MenuItem label="Disabled" disabled onClick={vi.fn()} />
        <MenuItem label="Two" shortcut="t" onClick={two} />
        <MenuItem label="Keep" keepOpen onClick={keep} />
      </Menu>,
    );
    return { onClose, one, two, keep };
  }

  it("runs a row's action and closes on click", () => {
    const { onClose, two } = mountFlat();
    click(row("Two"));
    expect(two).toHaveBeenCalledTimes(1);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("stays open for a keepOpen row", () => {
    const { onClose, keep } = mountFlat();
    click(row("Keep"));
    expect(keep).toHaveBeenCalledTimes(1);
    expect(onClose).not.toHaveBeenCalled();
  });

  it("ignores a disabled row", () => {
    const { onClose } = mountFlat();
    click(row("Disabled"));
    expect(onClose).not.toHaveBeenCalled();
  });

  it("focuses the row under the mouse", () => {
    mountFlat();
    hover(row("Two"));
    expect(focused()).toContain("Two");
  });

  it("moves with the arrows past dividers and disabled rows, and runs the row on Enter", () => {
    const { onClose, two } = mountFlat();
    key("ArrowDown");
    expect(focused()).toContain("One");
    key("ArrowDown");
    expect(focused()).toContain("Two");
    key("End");
    expect(focused()).toContain("Keep");
    key("Home");
    expect(focused()).toContain("One");
    key("ArrowUp");
    expect(focused()).toContain("Keep");
    key("ArrowUp");
    key("Enter");
    expect(two).toHaveBeenCalledTimes(1);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("runs the row whose shortcut is typed", () => {
    const { one, onClose } = mountFlat();
    key("o");
    expect(one).toHaveBeenCalledTimes(1);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("closes on Escape", () => {
    const { onClose } = mountFlat();
    key("Escape");
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});

describe("Menu submenus", () => {
  function mountNested() {
    const onClose = vi.fn();
    const inner = vi.fn();
    const plain = vi.fn();
    mount(
      <Menu aria-label="Root" onClose={onClose} autoFocus={false}>
        <SubMenu label="More">
          <MenuItem label="Inner" onClick={inner} />
          <MenuItem label="Inner two" onClick={vi.fn()} />
        </SubMenu>
        <MenuItem label="Plain" onClick={plain} />
      </Menu>,
    );
    return { onClose, inner, plain };
  }

  it("opens a submenu on hover and closes it on a plain row", () => {
    mountNested();
    hover(row("More"));
    expect(flyouts()).toBe(1);
    expect(focused()).toContain("More");
    hover(row("Plain"));
    expect(flyouts()).toBe(0);
  });

  it("opens a submenu on Enter and ArrowRight, and runs its rows on their own menu", () => {
    const { onClose, inner } = mountNested();
    key("ArrowDown");
    key("ArrowRight");
    expect(flyouts()).toBe(1);
    click(row("Inner"));
    expect(inner).toHaveBeenCalledTimes(1);
    // The nested menu closes itself: its onClose is the parent's closeSubMenu.
    expect(flyouts()).toBe(0);
    expect(onClose).not.toHaveBeenCalled();
    key("Enter");
    expect(flyouts()).toBe(1);
  });

  it("focuses a row inside the flyout on hover, not the parent's", () => {
    mountNested();
    hover(row("More"));
    hover(row("Inner two"));
    expect(row("Inner two").getAttribute("tabindex")).toBe("0");
    expect(row("More").getAttribute("tabindex")).toBe("0");
  });

  it("closes the submenu, not the menu, on Escape", () => {
    const { onClose } = mountNested();
    hover(row("More"));
    key("Escape");
    expect(flyouts()).toBe(0);
    expect(onClose).not.toHaveBeenCalled();
  });
});

describe("a custom row", () => {
  const chat: ChatMeta = {
    id: "c1",
    title: "Chat one",
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    agent: "claude",
    state: "idle",
  };

  it("takes hover focus and runs its action through the menu", () => {
    const onClose = vi.fn();
    const open = vi.fn();
    mount(
      <Menu aria-label="Chats" onClose={onClose} autoFocus={false}>
        <MenuItem label="Before" onClick={vi.fn()} />
        <ChatRow chat={chat} issueId="xpo-1" hasActions={false} actionsOpen={false} onToggleActions={vi.fn()} onClick={open} />
      </Menu>,
    );
    hover(row("Chat one"));
    expect(focused()).toContain("Chat one");
    click(row("Chat one"));
    expect(open).toHaveBeenCalledTimes(1);
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
