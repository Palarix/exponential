import { useRef, useEffect, useLayoutEffect, type RefObject, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { useKeyboardShortcuts } from "../../keyboard";
import { cn } from "../../utils/cn";
import { computePopoverPosition, type Placement } from "./popover-utils";

interface PopoverProps {
  anchorRef: RefObject<HTMLElement | null>;
  onClose: () => void;
  placement?: Placement;
  offset?: number;
  className?: string;
  children: ReactNode;
}

export default function Popover({
  anchorRef,
  onClose,
  placement = "bottom-start",
  offset = 4,
  className,
  children,
}: PopoverProps) {
  const popRef = useRef<HTMLDivElement>(null);

  useLayoutEffect(() => {
    const pop = popRef.current;
    if (!pop) return;

    const position = () => {
      const anchor = anchorRef.current;
      if (!anchor) return;
      const aRect = anchor.getBoundingClientRect();
      const popRect = pop.getBoundingClientRect();
      const { top, left } = computePopoverPosition(
        aRect,
        { width: popRect.width, height: popRect.height },
        placement,
        offset,
        { width: window.innerWidth, height: window.innerHeight },
      );
      pop.style.top = `${top}px`;
      pop.style.left = `${left}px`;
      pop.style.visibility = "visible";
    };

    if (anchorRef.current) {
      position();
    } else {
      queueMicrotask(position);
    }
  });

  useKeyboardShortcuts({
    scope: "popover",
    priority: "overlay",
    shortcuts: [
      { id: "popover.close", key: "Escape", label: "Close popover", showInHelp: false, allowInEditable: true, run: onClose },
    ],
  });

  useEffect(() => {
    const handleClick = (e: MouseEvent) => {
      const target = e.target as Node;
      if (
        popRef.current && !popRef.current.contains(target) &&
        !(anchorRef.current && anchorRef.current.contains(target))
      ) {
        onClose();
      }
    };
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [onClose, anchorRef]);

  return createPortal(
    <div
      ref={popRef}
      style={{ position: "fixed", visibility: "hidden" }}
      className={cn("z-50", className)}
    >
      {children}
    </div>,
    document.body,
  );
}

export function PopoverPanel({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div className={cn(
      "min-w-50 bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-popover)] py-1",
      className,
    )}>
      {children}
    </div>
  );
}

export function PopoverHeader({ children }: { children: ReactNode }) {
  return (
    <div className="px-3 py-2 text-xs text-[var(--color-text-muted)]">
      {children}
    </div>
  );
}
