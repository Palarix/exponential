import { useRef, useEffect } from "react";

export default function Popover({
  children,
  onClose,
}: {
  children: React.ReactNode;
  onClose: () => void;
}) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) onClose();
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [onClose]);

  return (
    <div
      ref={ref}
      className="absolute left-0 top-full mt-1 z-50 min-w-[200px] bg-[var(--color-bg-elevated)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-popover)] py-1"
    >
      {children}
    </div>
  );
}

export function PopoverHeader({ children }: { children: React.ReactNode }) {
  return (
    <div className="px-3 py-1.5 text-xs text-[var(--color-text-muted)]">
      {children}
    </div>
  );
}
