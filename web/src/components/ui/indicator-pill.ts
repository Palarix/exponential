/**
 * The pill every row indicator sits in (branch, chat, attachments, sub-issue
 * progress), after Linear: fully rounded, 24px tall and at least 32px
 * wide, a faint border, a muted 12px icon.
 * Labels keep their own rectangular chip.
 */
export const indicatorPill =
  "inline-flex items-center justify-center gap-1 h-6 min-w-8 px-2 rounded-full border border-[var(--color-border-subtle)] bg-[var(--color-surface)] text-xs text-[var(--color-text-muted)] tabular-nums shrink-0";

/**
 * The indicator pill as a control: raised off the row instead of inset into
 * it, so a pill that does something on click (the release progress) reads
 * apart from the static counts.
 */
export const raisedIndicatorPill =
  "inline-flex items-center justify-center gap-1 h-6 min-w-8 px-2 rounded-full border border-[var(--color-border-elevated)] bg-[var(--color-surface-2)] text-xs text-[var(--color-text-muted)] tabular-nums shrink-0 hover:border-[var(--color-border-overlay)] hover:text-[var(--color-text-primary)] transition-colors";
