import { useRef } from "react";
import { createPortal } from "react-dom";
import { useActiveShortcuts, useKeyboardShortcuts } from "../../keyboard";
import { displayKeys, groupShortcuts, keyJoin } from "./keyboard-help-utils";

function Kbd({ children }: { children: string }) {
  return (
    <kbd className="inline-flex items-center justify-center min-w-5 h-5 px-1.5 text-xs font-medium text-[var(--color-text-secondary)] bg-[var(--color-surface-1)] border border-[var(--color-border-default)] rounded-[var(--radius-sm)]">
      {children}
    </kbd>
  );
}

const JOIN_GLYPHS = { or: "/", seq: "→", combo: "+" } as const;

function KeySeparator({ join = "or" }: { join?: "or" | "seq" | "combo" }) {
  return (
    <span className="text-xs text-[var(--color-text-muted)]">
      {JOIN_GLYPHS[join]}
    </span>
  );
}

interface KeyboardHelpProps {
  isOpen: boolean;
  onToggle: () => void;
  onClose: () => void;
}

export default function KeyboardHelp({ isOpen, onToggle, onClose }: KeyboardHelpProps) {
  const overlayRef = useRef<HTMLDivElement>(null);
  const groups = groupShortcuts(useActiveShortcuts());
  const isMac = navigator.platform.includes("Mac");

  useKeyboardShortcuts({
    scope: "keyboard-help.toggle",
    priority: "global",
    shortcuts: [{
      id: "keyboard-help.toggle",
      key: "?",
      label: "Show keyboard shortcuts",
      group: "Global",
      run: onToggle,
    }],
  });

  useKeyboardShortcuts({
    scope: "keyboard-help.overlay",
    priority: "overlay",
    enabled: isOpen,
    shortcuts: ["Escape", "?"].map((key) => ({
      id: `keyboard-help.close.${key}`,
      key,
      label: "Close keyboard shortcuts",
      showInHelp: false,
      allowInEditable: true,
      run: onClose,
    })),
  });

  if (!isOpen) return null;

  return createPortal(
    <div className="fixed inset-0 z-[200] flex items-start justify-center pt-[10vh]">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div
        ref={overlayRef}
        className="relative w-full max-w-5xl bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-xl)] shadow-[var(--shadow-lg)] overflow-hidden animate-fade-in"
      >
        <div className="flex items-center justify-between px-5 py-3 border-b border-[var(--color-border-subtle)]">
          <h2 className="text-sm font-semibold text-[var(--color-text-primary)]">Keyboard Shortcuts</h2>
          <button
            onClick={onClose}
            className="p-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-3)] transition-colors"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M18 6 6 18" /><path d="m6 6 12 12" />
            </svg>
          </button>
        </div>

        <div className="max-h-[65vh] overflow-y-auto px-5 py-4 space-y-6">
          {groups.map(group => (
            <div key={group.title}>
              <h3 className="text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wider mb-3">{group.title}</h3>
              <div className="columns-1 gap-8 md:columns-2 lg:columns-3">
                {group.shortcuts.map(shortcut => {
                  return (
                    <div key={shortcut.id} className="mb-2 flex break-inside-avoid items-center justify-between gap-3">
                      <span className="text-sm text-[var(--color-text-secondary)]">{shortcut.label}</span>
                      <span className="flex items-center gap-1 shrink-0">
                        {shortcut.bindings.map((binding, bindingIndex) => {
                          const keys = displayKeys(binding, isMac);
                          const join = keyJoin(binding);
                          return (
                            <span key={binding.id} className="flex items-center gap-1">
                              {bindingIndex > 0 && <KeySeparator />}
                              {keys.map((key, keyIndex) => (
                                <span key={keyIndex} className="flex items-center gap-1">
                                  {keyIndex > 0 && <KeySeparator join={join} />}
                                  <Kbd>{key}</Kbd>
                                </span>
                              ))}
                            </span>
                          );
                        })}
                      </span>
                    </div>
                  );
                })}
              </div>
            </div>
          ))}
        </div>

        <div className="flex items-center justify-end gap-1.5 px-5 py-2.5 border-t border-[var(--color-border-subtle)] text-xs text-[var(--color-text-muted)]">
          Press <Kbd>?</Kbd> or <Kbd>Esc</Kbd> to close
        </div>
      </div>
    </div>,
    document.body,
  );
}
