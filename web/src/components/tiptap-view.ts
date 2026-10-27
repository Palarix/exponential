import type { Editor } from "@tiptap/core";

/**
 * Whether the editor's ProseMirror view is mounted. Until EditorContent
 * mounts it, and while the editor is rebuilt, editor.view is a stand-in that
 * throws on most reads; isDestroyed is true then too.
 */
export function viewMounted(editor: Editor): boolean {
  return !editor.isDestroyed;
}

/** Runs fn now if the view is mounted, else once it mounts. Returns a cancel. */
export function whenViewMounted(editor: Editor, fn: () => void): () => void {
  if (viewMounted(editor)) {
    fn();
    return () => {};
  }
  const run = () => {
    editor.off("mount", run);
    fn();
  };
  editor.on("mount", run);
  return () => editor.off("mount", run);
}

/** Focuses the editor with the caret where the user clicked, once its view is mounted. */
export function focusAt(editor: Editor, click: { clientX: number; clientY: number }): () => void {
  return whenViewMounted(editor, () => {
    editor.commands.focus();
    const pos = editor.view.posAtCoords({ left: click.clientX, top: click.clientY });
    if (pos) editor.commands.setTextSelection(pos.pos);
  });
}
