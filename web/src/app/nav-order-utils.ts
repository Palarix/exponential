/**
 * The order IssueDetail's prev/next walks: the order published by the view the
 * issue was opened from, or the default sorted order when that view published
 * none or the issue isn't in it (deep links, views without a list).
 */
export function resolveNavOrder(
  published: string[] | null,
  fallback: string[],
  issueId: string | null,
): string[] {
  if (!published || published.length === 0) return fallback;
  if (issueId && !published.includes(issueId)) return fallback;
  return published;
}
