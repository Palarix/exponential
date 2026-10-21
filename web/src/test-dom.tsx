import { act, type ReactElement } from "react";
import { createRoot, type Root } from "react-dom/client";
import type { Editor } from "@tiptap/core";

/**
 * Helpers for the few tests that need a live DOM (files with
 * `// @vitest-environment jsdom`), such as the TipTap chat input.
 */

/** Call once per file, in beforeAll: act() and ProseMirror's layout calls. */
export function setupDom() {
  (globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;
  // jsdom has no layout; ProseMirror measures when it scrolls the selection into view.
  const rect = () => ({ x: 0, y: 0, top: 0, left: 0, right: 0, bottom: 0, width: 0, height: 0, toJSON: () => ({}) }) as DOMRect;
  Range.prototype.getClientRects = () => ({ length: 0, item: () => null, [Symbol.iterator]: [][Symbol.iterator] }) as unknown as DOMRectList;
  Range.prototype.getBoundingClientRect = rect;
  document.elementFromPoint = () => null;
}

/** Renders into a fresh element on the page; cleanup unmounts it. */
export function mountDom(el: ReactElement) {
  const host = document.createElement("div");
  document.body.appendChild(host);
  const root: Root = createRoot(host);
  act(() => root.render(el));
  return {
    host,
    rerender: (next: ReactElement) => act(() => root.render(next)),
    cleanup: () => {
      act(() => root.unmount());
      host.remove();
    },
  };
}

/** The TipTap editor behind a ProseMirror element. */
export function editorIn(host: HTMLElement): { dom: HTMLElement; editor: Editor } {
  const dom = host.querySelector(".ProseMirror") as HTMLElement & { editor: Editor };
  return { dom, editor: dom.editor };
}

/** Types text the way the browser does, so input rules run. */
export function typeInto(editor: Editor, text: string) {
  act(() => {
    for (const ch of text) {
      const view = editor.view;
      const { from, to } = view.state.selection;
      const handled = view.someProp("handleTextInput", (f) => f(view, from, to, ch, () => view.state.tr.insertText(ch, from, to)));
      if (!handled) view.dispatch(view.state.tr.insertText(ch, from, to));
    }
  });
}

export function pressKey(el: HTMLElement, key: string, opts: KeyboardEventInit = {}) {
  act(() => {
    el.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true, ...opts }));
  });
}
