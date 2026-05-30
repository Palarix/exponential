import { useRef, useEffect, useState, useLayoutEffect } from "react";
import { createPortal } from "react-dom";

export default function Popover({
  children,
  onClose,
}: {
  children: React.ReactNode;
  onClose: () => void;
}) {
  const markerRef = useRef<HTMLSpanElement>(null);
  const popRef = useRef<HTMLDivElement>(null);
  const [style, setStyle] = useState<React.CSSProperties>({
    position: "fixed",
    visibility: "hidden",
  });

  useLayoutEffect(() => {
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

    setStyle({ position: "fixed", top, left, visibility: "visible" });
  }, []);

  useEffect(() => {
    const handleClick = (e: MouseEvent) => {
      if (popRef.current && !popRef.current.contains(e.target as Node)) onClose();
    };
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("mousedown", handleClick);
    document.addEventListener("keydown", handleKey);
    return () => {
      document.removeEventListener("mousedown", handleClick);
      document.removeEventListener("keydown", handleKey);
    };
  }, [onClose]);

  return (
    <>
      <span ref={markerRef} className="hidden" />
      {createPortal(
        <div
          ref={popRef}
          style={style}
          className="z-50 min-w-[200px] bg-[var(--color-bg-elevated)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-popover)] py-1"
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
