import { CircleCheck } from "lucide-react";

const R = 10;
const CIRCUMFERENCE = 2 * Math.PI * R;

export default function SubProgress({ done, total }: { done: number; total: number }) {
  if (total > 0 && done >= total) {
    return <CircleCheck size={14} strokeWidth={1.5} className="shrink-0 text-[var(--color-success)]" />;
  }

  const pct = total > 0 ? done / total : 0;

  return (
    <svg width="14" height="14" viewBox="0 0 24 24" className="shrink-0" fill="none">
      <circle
        cx="12"
        cy="12"
        r={R}
        stroke="var(--color-text-muted)"
        strokeWidth="2.5"
        strokeDasharray="3 3"
        opacity={0.4}
      />
      {pct > 0 && (
        <circle
          cx="12"
          cy="12"
          r={R}
          stroke="var(--color-accent-primary)"
          strokeWidth="2.5"
          strokeLinecap="round"
          strokeDasharray={`${pct * CIRCUMFERENCE} ${CIRCUMFERENCE}`}
          transform="rotate(-90 12 12)"
        />
      )}
    </svg>
  );
}
