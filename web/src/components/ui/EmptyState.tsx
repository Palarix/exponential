interface EmptyStateProps {
  title: string;
  description: string;
  icon: React.ReactNode;
  actionLabel?: string;
  onAction?: () => void;
}

export default function EmptyState({ title, description, icon, actionLabel, onAction }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center h-full gap-6 px-8">
      <div className="opacity-25">
        {icon}
      </div>

      <div className="flex flex-col items-center gap-2 max-w-96 text-center">
        <h2 className="text-lg font-semibold text-[var(--color-text-primary)]">{title}</h2>
        <p className="text-sm text-[var(--color-text-secondary)] leading-relaxed">{description}</p>
      </div>

      {actionLabel && onAction && (
        <button
          onClick={onAction}
          className="px-4 py-2 text-sm bg-[var(--color-accent-primary)] text-white rounded-[var(--radius-md)] hover:opacity-90 transition-opacity"
        >
          {actionLabel}
        </button>
      )}

      <p className="text-xs text-[var(--color-text-muted)]">
        Press <kbd className="px-1.5 py-0.5 text-xs font-mono bg-[var(--color-surface-1)] border border-[var(--color-border-default)] rounded">C</kbd> anytime to create an issue
      </p>
    </div>
  );
}
