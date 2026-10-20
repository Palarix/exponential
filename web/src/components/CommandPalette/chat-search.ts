import type { Issue } from "../../api/client";
import type { ChatMeta } from "../../api/chat";
import { fuzzyScore } from "../../utils/format";

const RECENT = 5;
const LIMIT = 8;
// The score of an issue ID match in the palette's issue search.
const ID_SCORE = 6000;

export interface ChatMatch {
  chat: ChatMeta;
  /** The chat's linked issue: its Work Mode issue, else the last issue it worked on. */
  issueId?: string;
}

function linkedIssue(c: ChatMeta): string | undefined {
  if (c.mode === "work" && c.work) return c.work.issue_id;
  return c.worked_issues?.at(-1);
}

/**
 * The chats for the command palette. Without a query, the newest few. With
 * one, the chats whose title, worked issue IDs or linked issue title match,
 * best first; equal scores keep the chat list's newest-first order.
 */
export function searchChats(chats: ChatMeta[], issues: Issue[], query: string): ChatMatch[] {
  const q = query.trim().toLowerCase();
  if (!q) return chats.slice(0, RECENT).map((chat) => ({ chat, issueId: linkedIssue(chat) }));

  const titles = new Map(issues.map((i) => [i.id, i.title]));
  return chats
    .map((chat) => {
      const issueId = linkedIssue(chat);
      const ids = [...(chat.worked_issues ?? []), ...(chat.work ? [chat.work.issue_id] : [])];
      const issueTitle = issueId ? titles.get(issueId) : undefined;
      const score = ids.some((id) => id.includes(q))
        ? ID_SCORE
        : Math.max(fuzzyScore(chat.title, q), issueTitle ? fuzzyScore(issueTitle, q) : 0);
      return { match: { chat, issueId }, score };
    })
    .filter((r) => r.score > 0)
    .sort((a, b) => b.score - a.score)
    .slice(0, LIMIT)
    .map((r) => r.match);
}
