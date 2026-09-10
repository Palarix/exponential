export type ShortcutPriority = "global" | "view" | "control" | "overlay";

export interface ShortcutBinding {
  id: string;
  key: string | readonly string[];
  label: string;
  group?: string;
  showInHelp?: boolean;
  run: (event: KeyboardEvent) => void;
  modifiers?: {
    meta?: boolean;
    ctrl?: boolean;
    alt?: boolean;
    shift?: boolean;
  };
  allowInEditable?: boolean;
  preventDefault?: boolean;
  allowRepeat?: boolean;
}

export interface ShortcutRegistration {
  scope: string;
  priority?: ShortcutPriority;
  enabled?: boolean;
  blocksLowerPriorities?: boolean;
  shortcuts: readonly ShortcutBinding[];
}

export interface KeyboardHandlerRegistration extends Omit<ShortcutRegistration, "shortcuts"> {
  shortcuts: readonly Omit<ShortcutBinding, "run">[];
  handler: (event: KeyboardEvent) => void;
}

export interface ActiveShortcut extends Omit<ShortcutBinding, "run"> {
  scope: string;
  priority: ShortcutPriority;
}
