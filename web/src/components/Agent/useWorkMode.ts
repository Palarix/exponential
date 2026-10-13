import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { leaveWork, startWorkChat, type ChatMeta } from "../../api/chat";
import { queryKeys } from "../../api/query-keys";
import { useStartWork } from "../../api/mutations";
import { useIssueList } from "../../api/queries";
import { useAppNav } from "../../app/hooks";
import { useToast } from "../ui";
import { agentQueryKeys, useOptionalAgentChat } from "./agent-context";
import type { WorkModeInfo } from "./ComposerPickers";
import { errorMessage } from "./useChatSession";

/** Start Work's first message. The user sends it, or adds to it first. */
export function workDraft(issueId: string): string {
  return `Let's implement ${issueId}.`;
}

/**
 * Start Work or Continue Work on an issue: opens the chat in Work Mode on
 * it, or a new one with a prefilled first message. The server starts the
 * issue unless it is DOING already. Without an available agent (canChat
 * false) it only starts the issue.
 */
export function useStartWorkChat(): { startWork: (issueId: string) => void; starting: boolean; canChat: boolean } {
  const agent = useOptionalAgentChat();
  const qc = useQueryClient();
  const showToast = useToast();
  const { mutate: startOnly, isPending: startingOnly } = useStartWork();
  const [startingChat, setStartingChat] = useState(false);

  const startWork = (issueId: string) => {
    if (!agent?.status?.available) {
      startOnly(issueId);
      return;
    }
    setStartingChat(true);
    startWorkChat(issueId)
      .then(({ chat, created }) => {
        qc.invalidateQueries({ queryKey: queryKeys.issues });
        qc.invalidateQueries({ queryKey: agentQueryKeys.chats });
        if (created) agent.chatCreated(chat.id);
        agent.openChat(chat.id, created ? workDraft(issueId) : undefined);
      })
      .catch((err: unknown) => showToast(errorMessage(err), { variant: "error" }))
      .finally(() => setStartingChat(false));
  };
  return { startWork, starting: startingOnly || startingChat, canChat: !!agent?.status?.available };
}

/** A chat's Work Mode for the composer: the bound issue, and how to leave. */
export function useWorkMode(meta: ChatMeta | undefined): { work?: WorkModeInfo; onLeaveWork(): void; onOpenIssue(id: string): void } {
  const issues = useIssueList();
  const qc = useQueryClient();
  const showToast = useToast();
  const { openIssue } = useAppNav();
  const issueId = meta?.mode === "work" ? meta.work?.issue_id : undefined;
  const work = issueId ? { issueId, title: issues.find((i) => i.id === issueId)?.title } : undefined;

  const onLeaveWork = () => {
    if (!meta) return;
    leaveWork(meta.id)
      .then(() => qc.invalidateQueries({ queryKey: agentQueryKeys.chats }))
      .catch((err: unknown) => showToast(errorMessage(err), { variant: "error" }));
  };
  return { work, onLeaveWork, onOpenIssue: openIssue };
}
