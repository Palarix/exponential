import { useEffect, useRef } from "react";
import { createPortal } from "react-dom";

interface ShortcutEntry {
  keys: string[];
  label: string;
}

interface ShortcutGroup {
  title: string;
  shortcuts: ShortcutEntry[];
}

const mod = navigator.platform.includes("Mac") ? "⌘" : "Ctrl";

const GLOBAL: ShortcutGroup = {
  title: "Global",
  shortcuts: [
    { keys: ["?"], label: "Show keyboard shortcuts" },
    { keys: [mod, "K"], label: "Open command palette" },
    { keys: ["C"], label: "Create new issue" },
    { keys: ["G", "O"], label: "Go to Overview" },
    { keys: ["G", "M"], label: "Go to My Issues" },
    { keys: ["G", "N"], label: "Go to Notifications" },
    { keys: ["G", "I"], label: "Go to Issues" },
    { keys: ["G", "B"], label: "Go to Board" },
    { keys: ["G", "C"], label: "Go to Cycles" },
    { keys: ["G", "L"], label: "Go to Labels" },
    { keys: ["G", "D"], label: "Go to Dependencies" },
    { keys: ["G", "T"], label: "Go to Timeline" },
  ],
};

const BACKLOG: ShortcutGroup = {
  title: "Backlog",
  shortcuts: [
    { keys: ["J", "↓"], label: "Next issue" },
    { keys: ["K", "↑"], label: "Previous issue" },
    { keys: ["Enter"], label: "Open issue" },
    { keys: ["→"], label: "Expand group / sub-issues" },
    { keys: ["←"], label: "Collapse group / sub-issues" },
    { keys: ["S"], label: "Set status" },
    { keys: ["L"], label: "Set labels" },
    { keys: ["E"], label: "Set estimate" },
    { keys: ["/"], label: "Focus search" },
    { keys: ["F"], label: "Toggle filters" },
    { keys: ["."], label: "Copy issue ID" },
    { keys: ["["], label: "Previous tab" },
    { keys: ["]"], label: "Next tab" },
    { keys: ["}"], label: "Expand all parents" },
    { keys: ["{"], label: "Collapse all parents" },
  ],
};

const ISSUE_DETAIL: ShortcutGroup = {
  title: "Issue Detail",
  shortcuts: [
    { keys: ["J", "→"], label: "Next issue" },
    { keys: ["K", "←"], label: "Previous issue" },
    { keys: ["S"], label: "Set status" },
    { keys: ["L"], label: "Set labels" },
    { keys: ["E"], label: "Set estimate" },
    { keys: ["P"], label: "Set priority" },
    { keys: ["A"], label: "Set assignee" },
    { keys: ["M"], label: "Add comment" },
    { keys: ["1–5"], label: "Quick set status" },
    { keys: ["."], label: "Copy issue ID" },
    { keys: ["Esc"], label: "Close issue" },
  ],
};

const PICKERS: ShortcutGroup = {
  title: "Pickers & Menus",
  shortcuts: [
    { keys: ["↑", "↓"], label: "Navigate options" },
    { keys: ["1–5"], label: "Quick select by number" },
    { keys: ["Enter"], label: "Confirm selection" },
    { keys: ["Esc"], label: "Close picker" },
    { keys: ["Type"], label: "Filter options" },
  ],
};

const INBOX: ShortcutGroup = {
  title: "Notifications",
  shortcuts: [
    { keys: ["J", "↓"], label: "Next notification" },
    { keys: ["K", "↑"], label: "Previous notification" },
    { keys: ["F"], label: "Toggle filters" },
    { keys: ["R"], label: "Mark all as read" },
  ],
};

const ALL_GROUPS = [GLOBAL, BACKLOG, ISSUE_DETAIL, INBOX, PICKERS];

function Kbd({ children }: { children: string }) {
  return (
    <kbd className="inline-flex items-center justify-center min-w-5 h-5 px-1.5 text-xs font-medium text-[var(--color-text-secondary)] bg-[var(--color-surface-1)] border border-[var(--color-border-default)] rounded-[var(--radius-sm)]">
      {children}
    </kbd>
  );
}

interface KeyboardHelpProps {
  isOpen: boolean;
  onClose: () => void;
}

export default function KeyboardHelp({ isOpen, onClose }: KeyboardHelpProps) {
  const overlayRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!isOpen) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape" || e.key === "?") {
        e.preventDefault();
        onClose();
      }
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return createPortal(
    <div className="fixed inset-0 z-[200] flex items-start justify-center pt-[10vh]">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div
        ref={overlayRef}
        className="relative w-full max-w-2xl bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-xl)] shadow-[var(--shadow-lg)] overflow-hidden animate-fade-in"
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

        <div className="max-h-[65vh] overflow-y-auto px-5 py-4 grid grid-cols-2 gap-x-12 gap-y-6">
          {ALL_GROUPS.map(group => (
            <div key={group.title}>
              <h3 className="text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wider mb-3">{group.title}</h3>
              <div className="space-y-2">
                {group.shortcuts.map(sc => (
                  <div key={sc.label} className="flex items-center justify-between gap-3">
                    <span className="text-sm text-[var(--color-text-secondary)]">{sc.label}</span>
                    <span className="flex items-center gap-1 shrink-0">
                      {sc.keys.map((k, i) => (
                        <span key={i} className="flex items-center gap-1">
                          {i > 0 && <span className="text-xs text-[var(--color-text-muted)]">/</span>}
                          <Kbd>{k}</Kbd>
                        </span>
                      ))}
                    </span>
                  </div>
                ))}
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
