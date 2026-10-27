// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest";
import { Editor } from "@tiptap/core";
import { EditorView } from "@tiptap/pm/view";
import StarterKit from "@tiptap/starter-kit";
import { focusAt, viewMounted, whenViewMounted } from "./tiptap-view";

let editor: Editor | undefined;
afterEach(() => {
  editor?.destroy();
  editor = undefined;
  vi.restoreAllMocks();
});

// An editor with a null element has no view until mount() gives it one.
const unmounted = () => (editor = new Editor({ element: null, extensions: [StarterKit] }));

describe("whenViewMounted", () => {
  it("runs right away when the view is mounted", () => {
    editor = new Editor({ element: document.createElement("div"), extensions: [StarterKit] });
    const fn = vi.fn();
    whenViewMounted(editor, fn);
    expect(viewMounted(editor)).toBe(true);
    expect(fn).toHaveBeenCalledOnce();
  });

  it("waits for the view to mount, then runs once", () => {
    const ed = unmounted();
    const fn = vi.fn(() => ed.view.dom);
    whenViewMounted(ed, fn);
    expect(viewMounted(ed)).toBe(false);
    expect(fn).not.toHaveBeenCalled();
    ed.mount(document.createElement("div"));
    expect(fn).toHaveBeenCalledOnce();
    expect(fn).toHaveReturnedWith(ed.view.dom);
    ed.unmount();
    ed.mount(document.createElement("div"));
    expect(fn).toHaveBeenCalledOnce();
  });

  it("does not run once cancelled", () => {
    const ed = unmounted();
    const fn = vi.fn();
    whenViewMounted(ed, fn)();
    ed.mount(document.createElement("div"));
    expect(fn).not.toHaveBeenCalled();
  });
});

describe("focusAt", () => {
  it("puts the caret where the click was once the view mounts, without throwing before", () => {
    vi.spyOn(EditorView.prototype, "posAtCoords").mockReturnValue({ pos: 3, inside: -1 });
    const ed = unmounted();
    ed.commands.setContent("<p>hello</p>");
    expect(() => focusAt(ed, { clientX: 5, clientY: 5 })).not.toThrow();
    expect(ed.state.selection.from).not.toBe(3);
    ed.mount(document.createElement("div"));
    expect(ed.state.selection.from).toBe(3);
  });
});
