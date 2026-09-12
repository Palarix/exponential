export interface HeadingOptions {
  leading?: 'snug';
  truncate?: boolean;
  clamp?: number;
}

export function headingClass(opts: HeadingOptions = {}): string {
  let cls = 'text-sm font-medium text-[var(--color-text-primary)]';
  if (opts.leading === 'snug') cls += ' leading-snug';
  if (opts.truncate) cls += ' min-w-0 truncate';
  if (opts.clamp) cls += ` line-clamp-${opts.clamp}`;
  return cls;
}

export function headingWrapperClass(): string {
  return 'flex items-center gap-3';
}
