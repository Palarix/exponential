// @vitest-environment jsdom
import { act, type ReactNode } from "react";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { KeyboardNavProvider } from "../../keyboard";
import { mountDom, pressKey, setupDom } from "../../test-dom";
import { makeIssue } from "../../test-utils";
import type { Milestone, MilestonesResponse } from "../../api/types";
import { queryKeys } from "../../api/query-keys";
import { AppNavContext, ViewStateContext, type AppNav } from "../../app/contexts";
import { createViewStateStore, type ViewStateStore } from "../../app/view-state-utils";
import { ToastContext } from "../ui/ToastContext";
import Backlog from "./Backlog";

/**
 * The backlog's keyboard and its per-tab view settings, on real rows.
 * Mutations are stubbed; the data comes from a seeded query cache.
 */

const mutations = vi.hoisted(() => ({ updateIssue: vi.fn(), updateIssues: vi.fn(), createIssue: vi.fn() }));
vi.mock("../../api/mutations", () => ({
  useUpdateIssue: () => mutations.updateIssue,
  useUpdateIssues: () => ({ mutate: mutations.updateIssues }),
  useCreateIssue: () => ({ mutate: mutations.createIssue }),
}));

beforeAll(setupDom);

const issues = [
  makeIssue({ id: "xpo-p1", title: "Parent", status: "BACKLOG", sort_order: "a0" }),
  makeIssue({ id: "xpo-c1", title: "Child one", status: "BACKLOG", parent_id: "xpo-p1", sort_order: "a1" }),
  makeIssue({ id: "xpo-c2", title: "Child two", status: "BACKLOG", parent_id: "xpo-p1", sort_order: "a2" }),
  makeIssue({ id: "xpo-b2", title: "Loose", status: "BACKLOG", sort_order: "a3" }),
  makeIssue({ id: "xpo-d1", title: "Doing it", status: "DOING", sort_order: "a4" }),
  makeIssue({ id: "xpo-x1", title: "Finished", status: "DONE", sort_order: "a5" }),
];

let nav: AppNav;
let store: ViewStateStore;
let cleanup: (() => void) | undefined;

beforeEach(() => {
  localStorage.clear();
  mutations.updateIssue.mockClear();
});
afterEach(() => {
  cleanup?.();
  cleanup = undefined;
});

function mount() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, enabled: false } } });
  client.setQueryData(queryKeys.issues, issues);
  const ms = (number: number, slot: Milestone["slot"]): Milestone => ({ id: `ms-${number}`, number, name: `#${number}`, state: "OPEN", slot, counts: {}, total: 0, done: 0 }) as Milestone;
  const board: MilestonesResponse = { current: ms(1, "current"), next: ms(2, "next"), upcoming: [], past: [] };
  client.setQueryData(queryKeys.milestones, board);
  client.setQueryData(queryKeys.cycles, []);
  store = createViewStateStore(localStorage);
  nav = {
    view: "backlog",
    navigate: vi.fn(),
    openIssue: vi.fn(),
    closeIssue: vi.fn(),
    newIssue: vi.fn(),
    openDraft: vi.fn(),
    cycleId: null,
    setCycleId: vi.fn(),
    depFocusId: null,
    setDepFocusId: vi.fn(),
    chatId: null,
    openChatView: vi.fn(),
    chatHistory: null,
    openChatHistory: vi.fn(),
    leaveChatView: vi.fn(),
    publishNavOrder: vi.fn(),
    takeIntent: () => undefined,
  };
  const Providers = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>
      <ToastContext.Provider value={{ showToast: () => {} }}>
        <ViewStateContext.Provider value={store}>
          <AppNavContext.Provider value={nav}>
            <KeyboardNavProvider>
              <TooltipPrimitive.Provider>{children}</TooltipPrimitive.Provider>
            </KeyboardNavProvider>
          </AppNavContext.Provider>
        </ViewStateContext.Provider>
      </ToastContext.Provider>
    </QueryClientProvider>
  );
  const m = mountDom(<Providers><Backlog /></Providers>);
  cleanup = m.cleanup;
}

const FOCUS_RING = "ring-[var(--color-accent-primary)]/40";

