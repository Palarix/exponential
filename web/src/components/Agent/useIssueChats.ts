import { useOptionalAgentChat } from "./agent-context";
import { chatsForIssue, type IssueChats } from "./issue-chats";

const NO_CHATS: IssueChats = { chats: [], holder: undefined, state: null, archived: 0 };

/** The chats linked to an issue, with a count of the archived ones; none when the agent is unavailable. */
export function useIssueChats(issueId: string): IssueChats {
  const agent = useOptionalAgentChat();
  if (!agent?.status?.available) return NO_CHATS;
  return chatsForIssue(agent.allChats, issueId);
}

/** How a list names a count of chats: "1 chat", "3 chats". */
export function chatCount(n: number): string {
  return `${n} ${n === 1 ? "chat" : "chats"}`;
}
