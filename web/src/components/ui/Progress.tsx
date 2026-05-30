interface ProgressProps {
  value: number;
  max?: number;
  variant?: 'default' | 'success' | 'warning' | 'accent';
  size?: 'sm' | 'md' | 'lg';
  showLabel?: boolean;
  label?: string;
}

const variantColors: Record<string, string> = {
  default: 'bg-[var(--color-text-muted)]',
  success: 'bg-[var(--color-success)]',
  warning: 'bg-[var(--color-warning)]',
  accent: 'bg-[var(--color-accent-primary)]',
};

const sizeStyles: Record<string, string> = {
  sm: 'h-1',
  md: 'h-1',
  lg: 'h-2',
};

export default function Progress({
  value,
  max = 100,
  variant = 'accent',
  size = 'md',
  showLabel = false,
  label,
}: ProgressProps) {
  const percentage = Math.min(100, Math.max(0, (value / max) * 100));

  return (
    <div className="w-full">
      {(showLabel || label) && (
        <div className="flex justify-between items-center mb-2">
          {label && (
            <span className="text-xs text-[var(--color-text-secondary)]">{label}</span>
          )}
          {showLabel && (
            <span className="text-xs text-[var(--color-text-muted)]">
              {Math.round(percentage)}%
            </span>
          )}
        </div>
      )}
      <div
        className={`w-full bg-[var(--color-bg-tertiary)] rounded-full overflow-hidden ${sizeStyles[size]}`}
        role="progressbar"
        aria-valuenow={value}
        aria-valuemin={0}
        aria-valuemax={max}
      >
        <div
          className={`h-full rounded-full transition-all duration-500 ease-out ${variantColors[variant]}`}
          style={{ width: `${percentage}%` }}
        />
      </div>
    </div>
  );
}

interface ProgressRingProps {
  value: number;
  max?: number;
  size?: number;
  strokeWidth?: number;
  color?: string;
  label?: string;
  sublabel?: string;
}

export function ProgressRing({
  value,
  max = 100,
  size = 80,
  strokeWidth = 6,
  color = 'var(--color-accent-primary)',
  label,
  sublabel,
}: ProgressRingProps) {
  const percentage = Math.min(100, Math.max(0, (value / max) * 100));
  const radius = (size - strokeWidth) / 2;
  const circumference = radius * 2 * Math.PI;
  const offset = circumference - (percentage / 100) * circumference;

  return (
    <div className="relative inline-flex items-center justify-center">
      <svg width={size} height={size} className="-rotate-90">
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          stroke="var(--color-bg-tertiary)"
          strokeWidth={strokeWidth}
        />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          stroke={color}
          strokeWidth={strokeWidth}
          strokeLinecap="round"
          strokeDasharray={circumference}
          strokeDashoffset={offset}
          className="transition-all duration-700 ease-out"
        />
      </svg>
      {(label || sublabel) && (
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          {label && (
            <span className="text-lg font-bold text-[var(--color-text-primary)]">{label}</span>
          )}
          {sublabel && (
            <span className="text-xs text-[var(--color-text-muted)]">{sublabel}</span>
          )}
        </div>
      )}
    </div>
  );
}
