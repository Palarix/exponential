export function formatCount(count: number, label = 'issue', short = false): string {
  if (short) return String(count);
  return `${count} ${label}${count !== 1 ? 's' : ''}`;
}

export function countBadgeClass(rounded = false): string {
  const base = 'text-xs tabular-nums text-[var(--color-text-muted)]';
  if (rounded) {
    return `${base} inline-flex items-center justify-center min-w-[1.25rem] px-1.5 py-0.5 rounded-full bg-[var(--color-hover-surface)]`;
  }
  return base;
}
