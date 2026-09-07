import { useState, type ReactNode } from "react";
import { Card } from "../ui";

export function Section({
  title,
  icon,
  count,
  collapsible = false,
  defaultOpen = true,
  storageKey,
  children,
}: {
  title: string;
  icon: ReactNode;
  count?: number | string;
  collapsible?: boolean;
  defaultOpen?: boolean;
  storageKey?: string;
  children: ReactNode;
}) {
  const [open, setOpen] = useState<boolean>(() => {
    if (storageKey) {
      const stored = localStorage.getItem(storageKey);
      if (stored !== null) return stored === "1";
    }
    return defaultOpen;
  });

  const toggle = () => {
    if (!collapsible) return;
    setOpen((v) => {
      const next = !v;
      if (storageKey) localStorage.setItem(storageKey, next ? "1" : "0");
      return next;
    });
  };

  return (
    <section>
      <div
        onClick={toggle}
        className={`flex items-center gap-2 px-5 py-2 bg-[var(--color-surface-1)] select-none ${collapsible ? "cursor-pointer hover:bg-[var(--color-hover-surface)] transition-colors duration-[var(--duration-fast)]" : ""}`}
      >
        {collapsible && (
          <svg
            className={`w-3 h-3 text-[var(--color-text-muted)] transition-transform duration-100 ${open ? "rotate-90" : ""}`}
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            strokeWidth={2.5}
          >
            <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
          </svg>
        )}
        <span className="text-[var(--color-text-muted)] flex shrink-0">{icon}</span>
        <span className="text-sm font-medium text-[var(--color-text-primary)]">{title}</span>
        {count !== undefined && (
          <span className="text-sm text-[var(--color-text-muted)] tabular-nums">{count}</span>
        )}
      </div>
      {(!collapsible || open) && <div>{children}</div>}
    </section>
  );
}

export function SectionIcon({ d }: { d: string }) {
  return (
    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.75}>
      <path strokeLinecap="round" strokeLinejoin="round" d={d} />
    </svg>
  );
}

export function PulseCard({ title, children }: { title: string; children: ReactNode }) {
  return (
    <Card variant="elevated" className="min-h-28 px-4 py-3 flex flex-col">
      <p className="text-xs uppercase tracking-wider text-[var(--color-text-muted)] mb-3">{title}</p>
      <div className="flex-1 flex flex-col justify-center">{children}</div>
    </Card>
  );
}

