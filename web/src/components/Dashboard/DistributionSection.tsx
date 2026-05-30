import type { ReactNode } from "react";
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
}: {
  title: string;
  filter: DistFilter;
  onFilterChange: (f: DistFilter) => void;
  rows: DistRow[];
  total: number;
  emptyHasIssues?: string;
  emptyNoIssues?: string;
}) {
  return (
    <div>
      <div className="flex items-center justify-between mb-3">
        <h2 className="text-xs uppercase tracking-wider text-[var(--color-text-muted)]">{title}</h2>
        <div className="flex items-center gap-1 bg-[var(--color-bg-tertiary)] rounded-[var(--radius-md)] p-1">
          {(["active", "all"] as const).map((opt) => (
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
      <Card variant="elevated" className="min-h-[160px] flex flex-col">
        {rows.length === 0 ? (
          <div className="flex-1 flex items-center justify-center">
            <p className="text-sm text-[var(--color-text-muted)] text-center">
              {total === 0 ? emptyNoIssues : emptyHasIssues}
            </p>
          </div>
        ) : (
          <div className="space-y-2">
            {rows.map(({ key, label, count, pct, barWidth }) => (
              <div key={key} className="flex items-center gap-2">
                <div className="flex-1 min-w-0">{label}</div>
                <div className="w-16 h-2 rounded-full bg-[var(--color-bg-tertiary)] overflow-hidden shrink-0">
                  <div
                    className="h-full rounded-full bg-[var(--color-text-secondary)] transition-all duration-500"
                    style={{ width: `${barWidth}%` }}
                  />
                </div>
                <div className="w-14 shrink-0 text-right text-xs tabular-nums">
                  <span className="text-[var(--color-text-primary)] font-medium">{count}</span>
                  <span className="text-[var(--color-text-muted)] ml-1">{pct.toFixed(0)}%</span>
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>
    </div>
  );
}
