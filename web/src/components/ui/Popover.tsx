import { useRef, useEffect, useLayoutEffect } from "react";
import { createPortal } from "react-dom";
import { useKeyboardShortcuts } from "../../keyboard";

export default function Popover({
  children,
  onClose,
}: {
  children: React.ReactNode;
  onClose: () => void;
}) {
  const markerRef = useRef<HTMLSpanElement>(null);
  const popRef = useRef<HTMLDivElement>(null);

  const positionPopover = () => {
    const marker = markerRef.current;
    const pop = popRef.current;
    if (!marker || !pop) return;

    const anchor = marker.parentElement;
    if (!anchor) return;

    const aRect = anchor.getBoundingClientRect();
    const popRect = pop.getBoundingClientRect();
    const pad = 8;

    let top = aRect.bottom + 4;
    if (top + popRect.height > window.innerHeight - pad) {
      top = aRect.top - popRect.height - 4;
    }
    let left = aRect.left;
    if (left + popRect.width > window.innerWidth - pad) {
      left = window.innerWidth - popRect.width - pad;
    }

    pop.style.top = `${top}px`;
    pop.style.left = `${left}px`;
    pop.style.visibility = "visible";
  };

  useLayoutEffect(() => {
    positionPopover();
  }, []);

  useKeyboardShortcuts({
    scope: "popover",
    priority: "overlay",
    shortcuts: [{ id: "popover.close", key: "Escape", label: "Close popover", showInHelp: false, allowInEditable: true, run: onClose }],
  });

  useEffect(() => {
    const handleClick = (e: MouseEvent) => {
      if (popRef.current && !popRef.current.contains(e.target as Node)) onClose();
    };
    document.addEventListener("mousedown", handleClick);
    return () => {
      document.removeEventListener("mousedown", handleClick);
    };
  }, [onClose]);

  return (
    <>
      <span ref={markerRef} className="hidden" />
      {createPortal(
        <div
          ref={popRef}
          style={{ position: "fixed", visibility: "hidden" }}
          className="z-50 min-w-50 bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-popover)] py-1"
        >
          {children}
        </div>,
        document.body,
      )}
    </>
  );
}

export function PopoverHeader({ children }: { children: React.ReactNode }) {
  return (
    <div className="px-3 py-2 text-xs text-[var(--color-text-muted)]">
      {children}
    </div>
  );
}
