import { useId, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { useKeyboardShortcuts } from "../../keyboard";
import { cn } from "../../utils/cn";

interface ConfirmDialogContent {
  title: string;
  description: ReactNode;
  /** Content between the title and the description, such as a card naming what is affected. */
  body?: ReactNode;
  /** The buttons, right-aligned in order. */
  children: ReactNode;
}

interface ConfirmDialogProps extends ConfirmDialogContent {
  /** Unique per call site; names the Escape shortcut scope. */
  scope: string;
  onCancel: () => void;
}

/** A small alertdialog over everything else. Escape and a backdrop click cancel it. */
export default function ConfirmDialog({ scope, onCancel, ...content }: ConfirmDialogProps) {
  const id = useId();

  useKeyboardShortcuts({
    scope,
    priority: "overlay",
    shortcuts: [{
      id: `${scope}.cancel`,
      key: "Escape",
      label: "Cancel",
      showInHelp: false,
      allowInEditable: true,
      run: onCancel,
    }],
  });

  return createPortal(
    <div className="fixed inset-0 z-[60] flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/30" onClick={onCancel} aria-hidden="true" />
      <ConfirmDialogPanel id={id} {...content} />
    </div>,
    document.body,
  );
}

/** The dialog itself, without the overlay: title, body and description, then a footer bar with the buttons. */
export function ConfirmDialogPanel({ id, title, description, body, children }: ConfirmDialogContent & { id: string }) {
  return (
    <div role="alertdialog" aria-labelledby={`${id}-title`} aria-describedby={`${id}-desc`} className="relative bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-lg)] max-w-md w-full overflow-hidden">
      <div className="px-5 pt-5 pb-4">
        <h3 id={`${id}-title`} className="text-sm font-semibold text-[var(--color-text-primary)]">{title}</h3>
        {body && <div className="mt-3">{body}</div>}
        <p id={`${id}-desc`} className={cn("text-sm text-[var(--color-text-secondary)]", body ? "mt-3" : "mt-1")}>{description}</p>
      </div>
      <div className="flex items-center gap-2 px-5 py-3 border-t border-[var(--color-border-subtle)] bg-[var(--color-surface-2)]">
        <span className="flex items-center gap-1.5 text-xs text-[var(--color-text-muted)]">
          <kbd className="inline-flex items-center justify-center min-w-4 h-4 px-1 text-[10px] font-medium text-[var(--color-text-muted)] bg-[var(--color-surface-1)] border border-[var(--color-border-default)] rounded-[var(--radius-sm)]">esc</kbd>
          to cancel
        </span>
        <div className="flex-1" />
        {children}
      </div>
    </div>
  );
}
