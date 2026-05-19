import type { PendingState } from '../api/client';

interface PendingEventsPanelProps {
  pending: PendingState;
  isOpen: boolean;
  onToggle: () => void;
  onSave: () => void;
  onDiscard: () => void;
}

export default function PendingEventsPanel({
  pending,
  isOpen,
  onToggle,
  onSave,
  onDiscard,
}: PendingEventsPanelProps) {
  const eventTypeLabels: Record<string, string> = {
    CREATE: 'Created',
    UPDATE: 'Updated',
    COMMENT: 'Commented',
    DELETE: 'Deleted',
  };

  const eventTypeColors: Record<string, string> = {
    CREATE: 'var(--color-success)',
    UPDATE: 'var(--color-info)',
    COMMENT: 'var(--color-accent-primary)',
    DELETE: 'var(--color-error)',
  };

  const eventCount = pending.events?.length || 0;
  if (eventCount === 0) return null;

  return (
    <>
      {/* Slide-out Panel */}
      <div
        className={`
          fixed top-0 right-0 bottom-0 z-50
          w-full max-w-sm
          bg-[var(--color-bg-secondary)]
          border-l border-[var(--color-border-subtle)]
          shadow-[var(--shadow-lg)]
          transform transition-transform duration-200 ease-out
          flex flex-col
          ${isOpen ? 'translate-x-0' : 'translate-x-full'}
        `.trim().replace(/\s+/g, ' ')}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-4 h-11 border-b border-[var(--color-border-subtle)]">
          <div className="flex items-center gap-2">
            <span className="w-1.5 h-1.5 rounded-full bg-[var(--color-warning)]" />
            <h2 className="text-sm font-medium text-[var(--color-text-primary)]">
              Pending Changes
            </h2>
            <span className="text-xs text-[var(--color-text-muted)] tabular-nums">
              {eventCount}
            </span>
          </div>
          <button
            onClick={onToggle}
            className="p-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Events List */}
        <div className="flex-1 overflow-y-auto">
          {pending.events && pending.events.length > 0 ? (
            pending.events.map((event, i) => (
              <div
                key={i}
                className="flex items-center gap-3 px-4 py-2.5 border-b border-[var(--color-border-subtle)]"
              >
                <span
                  className="w-1.5 h-1.5 rounded-full shrink-0"
                  style={{ background: eventTypeColors[event.type] || 'var(--color-text-muted)' }}
                />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium text-[var(--color-text-secondary)]">
                      {eventTypeLabels[event.type] || event.type}
                    </span>
                    <span className="text-xs font-mono text-[var(--color-text-muted)]">
                      {event.issue_id}
                    </span>
                  </div>
                  <p className="text-xs text-[var(--color-text-muted)]">
                    {new Date(event.created_at).toLocaleTimeString()}
                  </p>
                </div>
              </div>
            ))
          ) : (
            <div className="text-center py-8 text-[var(--color-text-muted)] text-sm">
              {eventCount} change{eventCount !== 1 ? 's' : ''} pending
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="p-3 border-t border-[var(--color-border-subtle)] space-y-2">
          <p className="text-xs text-[var(--color-text-muted)] text-center">
            Changes are stored locally until you save
          </p>
          <div className="flex gap-2">
            <button
              onClick={onDiscard}
              className="flex-1 px-3 py-1.5 text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] rounded-[var(--radius-md)] transition-colors"
            >
              Discard
            </button>
            <button
              onClick={onSave}
              className="flex-1 px-3 py-1.5 text-sm text-white bg-[var(--color-accent-primary)] hover:bg-[var(--color-accent-primary-hover)] rounded-[var(--radius-md)] transition-colors"
            >
              Save & Sync
            </button>
          </div>
        </div>
      </div>

      {/* Backdrop */}
      {isOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/40 animate-fade-in"
          onClick={onToggle}
        />
      )}
    </>
  );
}
