// @vitest-environment jsdom
import { act, createRef, useState } from "react";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { Issue, Milestone, MilestonesResponse } from "../../api/types";
import { queryKeys } from "../../api/query-keys";
import { KeyboardNavProvider } from "../../keyboard";
import { mountDom, pressKey, setupDom } from "../../test-dom";
import FilterMenu from "./FilterMenu";
import { EMPTY_FILTERS, type BacklogFilters } from "./filters";

beforeAll(setupDom);

const milestone = (number: number, name: string, slot: string): Milestone =>
  ({ id: `ms-${number}`, number, name, slot, state: slot === "past" ? "COMPLETED" : "OPEN", counts: {}, total: 0, done: 0 }) as Milestone;

const BOARD: MilestonesResponse = {
  current: milestone(3, "0.3.0", "current"),
  next: milestone(4, "0.4.0", "next"),
  upcoming: [milestone(5, "0.5.0", "upcoming"), milestone(6, "0.6.0", "upcoming")],
  past: [milestone(2, "0.2.0", "past"), milestone(1, "0.1.0", "past")],
};

const issue = (id: string, milestone_id?: string) => ({ id, title: id, status: "PLANNED", priority: 0, milestone_id }) as Issue;
const ISSUES = [issue("a", "ms-5"), issue("b", "ms-5"), issue("c", "ms-2"), issue("d")];

let cleanup: (() => void) | undefined;
afterEach(() => {
  cleanup?.();
  cleanup = undefined;
});

function mount(board = BOARD) {
  const client = new QueryClient({ defaultOptions: { queries: { staleTime: Infinity } } });
  client.setQueryData(queryKeys.milestones, board);
  const onClose = vi.fn();
  const anchorRef = createRef<HTMLButtonElement>();
  let latest: BacklogFilters = EMPTY_FILTERS;
  function Host() {
    const [filters, setFilters] = useState<BacklogFilters>(EMPTY_FILTERS);
    const onChange = (next: BacklogFilters) => {
      latest = next;
      setFilters(next);
    };
    return (
      <QueryClientProvider client={client}>
        <KeyboardNavProvider>
          <button ref={anchorRef}>Filter</button>
          <FilterMenu issues={ISSUES} filters={filters} onChange={onChange} anchorRef={anchorRef} onClose={onClose} />
        </KeyboardNavProvider>
      </QueryClientProvider>
    );
  }
  const m = mountDom(<Host />);
  cleanup = m.cleanup;
  return { onClose, filters: () => latest };
}

