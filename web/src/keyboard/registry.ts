import { isEditableTarget } from "../utils/keyboard";
import type { ActiveShortcut, ShortcutBinding, ShortcutPriority, ShortcutRegistration } from "./types";

type RegistrationRef = { current: ShortcutRegistration };
type Entry = { order: number; registration: RegistrationRef };

const PRIORITY: Record<ShortcutPriority, number> = {
  global: 0,
  view: 1,
  control: 2,
  overlay: 3,
};

function keysOf(binding: ShortcutBinding): readonly string[] {
  return typeof binding.key === "string" ? [binding.key] : binding.key;
}

function modifiersMatch(event: KeyboardEvent, binding: ShortcutBinding): boolean {
  const modifiers = binding.modifiers;
  return (
    event.metaKey === (modifiers?.meta ?? false) &&
    event.ctrlKey === (modifiers?.ctrl ?? false) &&
    event.altKey === (modifiers?.alt ?? false) &&
    (modifiers?.shift === undefined || event.shiftKey === modifiers.shift)
  );
}

export class KeyboardRegistry {
  private entries = new Map<symbol, Entry>();
  private listeners = new Set<() => void>();
  private nextOrder = 0;
  private pending: string[] = [];
  private sequenceTimer: ReturnType<typeof setTimeout> | undefined;
  private snapshot: ActiveShortcut[] = [];

  register(token: symbol, registration: RegistrationRef): () => void {
    this.entries.set(token, { order: this.nextOrder++, registration });
    this.updateSnapshot();
    return () => {
      this.entries.delete(token);
      this.updateSnapshot();
    };
  }

  registrationChanged(): void {
    this.updateSnapshot();
  }

  subscribe = (listener: () => void): (() => void) => {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  };

  getSnapshot = (): ActiveShortcut[] => this.snapshot;

  dispatch = (event: KeyboardEvent): void => {
    if (event.isComposing) return;

    const enabledEntries = [...this.entries.values()]
      .filter(({ registration }) => registration.current.enabled !== false);
    const blockingPriority = enabledEntries.reduce((highest, { registration }) => {
      const { priority = "view", blocksLowerPriorities = priority === "overlay" } = registration.current;
      return blocksLowerPriorities ? Math.max(highest, PRIORITY[priority]) : highest;
    }, -1);

    const available = enabledEntries
      .filter(({ registration }) => PRIORITY[registration.current.priority ?? "view"] >= blockingPriority)
      .flatMap((entry) =>
        entry.registration.current.shortcuts.map((binding) => ({
          binding,
          order: entry.order,
          priority: entry.registration.current.priority ?? "view",
        })),
      )
      .filter(({ binding }) =>
        modifiersMatch(event, binding) &&
        (!event.repeat || binding.allowRepeat !== false) &&
        (!isEditableTarget(event) || binding.allowInEditable === true),
      );

    const tryKeys = (pressed: readonly string[]) => {
      const matches = available.filter(({ binding }) => {
        const keys = keysOf(binding);
        return pressed.every((key, index) => keys[index] === key);
      });
      const complete = matches.filter(({ binding }) => keysOf(binding).length === pressed.length);
      return { matches, complete };
    };

    const pressed = [...this.pending, event.key];
    const result = tryKeys(pressed);
    if (result.matches.length === 0 && this.pending.length > 0) {
      this.resetSequence();
      return;
    }
    if (result.matches.length === 0) return;

    if (result.complete.length === 0) {
      event.preventDefault();
      this.pending = pressed;
      clearTimeout(this.sequenceTimer);
      this.sequenceTimer = setTimeout(() => this.resetSequence(), 1000);
      return;
    }

    this.resetSequence();
    const sorted = result.complete.sort(
      (a, b) => PRIORITY[b.priority] - PRIORITY[a.priority] || b.order - a.order,
    );
    const winner = sorted[0];
    const conflicting = sorted[1];
    if (conflicting && PRIORITY[conflicting.priority] === PRIORITY[winner.priority] && import.meta.env.DEV) {
      console.warn(`Keyboard shortcut conflict: ${winner.binding.id} overrides ${conflicting.binding.id}`);
    }
    if (winner.binding.preventDefault !== false) event.preventDefault();
    winner.binding.run(event);
  };

  private resetSequence(): void {
    this.pending = [];
    clearTimeout(this.sequenceTimer);
    this.sequenceTimer = undefined;
  }

  private updateSnapshot(): void {
    this.snapshot = [...this.entries.values()]
      .filter(({ registration }) => registration.current.enabled !== false)
      .flatMap(({ registration }) => {
        const { scope, priority = "view", shortcuts } = registration.current;
        return shortcuts
          .filter((shortcut) => shortcut.showInHelp !== false)
          .map((shortcut) => ({
            id: shortcut.id,
            key: shortcut.key,
            label: shortcut.label,
            group: shortcut.group,
            showInHelp: shortcut.showInHelp,
            modifiers: shortcut.modifiers,
            allowInEditable: shortcut.allowInEditable,
            preventDefault: shortcut.preventDefault,
            allowRepeat: shortcut.allowRepeat,
            scope,
            priority,
          }));
      });
    this.listeners.forEach((listener) => listener());
  }
}
