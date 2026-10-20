import type { NavIntent } from "../../app/contexts";
import type { BacklogFilters } from "./filters";
import type { Tab } from "./Backlog";

/**
 * The tab a navigation intent opens (its own, or the open one) and the
 * Issues filters with the intent's filters merged in; null when the intent
 * has no filters. The tabs share one filter set.
 */
export function intentTarget(
  intent: NavIntent,
  activeTab: Tab,
  filters: BacklogFilters,
): { tab: Tab; filters: BacklogFilters | null } {
  return { tab: intent.tab ?? activeTab, filters: intent.filters ? { ...filters, ...intent.filters } : null };
}
