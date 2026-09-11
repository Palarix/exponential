import { cn } from '../../utils/cn';

const BASE =
  'flex items-center justify-center w-7 h-7 rounded-[var(--radius-md)] bg-[var(--color-surface-1)] border border-[var(--color-border-default)] text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] transition-colors';

export function iconButtonClass(active?: boolean, className?: string): string {
  return cn(BASE, active && 'relative', className);
}
