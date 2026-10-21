import { useId, type ReactNode } from "react";
import { Dialog, DialogOverlay } from "./Dialog";

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
  return (
    <DialogOverlay scope={scope} onClose={onCancel}>
      <ConfirmDialogPanel id={id} {...content} />
    </DialogOverlay>
  );
}

/** The dialog itself, without the overlay: title, body and description, then a footer with the buttons. */
export function ConfirmDialogPanel({ id, title, description, body, children }: ConfirmDialogContent & { id: string }) {
  return (
    <Dialog id={id} role="alertdialog" title={title} describedBy={`${id}-desc`} footer={children}>
      {body && <div className="mb-3">{body}</div>}
      <p id={`${id}-desc`} className="text-sm text-[var(--color-text-secondary)]">{description}</p>
    </Dialog>
  );
}
