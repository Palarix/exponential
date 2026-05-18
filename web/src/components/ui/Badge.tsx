import type { ReactNode } from 'react';

type BadgeVariant = 'default' | 'success' | 'warning' | 'error' | 'info' |
  'backlog' | 'planned' | 'doing' | 'blocked' | 'done';

interface BadgeProps {
  children: ReactNode;
  variant?: BadgeVariant;
  size?: 'sm' | 'md';
  dot?: boolean;
}

const variantStyles: Record<BadgeVariant, string> = {
  default: 'bg-[var(--color-bg-tertiary)] text-[var(--color-text-secondary)]',
  success: 'bg-[var(--color-success-bg)] text-[var(--color-success)]',
  warning: 'bg-[var(--color-warning-bg)] text-[var(--color-warning)]',
  error: 'bg-[var(--color-error-bg)] text-[var(--color-error)]',
  info: 'bg-[var(--color-info-bg)] text-[var(--color-info)]',
  backlog: 'bg-[var(--color-bg-tertiary)] text-[var(--color-status-backlog)]',
  planned: 'bg-[var(--color-bg-tertiary)] text-[var(--color-status-planned)]',
  doing: 'bg-[var(--color-warning-bg)] text-[var(--color-status-doing)]',
  blocked: 'bg-[var(--color-error-bg)] text-[var(--color-status-blocked)]',
  done: 'bg-[var(--color-success-bg)] text-[var(--color-status-done)]',
};

const sizeStyles: Record<string, string> = {
  sm: 'px-1.5 py-0.5 text-[10px]',
  md: 'px-2 py-0.5 text-[11px]',
};

export default function Badge({
  children,
  variant = 'default',
  size = 'md',
  dot = false,
}: BadgeProps) {
  return (
    <span
      className={`
        inline-flex items-center gap-1 font-medium
        rounded-[var(--radius-sm)]
        ${variantStyles[variant]}
        ${sizeStyles[size]}
      `.trim().replace(/\s+/g, ' ')}
    >
      {dot && (
        <span className="w-1.5 h-1.5 rounded-full bg-current opacity-80" />
      )}
      {children}
    </span>
  );
}

const LABEL_COLORS: Record<string, string> = {
  bug: 'var(--color-label-bug)',
  feature: 'var(--color-label-feature)',
  epic: 'var(--color-label-epic)',
  improvement: 'var(--color-label-improvement)',
};

export function LabelBadge({ label }: { label: string }) {
  const color = LABEL_COLORS[label.toLowerCase()] || 'var(--color-text-muted)';
  const displayLabel = label.charAt(0).toUpperCase() + label.slice(1).toLowerCase();

  return (
    <span
      className="inline-flex items-center gap-1.5 text-[11px] font-medium px-2 py-0.5 rounded-full"
      style={{ background: `color-mix(in srgb, ${color} 15%, transparent)`, color }}
    >
      <span
        className="w-[6px] h-[6px] rounded-full shrink-0"
        style={{ background: color }}
      />
      {displayLabel}
    </span>
  );
}

export function StatusBadge({ status }: { status: string }) {
  const variant = status.toLowerCase() as BadgeVariant;
  return (
    <Badge
      variant={['backlog', 'planned', 'doing', 'blocked', 'done'].includes(variant) ? variant : 'default'}
      dot
    >
      {status}
    </Badge>
  );
}
