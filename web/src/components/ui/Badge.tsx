import type { ReactNode } from 'react';

type BadgeVariant = 'default' | 'success' | 'warning' | 'error' | 'info' | 'epic' | 'task' | 'bug' |
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
  // Kind badges
  epic: 'bg-[oklch(0.30_0.08_300)] text-[var(--color-kind-epic)]',
  task: 'bg-[oklch(0.30_0.08_250)] text-[var(--color-kind-task)]',
  bug: 'bg-[oklch(0.30_0.08_25)] text-[var(--color-kind-bug)]',
  // Status badges
  backlog: 'bg-[oklch(0.25_0.04_260)] text-[var(--color-status-backlog)]',
  planned: 'bg-[oklch(0.28_0.08_250)] text-[var(--color-status-planned)]',
  doing: 'bg-[oklch(0.32_0.08_85)] text-[var(--color-status-doing)]',
  blocked: 'bg-[oklch(0.28_0.08_25)] text-[var(--color-status-blocked)]',
  done: 'bg-[oklch(0.28_0.08_145)] text-[var(--color-status-done)]',
};

const sizeStyles: Record<string, string> = {
  sm: 'px-1.5 py-0.5 text-[10px]',
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

// Convenience components for common badge types
export function KindBadge({ kind }: { kind: string }) {
  const variant = kind.toLowerCase() as BadgeVariant;
  return <Badge variant={['epic', 'task', 'bug'].includes(variant) ? variant : 'default'}>{kind}</Badge>;
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
