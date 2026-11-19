// @vitest-environment jsdom
import { act, useContext, type ReactNode } from "react";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { mountDom, pressKey, setupDom } from "./test-dom";
import { KeyboardNavProvider, useActiveShortcuts } from "./keyboard";
import { AppNavContext } from "./app/contexts";
import { deriveAppConfig } from "./api/query-utils";
import { VIEW_BY_GO_KEY } from "./views";
import App from "./App";

/**
 * App's global shortcuts: the palette, new issue, and the g-key views.
 * The views and the data are stubbed; the shortcut wiring is App's own.
 */

const config = vi.hoisted(() => ({ current: undefined as import("./api/query-utils").AppConfig | undefined }));
vi.mock("./api/queries", () => ({
  useIssues: () => ({ isError: false }),
  useIssueList: () => [],
  useConfig: () => (config.current ??= deriveAppConfig(undefined)),
  useInbox: () => ({ unread: 0 }),
  useIssueDrafts: () => [],
}));
vi.mock("./live/useLive", () => ({ useLive: () => {} }));
vi.mock("./components/ui", async (importOriginal) => ({
  ...(await importOriginal<typeof import("./components/ui")>()),
  useToast: () => () => {},
}));
const { passthrough } = vi.hoisted(() => ({
  passthrough: ({ children }: { children?: ReactNode }) => children,
}));
vi.mock("./components/Layout/Layout", () => ({ default: passthrough }));
vi.mock("./components/Agent/AgentChatProvider", () => ({ default: passthrough }));
vi.mock("./components/Milestones/MilestoneAssign", () => ({ default: passthrough }));
vi.mock("./components/Agent/TabStatus", () => ({ default: () => null }));
vi.mock("./components/Layout/ViewRouter", () => ({
  default: function View() {
    return <div data-testid="view">{useContext(AppNavContext)!.view}</div>;
  },
}));
vi.mock("./components/NewIssueModal/NewIssueModal", () => ({
  default: ({ isOpen }: { isOpen: boolean }) => (isOpen ? <div data-testid="new-issue" /> : null),
}));
vi.mock("./components/CommandPalette/CommandPalette", () => ({
  default: ({ isOpen }: { isOpen: boolean }) => (isOpen ? <div data-testid="palette" /> : null),
}));
vi.mock("./components/KeyboardHelp/KeyboardHelp", () => ({
  // The shortcut list the help overlay shows, in its order.
  default: function Help() {
    const list = useActiveShortcuts()
      .map(({ id, key, label, group, showInHelp, modifiers }) => ({ id, key, label, group, showInHelp, modifiers }));
    return <pre data-testid="shortcuts">{JSON.stringify(list)}</pre>;
  },
}));

beforeAll(setupDom);

let cleanup: (() => void) | undefined;
beforeEach(() => {
  window.location.hash = "";
  const m = mountDom(<KeyboardNavProvider><App /></KeyboardNavProvider>);
  cleanup = m.cleanup;
});
afterEach(() => {
  cleanup?.();
  cleanup = undefined;
});

const el = (id: string) => document.querySelector(`[data-testid="${id}"]`);
const view = () => el("view")?.textContent;
const key = (k: string, opts?: KeyboardEventInit) => pressKey(document.body, k, opts);

describe("App global shortcuts", () => {
  it.each(Object.entries(VIEW_BY_GO_KEY))("g %s goes to %s", (k, target) => {
    // Start elsewhere, so the shortcut has to move.
    window.location.hash = target === "board" ? "#/issues" : "#/board";
    act(() => { window.dispatchEvent(new HashChangeEvent("hashchange")); });
    expect(view()).not.toBe(target);
    key("g");
    key(k);
    expect(view()).toBe(target);
  });

  it("c opens a new issue", () => {
    key("c");
    expect(el("new-issue")).not.toBeNull();
  });

  it("meta+k and ctrl+k open the palette", () => {
    key("k", { metaKey: true });
    expect(el("palette")).not.toBeNull();
  });

  it("g-keys and c do nothing while the palette is open", () => {
    key("k", { ctrlKey: true });
    expect(el("palette")).not.toBeNull();
    const before = view();
    key("g");
    key("i");
    key("c");
    expect(view()).toBe(before);
    expect(el("new-issue")).toBeNull();
  });

  it("g-keys do nothing while the new-issue form is open", () => {
    key("c");
    const before = view();
    key("g");
    key("b");
    expect(view()).toBe(before);
  });

  it("lists the same shortcuts in help", () => {
    expect(el("shortcuts")?.textContent).toMatchSnapshot();
  });
});
