import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { KeyboardRegistry } from "./registry";
import type { ShortcutRegistration } from "./types";

class InputStub {}
class TextAreaStub {}
class ElementStub { isContentEditable = false; }

function keyboardEvent(key: string, overrides: Partial<KeyboardEvent> = {}) {
  return {
    key,
    target: null,
    metaKey: false,
    ctrlKey: false,
    altKey: false,
    shiftKey: false,
    repeat: false,
    isComposing: false,
    preventDefault: vi.fn(),
    ...overrides,
  } as unknown as KeyboardEvent;
}

function add(registry: KeyboardRegistry, registration: ShortcutRegistration) {
  return registry.register(Symbol(registration.scope), { current: registration });
}

describe("KeyboardRegistry", () => {
  beforeEach(() => {
    vi.stubGlobal("HTMLInputElement", InputStub);
    vi.stubGlobal("HTMLTextAreaElement", TextAreaStub);
    vi.stubGlobal("HTMLElement", ElementStub);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("runs only the highest-priority matching binding", () => {
    const registry = new KeyboardRegistry();
    const global = vi.fn();
    const overlay = vi.fn();
    add(registry, { scope: "global", priority: "global", shortcuts: [{ id: "global.x", key: "x", label: "Global", run: global }] });
    add(registry, { scope: "overlay", priority: "overlay", shortcuts: [{ id: "overlay.x", key: "x", label: "Overlay", run: overlay }] });

    registry.dispatch(keyboardEvent("x"));

    expect(overlay).toHaveBeenCalledOnce();
    expect(global).not.toHaveBeenCalled();
  });

  it("prevents an active overlay from leaking unmatched keys to lower scopes", () => {
    const registry = new KeyboardRegistry();
    const view = vi.fn();
    add(registry, { scope: "view", priority: "view", shortcuts: [{ id: "view.j", key: "j", label: "Next", run: view }] });
    add(registry, { scope: "menu", priority: "overlay", shortcuts: [{ id: "menu.escape", key: "Escape", label: "Close", run: vi.fn() }] });

    registry.dispatch(keyboardEvent("j"));

    expect(view).not.toHaveBeenCalled();
  });

  it("keeps underlying shortcuts visible while an overlay blocks their execution", () => {
    const registry = new KeyboardRegistry();
    const navigate = vi.fn();
    add(registry, {
      scope: "global.navigation",
      priority: "global",
      shortcuts: [{ id: "nav.issues", key: ["g", "i"], label: "Go to Issues", run: navigate }],
    });
    add(registry, {
      scope: "overlay.keyboard-help",
      priority: "overlay",
      shortcuts: [{ id: "help.close", key: "Escape", label: "Close help", run: vi.fn() }],
    });

    expect(registry.getSnapshot()).toEqual(expect.arrayContaining([
      expect.objectContaining({ id: "nav.issues", key: ["g", "i"] }),
    ]));

    registry.dispatch(keyboardEvent("g"));
    registry.dispatch(keyboardEvent("i"));

    expect(navigate).not.toHaveBeenCalled();
  });

  it("honors enabled state, modifiers, editable guards, and cleanup", () => {
    const registry = new KeyboardRegistry();
    const run = vi.fn();
    const remove = add(registry, {
      scope: "test",
      enabled: true,
      shortcuts: [{ id: "test.command", key: "k", label: "Command", modifiers: { meta: true }, run }],
    });

    registry.dispatch(keyboardEvent("k"));
    registry.dispatch(keyboardEvent("k", { metaKey: true, target: new InputStub() as EventTarget }));
    expect(run).not.toHaveBeenCalled();

    registry.dispatch(keyboardEvent("k", { metaKey: true }));
    expect(run).toHaveBeenCalledOnce();

    remove();
    registry.dispatch(keyboardEvent("k", { metaKey: true }));
    expect(run).toHaveBeenCalledOnce();
  });

  it("dispatches key sequences and resets an expired prefix", () => {
    vi.useFakeTimers();
    const registry = new KeyboardRegistry();
    const run = vi.fn();
    add(registry, { scope: "nav", shortcuts: [{ id: "nav.board", key: ["g", "b"], label: "Go to Board", run }] });

    registry.dispatch(keyboardEvent("g"));
    registry.dispatch(keyboardEvent("b"));
    expect(run).toHaveBeenCalledOnce();

    registry.dispatch(keyboardEvent("g"));
    vi.advanceTimersByTime(1001);
    registry.dispatch(keyboardEvent("b"));
    expect(run).toHaveBeenCalledOnce();
  });

  it("consumes an invalid sequence continuation", () => {
    const registry = new KeyboardRegistry();
    const sequence = vi.fn();
    const create = vi.fn();
    add(registry, { scope: "nav", shortcuts: [
      { id: "nav.board", key: ["g", "b"], label: "Go to Board", run: sequence },
      { id: "create", key: "c", label: "Create", run: create },
    ] });

    registry.dispatch(keyboardEvent("g"));
    registry.dispatch(keyboardEvent("c"));

    expect(sequence).not.toHaveBeenCalled();
    expect(create).not.toHaveBeenCalled();
  });

  it("publishes enabled, visible metadata without callbacks", () => {
    const registry = new KeyboardRegistry();
    add(registry, {
      scope: "view:board",
      priority: "view",
      shortcuts: [
        { id: "board.next", key: "j", label: "Next card", group: "Board", run: vi.fn() },
        { id: "board.hidden", key: "Escape", label: "Close", showInHelp: false, run: vi.fn() },
      ],
    });

    expect(registry.getSnapshot()).toEqual([
      expect.objectContaining({ id: "board.next", scope: "view:board", priority: "view" }),
    ]);
    expect(registry.getSnapshot()[0]).not.toHaveProperty("run");
  });
});
