import { type ReactNode, useState } from "react";
import { Card } from "../ui";

export type DistFilter = "all" | "active";

export type DistRow = {
  key: string;
  label: ReactNode;
  count: number;
  pct: number;
  barWidth: number;
};

export default function DistributionSection({
  title,
  filter,
  onFilterChange,
  rows,
  total,
  emptyHasIssues = "No data to display.",
  emptyNoIssues = "No issues to display.",
  hideBars = false,
}: {
  title: string;
  filter: DistFilter;
  onFilterChange: (f: DistFilter) => void;
  rows: DistRow[];
  total: number;
  emptyHasIssues?: string;
  emptyNoIssues?: string;
  hideBars?: boolean;
}) {
  const [showPct, setShowPct] = useState(false);

  return (
    <div>
      <div className="flex items-center justify-between mb-3">
        <h2 className="text-xs uppercase tracking-wider text-[var(--color-text-muted)]">{title}</h2>
        <div className="flex items-center gap-1 bg-[var(--color-bg-tertiary)] rounded-[var(--radius-md)] p-1">
          {(["all", "active"] as const).map((opt) => (
            <button
              key={opt}
              onClick={() => onFilterChange(opt)}
              className={`px-3 h-6 rounded-[var(--radius-sm)] text-xs transition-colors ${
                filter === opt
                  ? "bg-[var(--color-bg-elevated)] text-[var(--color-text-primary)]"
                  : "text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)]"
              }`}
            >
              {opt === "active" ? "Active" : "All"}
            </button>
          ))}
        </div>
      </div>
      <Card variant="elevated" className="flex flex-col">
        {rows.length === 0 ? (
          <div className="h-40 flex items-center justify-center">
            <p className="text-sm text-[var(--color-text-muted)] text-center">
              {total === 0 ? emptyNoIssues : emptyHasIssues}
            </p>
          </div>
        ) : (
          <div className="space-y-2 h-40 overflow-y-auto overscroll-contain pr-2">
            {rows.map(({ key, label, count, pct, barWidth }) => (
              <div key={key} className="flex items-center gap-2 h-6">
                <div className="flex-1 min-w-0">{label}</div>
                {!hideBars && (
                  <div className="w-16 h-2 rounded-full bg-[var(--color-bg-tertiary)] overflow-hidden shrink-0">
                    <div
                      className="h-full rounded-full bg-[var(--color-text-secondary)] transition-all duration-500"
                      style={{ width: `${barWidth}%` }}
                    />
                  </div>
                )}
                <button
                  onClick={() => setShowPct((v) => !v)}
                  className="w-10 shrink-0 text-right text-xs tabular-nums text-[var(--color-text-primary)] font-medium hover:text-[var(--color-accent-primary)] transition-colors"
                  title="Click to toggle count/percentage"
                >
                  {showPct ? `${pct.toFixed(0)}%` : count}
                </button>
              </div>
            ))}
          </div>
        )}
      </Card>
    </div>
  );
}
