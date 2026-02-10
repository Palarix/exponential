import type { PendingState } from '../api/client';
import { Card, Button } from './ui';

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
  const eventTypeIcons: Record<string, string> = {
    CREATE: '➕',
    UPDATE: '✏️',
    COMMENT: '💬',
    DELETE: '🗑️',
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
      {/* Toggle Button (floating) */}
      <button
        onClick={onToggle}
        className={`
          fixed bottom-6 right-6 z-40
          flex items-center gap-2 px-4 py-3
          rounded-[var(--radius-lg)]
          bg-[var(--color-warning)]
          text-[var(--color-bg-primary)]
          font-medium text-sm
          shadow-lg
          hover:scale-105
          transition-all duration-[var(--duration-fast)]
          ${isOpen ? 'opacity-0 pointer-events-none' : 'animate-pulse-glow'}
        `.trim().replace(/\s+/g, ' ')}
      >
        <span className="relative flex h-2 w-2">
          <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-white opacity-75"></span>
          <span className="relative inline-flex rounded-full h-2 w-2 bg-white"></span>
        </span>
        {eventCount} pending change{eventCount !== 1 ? 's' : ''}
      </button>

      {/* Slide-out Panel */}
      <div
        className={`
          fixed top-0 right-0 bottom-0 z-50
          w-full max-w-md
          bg-[var(--color-bg-secondary)]
          border-l border-[var(--color-border-default)]
          shadow-[-4px_0_24px_rgba(0,0,0,0.3)]
          transform transition-transform duration-300 ease-out
          flex flex-col
          ${isOpen ? 'translate-x-0' : 'translate-x-full'}
        `.trim().replace(/\s+/g, ' ')}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--color-border-subtle)]">
          <div className="flex items-center gap-3">
            <div className="relative">
              <div className="w-3 h-3 rounded-full bg-[var(--color-warning)]" />
              <div className="absolute inset-0 w-3 h-3 rounded-full bg-[var(--color-warning)] animate-ping" />
            </div>
            <h2 className="text-lg font-semibold text-[var(--color-text-primary)]">
              Pending Changes
            </h2>
            <span className="px-2 py-0.5 text-xs font-medium rounded-[var(--radius-full)] bg-[var(--color-warning-bg)] text-[var(--color-warning)]">
              {eventCount}
            </span>
          </div>
          <button
            onClick={onToggle}
            className="p-2 rounded-[var(--radius-md)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Events List */}
        <div className="flex-1 overflow-y-auto p-4 space-y-3">
          {pending.events && pending.events.length > 0 ? (
            pending.events.map((event: any, i: number) => (
              <Card key={i} variant="default" padding="sm">
                <div className="flex items-start gap-3">
                  <span
                    className="text-lg flex-shrink-0"
                    title={event.type}
                  >
                    {eventTypeIcons[event.type] || '📝'}
                  </span>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <span
                        className="text-xs font-medium px-1.5 py-0.5 rounded-[var(--radius-sm)]"
                        style={{
                          background: `${eventTypeColors[event.type] || 'var(--color-text-muted)'}20`,
                          color: eventTypeColors[event.type] || 'var(--color-text-muted)'
                        }}
                      >
                        {event.type}
                      </span>
                      <span className="text-[10px] font-mono text-[var(--color-text-muted)]">
                        {event.issue_id}
                      </span>
                    </div>
                    <p className="text-xs text-[var(--color-text-muted)]">
                      {new Date(event.created_at).toLocaleTimeString()}
                    </p>
                  </div>
                </div>
              </Card>
            ))
          ) : (
            <div className="text-center py-8 text-[var(--color-text-muted)]">
              <p className="text-sm">{eventCount} change{eventCount !== 1 ? 's' : ''} pending</p>
              <p className="text-xs mt-1">Event details loading...</p>
            </div>
          )}
        </div>

        {/* Footer Actions */}
        <div className="p-4 border-t border-[var(--color-border-subtle)] bg-[var(--color-bg-tertiary)]/50 space-y-3">
          <p className="text-xs text-[var(--color-text-muted)] text-center">
            Changes are stored locally until you save
          </p>
          <div className="flex gap-3">
            <Button variant="ghost" onClick={onDiscard} className="flex-1">
              Discard All
            </Button>
            <Button onClick={onSave} className="flex-1">
              Save & Sync
            </Button>
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
