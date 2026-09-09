import type { TimelineEntry } from "../../api/types";
import { shortName, displayActor, formatShortDate } from "../../utils/format";

export type EventCategory = "issues" | "closed" | "comments" | "merges" | "artifacts" | "commits";

const TERMINAL_STATUSES = new Set(["DONE", "CANCELED", "DUPLICATE"]);

export function categorizeEntry(entry: TimelineEntry): EventCategory {
  if (entry.kind === "commit") return "commits";
  switch (entry.event_type) {
    case "COMMENT": return "comments";
    case "MERGE": return "merges";
    case "ARTIFACT": return "artifacts";
    case "UPDATE": {
      const status = entry.payload?.status;
      if (status && TERMINAL_STATUSES.has(String(status))) return "closed";
      return "issues";
    }
    default: return "issues";
  }
}

export function groupByDay(entries: TimelineEntry[]): Map<string, TimelineEntry[]> {
  const groups = new Map<string, TimelineEntry[]>();
  for (const entry of entries) {
    const day = entry.timestamp.slice(0, 10);
    const existing = groups.get(day);
    if (existing) {
      existing.push(entry);
    } else {
      groups.set(day, [entry]);
    }
  }
  return groups;
}

export function dayLabel(dateStr: string): string {
  const today = new Date().toISOString().slice(0, 10);
  const yesterday = new Date(Date.now() - 86400000).toISOString().slice(0, 10);
  if (dateStr === today) return "Today";
  if (dateStr === yesterday) return "Yesterday";
  return formatShortDate(dateStr);
}

export function daySummary(entries: TimelineEntry[]): string {
  let commits = 0;
  let events = 0;
  for (const e of entries) {
    if (e.kind === "commit") commits++;
    else events++;
  }
  const parts: string[] = [];
  if (events > 0) parts.push(`${events} event${events === 1 ? "" : "s"}`);
  if (commits > 0) parts.push(`${commits} commit${commits === 1 ? "" : "s"}`);
  return parts.join(", ");
}

export function entryActor(entry: TimelineEntry): string {
  if (entry.kind === "commit") return shortName(entry.author ?? "");
  const actor = displayActor(entry.created_by ?? "", entry.on_behalf_of);
  return actor.principal;
}

export function extractContributors(entries: TimelineEntry[]): string[] {
  const seen = new Set<string>();
  for (const e of entries) {
    const name = entryActor(e);
    if (name) seen.add(name);
  }
  return Array.from(seen).sort((a, b) => a.localeCompare(b));
}
