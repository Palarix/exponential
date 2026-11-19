import { useContext, useEffect, useLayoutEffect, useRef, useSyncExternalStore } from "react";
import { KeyboardNavContext } from "./context";
import type { KeyboardHandlerRegistration, ShortcutRegistration } from "./types";

function useKeyboardRegistry() {
  const registry = useContext(KeyboardNavContext);
  if (!registry) throw new Error("Keyboard shortcuts require KeyboardNavProvider");
  return registry;
}

export function useKeyboardShortcuts(registration: ShortcutRegistration): void {
  const registry = useKeyboardRegistry();
  const registrationRef = useRef(registration);
  // After every render, before the browser can deliver the next key.
  useLayoutEffect(() => {
    registrationRef.current = registration;
  });
  const tokenRef = useRef(Symbol(registration.scope));
  const metadataSignature = JSON.stringify({
    scope: registration.scope,
    priority: registration.priority,
    enabled: registration.enabled,
    blocksLowerPriorities: registration.blocksLowerPriorities,
    shortcuts: registration.shortcuts.map((shortcut) => ({
      id: shortcut.id,
      key: shortcut.key,
      label: shortcut.label,
      group: shortcut.group,
      showInHelp: shortcut.showInHelp,
      modifiers: shortcut.modifiers,
      allowInEditable: shortcut.allowInEditable,
      preventDefault: shortcut.preventDefault,
      allowRepeat: shortcut.allowRepeat,
    })),
  });

  useEffect(() => registry.register(tokenRef.current, registrationRef), [registry]);
  useEffect(() => registry.registrationChanged(), [registry, metadataSignature]);
}

export function useActiveShortcuts() {
  const registry = useKeyboardRegistry();
  return useSyncExternalStore(registry.subscribe, registry.getSnapshot, registry.getSnapshot);
}

export function useKeyboardHandler({ handler, shortcuts, ...registration }: KeyboardHandlerRegistration): void {
  useKeyboardShortcuts({
    ...registration,
    shortcuts: shortcuts.map((shortcut) => ({ ...shortcut, run: handler })),
  });
}
