import type { Release } from "../../api/types";
import { ReleaseSummaryCard, ReleaseTitle } from "./ReleaseRow";

/**
 * A minimal release page: name, goal and the same breakdown as the progress
 * popover. The full detail page (xpo-e6ff99) replaces it.
 */
export default function ReleaseDetail({ number, release, today, onBack }: {
  number: number;
  release: Release | null;
  today: string;
  onBack: () => void;
}) {
  return (
    <div className="h-full scroll-stable">
      <div className="max-w-3xl mx-auto px-6 pt-5 pb-8">
        <button
          onClick={onBack}
          className="flex items-center gap-1 text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors mb-3"
        >
          <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 19.5L8.25 12l7.5-7.5" />
          </svg>
          Releases
        </button>
        {release ? (
          <>
            <ReleaseTitle release={release} />
            {release.placeholder ? (
              <p className="mt-4 text-sm text-[var(--color-text-muted)]">Not saved yet.</p>
            ) : (
              <div className="mt-5 max-w-sm rounded-[var(--radius-lg)] border border-[var(--color-border-subtle)] bg-[var(--color-surface-1)]">
                <ReleaseSummaryCard release={release} today={today} />
              </div>
            )}
          </>
        ) : (
          <p className="py-12 text-center text-sm text-[var(--color-text-muted)]">Release #{number} not found.</p>
        )}
      </div>
    </div>
  );
}
