import type { ReactNode } from "react";
import { createPortal } from "react-dom";
import { useKeyboardShortcuts } from "../../keyboard";

interface DialogOverlayProps {
  /** Unique per call site; names the Escape shortcut scope. */
  scope: string;
  onClose: () => void;
  /** Turn off Escape, e.g. while another dialog is stacked on top. */
  enabled?: boolean;
  children: ReactNode;
}

/** Renders a dialog over everything else. Escape and a backdrop click close it. */
export function DialogOverlay({ scope, onClose, enabled = true, children }: DialogOverlayProps) {
  useKeyboardShortcuts({
    scope,
    priority: "overlay",
    enabled,
    shortcuts: [{
      id: `${scope}.cancel`,
      key: "Escape",
      label: "Cancel",
      showInHelp: false,
      allowInEditable: true,
      run: onClose,
    }],
  });

  return createPortal(
    <div className="fixed inset-0 z-[60] flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/30" onClick={onClose} aria-hidden="true" />
      {children}
    </div>,
    document.body,
  );
}

interface DialogPanelProps {
  id: string;
  role?: "dialog" | "alertdialog";
  title: string;
  /** Extra ARIA wiring, such as `aria-describedby`. */
  describedBy?: string;
  children: ReactNode;
  /** The footer's left side; defaults to the esc hint. */
  footerLeft?: ReactNode;
  /** The buttons, right-aligned in order. */
  footer: ReactNode;
}

/** The dialog card: title and body, then a footer bar with the buttons. */
export function DialogPanel({ id, role = "dialog", title, describedBy, children, footerLeft, footer }: DialogPanelProps) {
  return (
    <div role={role} aria-modal="true" aria-labelledby={`${id}-title`} aria-describedby={describedBy} className="relative bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-lg)] max-w-md w-full overflow-hidden">
      <div className="px-5 pt-5 pb-4">
        <h3 id={`${id}-title`} className="text-sm font-semibold text-[var(--color-text-primary)]">{title}</h3>
        {children}
      </div>
      <div className="flex items-center gap-2 px-5 py-3 border-t border-[var(--color-border-subtle)] bg-[var(--color-surface-2)]">
        {footerLeft ?? <EscHint />}
        <div className="flex-1" />
        {footer}
      </div>
    </div>
  );
}

function EscHint() {
  return (
    <span className="flex items-center gap-1.5 text-xs text-[var(--color-text-muted)]">
      <kbd className="inline-flex items-center justify-center min-w-4 h-4 px-1 text-[10px] font-medium text-[var(--color-text-muted)] bg-[var(--color-surface-1)] border border-[var(--color-border-default)] rounded-[var(--radius-sm)]">esc</kbd>
      to cancel
    </span>
  );
}
