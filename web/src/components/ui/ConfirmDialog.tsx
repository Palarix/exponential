import { useId, type ReactNode } from "react";
import { cn } from "../../utils/cn";
import { DialogOverlay, DialogPanel } from "./Dialog";

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

/** The dialog itself, without the overlay: title, body and description, then a footer bar with the buttons. */
export function ConfirmDialogPanel({ id, title, description, body, children }: ConfirmDialogContent & { id: string }) {
  return (
    <DialogPanel id={id} role="alertdialog" title={title} describedBy={`${id}-desc`} footer={children}>
      {body && <div className="mt-3">{body}</div>}
      <p id={`${id}-desc`} className={cn("text-sm text-[var(--color-text-secondary)]", body ? "mt-3" : "mt-1")}>{description}</p>
    </DialogPanel>
  );
}