/** Each visible row: a group header by its status, an issue by its id, `*` when focused, `^` with a parent breadcrumb. */
function rows(): string[] {
  return [...document.querySelectorAll<HTMLElement>("[data-row]")].map((el) => {
    const mark = el.className.includes(FOCUS_RING) ? "*" : "";
    if (el.hasAttribute("data-group-header")) return `${mark}[${el.closest("[data-group-status]")?.getAttribute("data-group-status")}]`;
    const id = el.closest("[data-context-issue]")?.getAttribute("data-context-issue") ?? "?";
    // Flat mode shows a child's parent as a breadcrumb; nested mode indents it instead.
    const parent = issues.find((i) => i.id === issues.find((c) => c.id === id)?.parent_id);
    const crumb = parent && el.textContent?.includes(parent.title) ? "^" : "";
    return `${mark}${id}${crumb}`;
  });
}
const focusedRow = () => rows().find((r) => r.startsWith("*"))?.slice(1) ?? null;
const key = (k: string) => pressKey(document.body, k);
const setTab = (tab: string) => act(() => store.set("backlog.tab", tab));
/** Popovers are portalled to the body as fixed panels. */
const popovers = () => [...document.body.children].filter((el) => (el as HTMLElement).style?.position === "fixed").length;

describe("Backlog keyboard", () => {
  it("lists the groups and nested rows of the All tab", () => {
    mount();
    expect(rows()).toMatchSnapshot();
  });

  it("moves with j/k and the arrows, clamped at both ends", () => {
    mount();
    key("j");
    expect(focusedRow()).toBe(rows()[0].replace("*", ""));
    key("ArrowDown");
    key("j");
    expect(focusedRow()).toBe("xpo-c1");
    key("k");
    key("ArrowUp");
    key("ArrowUp");
    key("ArrowUp");
    expect(focusedRow()).toBe(rows()[0].replace("*", ""));
    for (let i = 0; i < 40; i++) key("j");
    expect(focusedRow()).toBe(rows().at(-1)!.replace("*", ""));
  });

  it("opens the focused issue on Enter", () => {
    mount();
    key("j");
    key("j");
    key("Enter");
    expect(nav.openIssue).toHaveBeenCalledWith("xpo-p1");
  });

  it("collapses and expands a group with Enter and the arrows", () => {
    mount();
    key("j");
    key("ArrowLeft");
    expect(rows()).not.toContain("xpo-p1");
    key("ArrowRight");
    expect(rows()).toContain("xpo-p1");
    key("Enter");
    expect(rows()).not.toContain("xpo-p1");
    key("Enter");
    expect(rows()).toContain("xpo-p1");
  });

  it("collapses and expands a parent with the arrows, and all parents with { and }", () => {
    mount();
    key("j");
    key("j");
    key("ArrowLeft");
    expect(rows()).not.toContain("xpo-c1");
    key("ArrowRight");
    expect(rows()).toContain("xpo-c1");
    key("{");
    expect(rows()).not.toContain("xpo-c1");
    key("}");
    expect(rows()).toContain("xpo-c1");
  });

  it.each(["s", "l", "e", "p", "a", "r"])("opens a row popover on %s, and ignores keys while it is open", (k) => {
    mount();
    key("j");
    key("j");
    expect(popovers()).toBe(0);
    key(k);
    expect(popovers()).toBe(1);
    const focus = focusedRow();
    key("j");
    expect(focusedRow()).toBe(focus);
  });
});

describe("Backlog tabs", () => {
  it("loads each tab's expanded groups, hierarchy mode and empty groups from storage", () => {
    localStorage.setItem("exponential-backlog-expanded-backlog", JSON.stringify([]));
    localStorage.setItem("exponential-backlog-expanded-active", JSON.stringify(["DOING"]));
    localStorage.setItem("exponential-backlog-hierarchy-all", "flat");
    localStorage.setItem("exponential-backlog-empty-groups-active", "true");
    mount();
    const all = rows();
    expect(all).toMatchSnapshot("all");
    setTab("backlog");
    expect(rows()).toEqual(["[BACKLOG]"]);
    setTab("active");
    expect(rows()).toMatchSnapshot("active");
    setTab("done");
    expect(rows()).toMatchSnapshot("done");
    setTab("all");
    expect(rows()).toEqual(all);
  });

  it("remembers a group toggle per tab", () => {
    mount();
    key("j");
    key("Enter");
    const collapsed = rows();
    setTab("backlog");
    setTab("all");
    expect(rows()).toEqual(collapsed);
  });
});
