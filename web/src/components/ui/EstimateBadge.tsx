import { Triangle, TriangleDashed } from "lucide-react";

interface EstimateBadgeProps {
  value: number;
}

export default function EstimateBadge({ value }: EstimateBadgeProps) {
  const hasEstimate = value > 0;
  const display = value || 1;
  const Icon = hasEstimate ? Triangle : TriangleDashed;

  return (
    <div className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[var(--color-text-muted)] hover:bg-[var(--color-bg-tertiary)] transition-colors">
      <Icon size={13} />
      <span className="text-sm tabular-nums">{display}</span>
    </div>
  );
}
