import type { ActiveShortcut } from "../../keyboard";

export interface ShortcutGroup {
  title: string;
  shortcuts: DisplayShortcut[];
}

export interface DisplayShortcut {
  id: string;
  label: string;
  bindings: ActiveShortcut[];
}

export function groupShortcuts(shortcuts: readonly ActiveShortcut[]): ShortcutGroup[] {
  const viewGroup = shortcuts.find((s) => s.priority === "view")?.group;

  const groups = new Map<string, Map<string, DisplayShortcut>>();
  for (const shortcut of shortcuts) {
    let title = shortcut.group || shortcut.scope;
    if (shortcut.priority === "control" && viewGroup && title !== "Global") {
      title = viewGroup;
    }
    const group = groups.get(title) || new Map<string, DisplayShortcut>();
    const display = group.get(shortcut.label) || {
      id: shortcut.id,
      label: shortcut.label,
      bindings: [],
    };
    display.bindings.push(shortcut);
    group.set(shortcut.label, display);
    groups.set(title, group);
  }
  return [...groups.entries()]
    .map(([title, entries]) => ({ title, shortcuts: [...entries.values()] }))
    .sort((a, b) => (a.title === "Global" ? -1 : b.title === "Global" ? 1 : 0));
}

const KEY_LABELS: Record<string, string> = {
  ArrowDown: "↓",
  ArrowUp: "↑",
  ArrowLeft: "←",
  ArrowRight: "→",
  Escape: "Esc",
  Enter: "Enter",
  " ": "Space",
};

export function displayKeys(shortcut: ActiveShortcut, isMac: boolean): string[] {
  const keys = typeof shortcut.key === "string" ? [shortcut.key] : [...shortcut.key];
  const modifiers: string[] = [];
  if (shortcut.modifiers?.meta) modifiers.push(isMac ? "⌘" : "Meta");
  if (shortcut.modifiers?.ctrl) modifiers.push("Ctrl");
  if (shortcut.modifiers?.alt) modifiers.push(isMac ? "⌥" : "Alt");
  if (shortcut.modifiers?.shift) modifiers.push("Shift");
  return [...modifiers, ...keys.map((key) => KEY_LABELS[key] || (key.length === 1 ? key.toUpperCase() : key))];
}

export function keyJoin(shortcut: ActiveShortcut): "seq" | "combo" | undefined {
  if (shortcut.modifiers && Object.values(shortcut.modifiers).some(Boolean)) return "combo";
  return Array.isArray(shortcut.key) ? "seq" : undefined;
}
