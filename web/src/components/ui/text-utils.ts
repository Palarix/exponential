export type TextColor = 'primary' | 'secondary' | 'muted' | 'success' | 'warning' | 'error' | 'accent';
export type TextSize = 'xs' | 'regular' | 'base' | 'lg';
export type TextWeight = 'regular' | 'medium' | 'semibold' | 'bold';

const COLOR_MAP: Record<TextColor, string> = {
  primary: 'text-[var(--color-text-primary)]',
  secondary: 'text-[var(--color-text-secondary)]',
  muted: 'text-[var(--color-text-muted)]',
  success: 'text-[var(--color-success)]',
  warning: 'text-[var(--color-warning)]',
  error: 'text-[var(--color-error)]',
  accent: 'text-[var(--color-accent-primary)]',
};

const SIZE_MAP: Record<TextSize, string> = {
  xs: 'text-xs',
  regular: 'text-sm',
  base: 'text-base',
  lg: 'text-lg',
};

const WEIGHT_MAP: Record<TextWeight, string> = {
  regular: 'font-normal',
  medium: 'font-medium',
  semibold: 'font-semibold',
  bold: 'font-bold',
};

export interface TextOptions {
  color?: TextColor;
  size?: TextSize;
  weight?: TextWeight;
  mono?: boolean;
  tabular?: boolean;
  truncate?: boolean;
}

export function textClass(opts: TextOptions = {}): string {
  const parts = [
    SIZE_MAP[opts.size ?? 'regular'],
    COLOR_MAP[opts.color ?? 'muted'],
  ];
  const w = WEIGHT_MAP[opts.weight ?? 'regular'];
  if (w) parts.push(w);
  if (opts.mono) parts.push('font-mono');
  if (opts.tabular) parts.push('tabular-nums');
  if (opts.truncate) parts.push('min-w-0 truncate');
  return parts.join(' ');
}
