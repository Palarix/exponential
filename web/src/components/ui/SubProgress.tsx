export default function SubProgress({ done, total }: { done: number; total: number }) {
  const pct = total > 0 ? (done / total) * 100 : 0;
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" className="shrink-0">
      <circle
        cx="8"
        cy="8"
        r="6"
        fill="none"
        stroke="var(--color-bg-tertiary)"
        strokeWidth="2"
      />
      <circle
        cx="8"
        cy="8"
        r="6"
        fill="none"
        stroke={pct === 100 ? "var(--color-success)" : "var(--color-accent-primary)"}
        strokeWidth="2"
        strokeLinecap="round"
        strokeDasharray={`${pct * 0.377} 100`}
        transform="rotate(-90 8 8)"
      />
    </svg>
  );
}
