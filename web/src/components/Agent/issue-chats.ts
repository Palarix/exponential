import type { ChatMeta } from "../../api/chat";

/**
 * The state an issue's chat icon shows, most urgent first: a chat waits for
 * an approval, runs, was interrupted, is in Work Mode on the issue, or none
 * of these.
 */
export type IssueChatState = "waiting" | "running" | "interrupted" | "work" | "idle";

const URGENCY: IssueChatState[] = ["waiting", "running", "interrupted", "work", "idle"];

export interface IssueChats {
  /** The chats that worked on the issue in Work Mode, newest first. */
  chats: ChatMeta[];
  /** The chat in Work Mode on the issue now. */
  holder: ChatMeta | undefined;
  /** The most urgent state among the chats; null without chats. */
  state: IssueChatState | null;
  /** How many archived chats worked on the issue; they are not in chats. */
  archived: number;
}

/** A chat's own state in an issue's chat list. */
export function chatState(c: ChatMeta, issueId: string): IssueChatState {
  if (c.waiting) return "waiting";
  if (c.state === "running") return "running";
  if (c.state === "interrupted") return "interrupted";
  if (isHolder(c, issueId)) return "work";
  return "idle";
}

function isHolder(c: ChatMeta, issueId: string): boolean {
  return c.mode === "work" && c.work?.issue_id === issueId;
}

/** The active chats linked to an issue, from the chat list (which is newest first). */
export function chatsForIssue(all: ChatMeta[], issueId: string): IssueChats {
  const linked = all.filter((c) => c.worked_issues?.includes(issueId));
  const chats = linked.filter((c) => !c.archived_at);
  let state: IssueChatState | null = null;
  for (const c of chats) {
    const s = chatState(c, issueId);
    if (state === null || URGENCY.indexOf(s) < URGENCY.indexOf(state)) state = s;
  }
  return { chats, holder: chats.find((c) => isHolder(c, issueId)), state, archived: linked.length - chats.length };
}

/** The chats whose title contains query, ignoring case. */
export function filterChats(chats: ChatMeta[], query: string): ChatMeta[] {
  const q = query.trim().toLowerCase();
  return q ? chats.filter((c) => c.title.toLowerCase().includes(q)) : chats;
}

/** Why Work Mode is unavailable in a project without git, and what to do (xpo-e627e1). */
export function noGitHint(root: string): string {
  return `Work Mode needs a git repository. Run git init in ${root}`;
}

/**
 * Why a chat cannot enter Work Mode, or null when it can. noGitRoot is the
 * project root when the project is not a git repository.
 */
export function enterWorkBlocker(c: ChatMeta, noGitRoot?: string): string | null {
  if (noGitRoot) return noGitHint(noGitRoot);
  if (c.state === "running") return "The chat is running a reply";
  if (c.mode === "work" && c.work) return `The chat is in Work Mode on ${c.work.issue_id}`;
  return null;
}