const flyouts = () => [...document.querySelectorAll<HTMLElement>("[data-submenu-flyout]")];
const rowsIn = (el: ParentNode) => [...el.querySelectorAll<HTMLElement>("[role^=menuitem]")];
const name = (row: HTMLElement) => row.getAttribute("aria-label") ?? row.textContent ?? "";
const rowNamed = (el: ParentNode, text: string) => {
  const row = rowsIn(el).find((r) => name(r).startsWith(text));
  if (!row) throw new Error(`no row "${text}" in: ${rowsIn(el).map(name).join(" | ")}`);
  return row;
};
const hover = (el: HTMLElement) => act(() => { el.dispatchEvent(new MouseEvent("mouseover", { bubbles: true })); });
const click = (el: HTMLElement) => act(() => {
  el.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
  el.dispatchEvent(new MouseEvent("mouseup", { bubbles: true }));
  el.dispatchEvent(new MouseEvent("click", { bubbles: true }));
});
const typeSearch = (text: string) => {
  const input = flyouts()[0].querySelector("input") as HTMLInputElement;
  const set = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!;
  act(() => {
    set.call(input, text);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
};
const openMilestone = () => hover(rowNamed(document.body, "Milestone"));
const openGroup = (group: string) => hover(rowNamed(flyouts()[0], group));

describe("FilterMenu's Milestone submenu", () => {
  it("shows No milestone, Current and Next, a separator, then the Upcoming and Past submenus", () => {
    mount();
    openMilestone();
    expect(rowsIn(flyouts()[0]).map(name)).toEqual(["No milestone1", "0.3.0", "0.4.0", "Upcoming", "Past"]);
    // One under the search box, one above the submenus.
    expect(flyouts()[0].querySelectorAll("[role=separator]")).toHaveLength(2);
  });

  it("lists Upcoming nearest first and Past newest first, in a third level", () => {
    mount();
    openMilestone();
    openGroup("Upcoming");
    expect(rowsIn(flyouts()[1]).map(name)).toEqual(["0.5.02", "0.6.00"]);
    openGroup("Past");
    expect(flyouts()).toHaveLength(2);
    expect(rowsIn(flyouts()[1]).map(name)).toEqual(["0.2.01", "0.1.00"]);
  });

  it("hides a group with no milestones, and the separator with both", () => {
    mount({ ...BOARD, upcoming: [] });
    openMilestone();
    expect(rowsIn(flyouts()[0]).map(name)).toEqual(["No milestone1", "0.3.0", "0.4.0", "Past"]);
    cleanup?.();
    cleanup = undefined;

    mount({ ...BOARD, upcoming: [], past: [] });
    openMilestone();
    expect(rowsIn(flyouts()[0]).map(name)).toEqual(["No milestone1", "0.3.0", "0.4.0"]);
    expect(flyouts()[0].querySelectorAll("[role=separator]")).toHaveLength(0);
  });

  it("checks a third-level row on click and stays open, with the count on the submenu row", () => {
    const { onClose, filters } = mount();
    openMilestone();
    openGroup("Upcoming");
    expect(rowNamed(flyouts()[0], "Upcoming").textContent).toBe("Upcoming");

    click(rowNamed(flyouts()[1], "0.5.0"));

    expect(onClose).not.toHaveBeenCalled();
    expect(filters().milestones).toEqual(["#5"]);
    expect(flyouts()).toHaveLength(2);
    expect(rowNamed(flyouts()[1], "0.5.0").getAttribute("aria-checked")).toBe("true");
    expect(rowNamed(flyouts()[0], "Upcoming").textContent).toBe("Upcoming1");
    expect(rowNamed(flyouts()[0], "Past").textContent).toBe("Past");
  });

  it("closes when the click lands outside every flyout", () => {
    const { onClose } = mount();
    openMilestone();
    openGroup("Upcoming");
    click(document.body);
    expect(onClose).toHaveBeenCalled();
  });

  it("gives each flyout the ID of the menu that opened it", () => {
    mount();
    openMilestone();
    openGroup("Upcoming");
    const [second, third] = flyouts().map((f) => f.getAttribute("data-submenu-owner"));
    expect(second).toBeTruthy();
    expect(third).toBeTruthy();
    expect(third).not.toBe(second);
  });

  it("searches every group as one flat list, and finds Current and Next by slot", () => {
    mount();
    openMilestone();
    typeSearch("0.");
    expect(rowsIn(flyouts()[0]).map(name)).toEqual(["0.3.0", "0.4.0", "0.5.02", "0.6.00", "0.2.01", "0.1.00"]);
    // Only the line under the search box.
    expect(flyouts()[0].querySelectorAll("[role=separator]")).toHaveLength(1);
    typeSearch("next");
    expect(rowsIn(flyouts()[0]).map(name)).toEqual(["0.4.0"]);
  });

  it("closes an open third level on search, and does not reopen it when the search clears", () => {
    mount();
    openMilestone();
    openGroup("Upcoming");
    expect(flyouts()).toHaveLength(2);

    typeSearch("0.5");
    expect(flyouts()).toHaveLength(1);
    expect(rowsIn(flyouts()[0]).map(name)).toEqual(["0.5.02"]);

    typeSearch("");
    expect(flyouts()).toHaveLength(1);
    expect(rowsIn(flyouts()[0]).map(name)).toContain("Upcoming");
  });

  it("moves through the levels with the keyboard, Escape closing one level at a time", () => {
    const { onClose } = mount();
    openMilestone();
    for (let i = 0; i < 8 && rowNamed(flyouts()[0], "Upcoming").tabIndex !== 0; i++) pressKey(document.body, "ArrowDown");
    expect(rowNamed(flyouts()[0], "Upcoming").tabIndex).toBe(0);

    pressKey(document.body, "ArrowRight");
    expect(flyouts()).toHaveLength(2);

    pressKey(document.body, "ArrowLeft");
    expect(flyouts()).toHaveLength(1);

    pressKey(document.body, "ArrowRight");
    expect(flyouts()).toHaveLength(2);

    pressKey(document.body, "Escape");
    expect(flyouts()).toHaveLength(1);
    expect(onClose).not.toHaveBeenCalled();

    pressKey(document.body, "Escape");
    expect(flyouts()).toHaveLength(0);
    expect(onClose).not.toHaveBeenCalled();

    pressKey(document.body, "Escape");
    expect(onClose).toHaveBeenCalled();
  });
});
