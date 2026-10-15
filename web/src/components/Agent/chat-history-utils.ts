import type { ChatMeta } from "../../api/chat";

/** The chats not archived, in the order given. */
export function activeChats(chats: ChatMeta[]): ChatMeta[] {
  return chats.filter((c) => !c.archived_at);
}

/** How far back the history page looks, by updated_at. */
export type HistoryPeriod = "7d" | "30d" | "all";

const PERIOD_MS: Record<HistoryPeriod, number> = { "7d": 7 * 86_400_000, "30d": 30 * 86_400_000, all: Infinity };

export interface HistoryFilter {
  /** Matches titles, ignoring case. */
  query: string;
  /** Archived chats instead of active ones. */
  archived: boolean;
  /** A provider ID; "" is every agent. */
  agent: string;
  /** Chats that worked on this issue; "" is every chat. */
  issueId: string;
  period: HistoryPeriod;
}

/** The chats the history page shows for a filter, in the order given. */
export function filterHistory(chats: ChatMeta[], f: HistoryFilter, now: number): ChatMeta[] {
  const q = f.query.trim().toLowerCase();
  const since = now - PERIOD_MS[f.period];
  return chats.filter((c) =>
    !!c.archived_at === f.archived &&
    (!q || c.title.toLowerCase().includes(q)) &&
    (!f.agent || c.agent === f.agent) &&
    (!f.issueId || !!c.worked_issues?.includes(f.issueId)) &&
    (f.period === "all" || Date.parse(c.updated_at) >= since),
  );
}

export interface HistorySummary {
  count: number;
  tokensIn: number;
  tokensOut: number;
  /** The sum of the known costs. */
  costUSD: number;
  /** Some chat reported a cost. */
  costKnown: boolean;
  /** Some chats with turns reported no cost, so costUSD is a lower bound. */
  costPartial: boolean;
}

/** Totals over chats for the summary strip. */
export function summarize(chats: ChatMeta[]): HistorySummary {
  const s: HistorySummary = { count: chats.length, tokensIn: 0, tokensOut: 0, costUSD: 0, costKnown: false, costPartial: false };
  let unknown = false;
  for (const c of chats) {
    const u = c.usage;
    if (!u) continue;
    s.tokensIn += u.tokens_in;
    s.tokensOut += u.tokens_out;
    if (u.cost_known) {
      s.costUSD += u.cost_usd;
      s.costKnown = true;
    } else if (u.turns > 0) {
      unknown = true;
    }
  }
  s.costPartial = s.costKnown && unknown;
  return s;
}

/** A cost as "$1.23", or "$0.0042" under a cent; "—" when unknown. A partial total ends in "+". */
export function formatCost(usd: number, known: boolean, partial = false): string {
  if (!known) return "—";
  const text = usd > 0 && usd < 0.01 ? `$${usd.toFixed(4)}` : `$${usd.toFixed(2)}`;
  return partial ? `${text}+` : text;
}

/** A token count as "950", "12.3K" or "2.5M". */
export function formatTokens(n: number): string {
  if (n >= 1_000_000) return `${Math.round(n / 100_000) / 10}M`;
  if (n >= 1000) return `${Math.round(n / 100) / 10}K`;
  return String(n);
}

/** Why a chat cannot be archived, or null when it can. The server enforces the same rules. */
export function archiveBlocker(c: ChatMeta): string | null {
  if (c.state === "running") return "Stop the running reply before archiving";
  if (c.mode === "work" && c.work) return `Leave Work Mode on ${c.work.issue_id} before archiving`;
  return null;
}
