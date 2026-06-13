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
        className={`flex items-center gap-2 px-5 py-2 bg-[var(--color-bg-secondary)] select-none ${collapsible ? "cursor-pointer hover:bg-[var(--color-bg-hover)] transition-colors duration-[var(--duration-fast)]" : ""}`}
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

export const SECTION_ICONS = {
  pulse: "M3 12h3l3-9 6 18 3-9h3",
  workload: "M17 20h5v-2a4 4 0 00-3-3.87M9 20H2v-2a4 4 0 014-4h4a4 4 0 014 4v2M16 3.13a4 4 0 010 7.75M9 12a4 4 0 100-8 4 4 0 000 8z",
  attention: "M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z",
  epics: "M17.593 3.322c1.1.128 1.907 1.077 1.907 2.185V21L12 17.25 4.5 21V5.507c0-1.108.806-2.057 1.907-2.185a48.507 48.507 0 0111.186 0z",
  activity: "M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z",
  composition: "M2.25 7.125C2.25 6.504 2.754 6 3.375 6h6c.621 0 1.125.504 1.125 1.125v3.75c0 .621-.504 1.125-1.125 1.125h-6a1.125 1.125 0 01-1.125-1.125v-3.75zM14.25 8.625c0-.621.504-1.125 1.125-1.125h5.25c.621 0 1.125.504 1.125 1.125v8.25c0 .621-.504 1.125-1.125 1.125h-5.25a1.125 1.125 0 01-1.125-1.125v-8.25zM3.75 16.125c0-.621.504-1.125 1.125-1.125h5.25c.621 0 1.125.504 1.125 1.125v2.25c0 .621-.504 1.125-1.125 1.125h-5.25a1.125 1.125 0 01-1.125-1.125v-2.25z",
  trends: "M2.25 18L9 11.25l4.306 4.306a11.95 11.95 0 015.814-5.518l2.74-1.22m0 0l-5.94-2.281m5.94 2.281l-2.28 5.941",
};
