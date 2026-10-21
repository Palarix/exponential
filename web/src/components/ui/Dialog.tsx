import { useEffect, useRef, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { X } from "lucide-react";
import { useKeyboardShortcuts } from "../../keyboard";
import { cn } from "../../utils/cn";
import { DIALOG_LAYER } from "./layers";

const FOCUSABLE = 'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])';

interface DialogOverlayProps {
  /** Unique per call site; names the Escape shortcut scope. */
  scope: string;
  onClose: () => void;
  /** Turn off Escape and Tab, e.g. while another dialog is stacked on top. */
  enabled?: boolean;
  children: ReactNode;
}

/**
 * Renders a dialog over everything else. Escape and a backdrop click close it,
 * Tab stays inside it, and the page behind does not scroll. Focus moves in on
 * mount and back to the opener on unmount.
 */
export function DialogOverlay({ scope, onClose, enabled = true, children }: DialogOverlayProps) {
  const ref = useRef<HTMLDivElement>(null);

  const trapTab = (e: KeyboardEvent) => {
    const focusable = Array.from(ref.current?.querySelectorAll<HTMLElement>(FOCUSABLE) ?? []);
    if (focusable.length === 0) {
      e.preventDefault();
      return;
    }
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
  };

  useKeyboardShortcuts({
    scope,
    priority: "overlay",
    enabled,
    shortcuts: [
      { id: `${scope}.cancel`, key: "Escape", label: "Cancel", showInHelp: false, allowInEditable: true, run: onClose },
      // Without a shift modifier, this also matches Shift+Tab.
      { id: `${scope}.focus-trap`, key: "Tab", label: "Move focus", showInHelp: false, allowInEditable: true, preventDefault: false, run: trapTab },
    ],
  });

  useEffect(() => {
    const opener = document.activeElement;
    // Restore what was there, so a dialog stacked on another keeps the page locked.
    const overflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    const frame = requestAnimationFrame(() => {
      const root = ref.current;
      if (!root || root.contains(document.activeElement)) return;
      const target = root.querySelector<HTMLElement>(FOCUSABLE) ?? root.querySelector<HTMLElement>('[role="dialog"], [role="alertdialog"]');
      target?.focus();
    });
    return () => {
      cancelAnimationFrame(frame);
      document.body.style.overflow = overflow;
      if (opener instanceof HTMLElement) opener.focus();
    };
  }, []);

  return createPortal(
    <div ref={ref} className={cn("fixed inset-0 flex items-center justify-center p-4", DIALOG_LAYER)}>
      <div className="absolute inset-0 bg-black/40 animate-fade-in" onClick={onClose} aria-hidden="true" />
      {children}
    </div>,
    document.body,
  );
}

export type DialogSize = "sm" | "md" | "lg" | "xl" | "2xl";

const SIZE_CLASS: Record<DialogSize, string> = {
  sm: "max-w-96",
  md: "max-w-112",
  lg: "max-w-128",
  xl: "max-w-168",
  "2xl": "max-w-215",
};

interface DialogProps {
  id: string;
  role?: "dialog" | "alertdialog";
  title: ReactNode;
  /** Shown before the title. */
  icon?: ReactNode;
  /** Controls in the header, before the close button. */
  headerActions?: ReactNode;
  /** Shows a close button in the header. */
  onClose?: () => void;
  /** Extra ARIA wiring, such as `aria-describedby`. */
  describedBy?: string;
  size?: DialogSize;
  children: ReactNode;
  /** The footer's left side; defaults to the esc hint when there is a footer. */
  footerLeft?: ReactNode;
  /** The buttons, right-aligned in order. Without it and `footerLeft`, there is no footer. */
  footer?: ReactNode;
}

/**
 * The dialog card: a header, a body that scrolls, and a footer with the buttons.
 * All three share one surface with no dividers, and 16px separates every edge.
 */
export function Dialog({
  id, role = "dialog", title, icon, headerActions, onClose, describedBy, size = "md", children, footerLeft, footer,
}: DialogProps) {
  const hasFooter = footer != null || footerLeft != null;
  return (
    <div
      role={role}
      aria-modal="true"
      aria-labelledby={`${id}-title`}
      aria-describedby={describedBy}
      tabIndex={-1}
      className={cn(
        "relative flex flex-col w-full max-h-[85vh] overflow-hidden outline-none animate-fade-in",
        "bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-lg)]",
        SIZE_CLASS[size],
      )}
    >
      <div data-slot="header" className="flex items-center gap-2 p-4 pb-0 shrink-0">
        {icon && <span className="flex shrink-0 text-[var(--color-text-muted)]">{icon}</span>}
        <h2 id={`${id}-title`} className="flex-1 min-w-0 truncate text-sm font-semibold text-[var(--color-text-primary)]">{title}</h2>
        {headerActions}
        {onClose && (
          <button
            type="button"
            onClick={onClose}
            aria-label="Close"
            className="-m-1 p-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-3)] transition-colors duration-[var(--duration-fast)]"
          >
            <X size={16} />
          </button>
        )}
      </div>
      <div data-slot="body" className="p-4 flex-1 min-h-0 overflow-y-auto">
        {children}
      </div>
      {hasFooter && (
        <div data-slot="footer" className="p-4 pt-0 flex items-center gap-2 shrink-0">
          {footerLeft ?? <EscHint />}
          <div className="flex-1" />
          {footer}
        </div>
      )}
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
