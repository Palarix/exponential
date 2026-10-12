import { useId, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { useKeyboardShortcuts } from "../../keyboard";

interface ConfirmDialogProps {
  /** Unique per call site; names the Escape shortcut scope. */
  scope: string;
  title: string;
  description: ReactNode;
  onCancel: () => void;
  /** The buttons, right-aligned in order. */
  children: ReactNode;
}

/** A small alertdialog over everything else. Escape and a backdrop click cancel it. */
export default function ConfirmDialog({ scope, title, description, onCancel, children }: ConfirmDialogProps) {
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
      <div role="alertdialog" aria-labelledby={`${id}-title`} aria-describedby={`${id}-desc`} className="relative bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-lg)] p-5 max-w-sm w-full">
        <h3 id={`${id}-title`} className="text-sm font-semibold text-[var(--color-text-primary)]">{title}</h3>
        <p id={`${id}-desc`} className="text-sm text-[var(--color-text-secondary)] mt-3">{description}</p>
        <div className="flex items-center justify-end gap-2 mt-4">{children}</div>
      </div>
    </div>,
    document.body,
  );
}
