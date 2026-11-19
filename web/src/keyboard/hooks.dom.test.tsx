// @vitest-environment jsdom
import { useState } from "react";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { mountDom, pressKey, setupDom } from "../test-dom";
import { KeyboardNavProvider } from "./KeyboardNavProvider";
import { useKeyboardHandler, useKeyboardShortcuts } from "./hooks";

beforeAll(setupDom);

let cleanup: (() => void) | undefined;
afterEach(() => {
  cleanup?.();
  cleanup = undefined;
});

function Shortcut({ n, spy }: { n: number; spy: (n: number) => void }) {
  useKeyboardShortcuts({ scope: "test", shortcuts: [{ id: "x", key: "x", label: "X", run: () => spy(n) }] });
  return null;
}

function Handler({ n, spy }: { n: number; spy: (n: number) => void }) {
  useKeyboardHandler({ scope: "test", handler: () => spy(n), shortcuts: [{ id: "x", key: "x", label: "X" }] });
  return null;
}

/** Counts up on each "x"; the shortcut must see the count of the last render. */
function Counter({ spy }: { spy: (n: number) => void }) {
  const [n, setN] = useState(0);
  useKeyboardShortcuts({ scope: "test", shortcuts: [{ id: "x", key: "x", label: "X", run: () => { spy(n); setN(n + 1); } }] });
  return null;
}

describe("keyboard hooks run the closure of the latest render", () => {
  it("useKeyboardShortcuts, after a re-render with new props", () => {
    const spy = vi.fn();
    const m = mountDom(<KeyboardNavProvider><Shortcut n={1} spy={spy} /></KeyboardNavProvider>);
    cleanup = m.cleanup;
    m.rerender(<KeyboardNavProvider><Shortcut n={2} spy={spy} /></KeyboardNavProvider>);
    pressKey(document.body, "x");
    expect(spy).toHaveBeenLastCalledWith(2);
  });

  it("useKeyboardHandler, after a re-render with new props", () => {
    const spy = vi.fn();
    const m = mountDom(<KeyboardNavProvider><Handler n={1} spy={spy} /></KeyboardNavProvider>);
    cleanup = m.cleanup;
    m.rerender(<KeyboardNavProvider><Handler n={2} spy={spy} /></KeyboardNavProvider>);
    pressKey(document.body, "x");
    expect(spy).toHaveBeenLastCalledWith(2);
  });

  it("after a state update the shortcut itself caused", () => {
    const spy = vi.fn();
    const m = mountDom(<KeyboardNavProvider><Counter spy={spy} /></KeyboardNavProvider>);
    cleanup = m.cleanup;
    pressKey(document.body, "x");
    pressKey(document.body, "x");
    pressKey(document.body, "x");
    expect(spy.mock.calls).toEqual([[0], [1], [2]]);
  });
});
