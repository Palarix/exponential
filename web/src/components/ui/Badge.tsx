import type { ReactNode } from 'react';

type BadgeVariant = 'default' | 'success' | 'warning' | 'error' | 'info' |
  'feature' | 'bug' | 'epic' |
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

  // Label badges
  feature: 'bg-blue-300 text-blue-900 dark:bg-blue-900/30 dark:text-blue-300',
  bug: 'bg-red-300 text-red-900 dark:bg-red-900/30 dark:text-red-300',
  epic: 'bg-purple-300 text-purple-900 dark:bg-purple-900/30 dark:text-purple-300',

  // Status badges
  backlog: 'bg-[oklch(0.25_0.04_260)] text-[var(--color-status-backlog)]',
  planned: 'bg-[oklch(0.28_0.08_250)] text-[var(--color-status-planned)]',
  doing: 'bg-[oklch(0.32_0.08_85)] text-[var(--color-status-doing)]',
  blocked: 'bg-[oklch(0.28_0.08_25)] text-[var(--color-status-blocked)]',
  done: 'bg-[oklch(0.28_0.08_145)] text-[var(--color-status-done)]',
};

const sizeStyles: Record<string, string> = {
  sm: 'px-2 py-0.5 text-[10px]',
  md: 'px-2 py-1 text-xs',
};

export default function Badge({
  children,
  variant = 'default',
  size = 'md',
  dot = false
}: BadgeProps) {
  return (
    <span
      className={`
        inline-flex items-center gap-1
        font-medium
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

export function LabelBadge({ label }: { label: string }) {
  let variant: BadgeVariant = 'default';
  const lowerLabel = label.toLowerCase();
  const upperLabel = label.toUpperCase()

  if (lowerLabel === 'feature') variant = 'feature';
  if (lowerLabel === 'bug') variant = 'bug';
  if (lowerLabel === 'epic') variant = 'epic';

  return <Badge size="sm" variant={variant}>{upperLabel}</Badge>;
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
