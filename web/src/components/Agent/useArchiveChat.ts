import { useQueryClient } from "@tanstack/react-query";
import { archiveChat, deleteChat, unarchiveChat } from "../../api/chat";
import { useToast } from "../ui";
import { agentQueryKeys, useAgentChat } from "./agent-context";
import { errorMessage } from "./useChatSession";

/**
 * Archive, unarchive and delete for one chat or many. Each refreshes the
 * chat list, toasts what failed, and resolves to how many chats it changed.
 */
export function useArchiveChat() {
  const { chatDeleted } = useAgentChat();
  const qc = useQueryClient();
  const showToast = useToast();

  const run = async (ids: string[], call: (id: string) => Promise<unknown>, verb: string): Promise<number> => {
    const results = await Promise.allSettled(ids.map(call));
    qc.invalidateQueries({ queryKey: agentQueryKeys.chats });
    const failed = results.filter((r): r is PromiseRejectedResult => r.status === "rejected");
    if (failed.length === 1 && ids.length === 1) showToast(errorMessage(failed[0].reason), { variant: "error" });
    else if (failed.length > 0) showToast(`${failed.length} of ${ids.length} chats could not be ${verb}`, { variant: "error" });
    return ids.length - failed.length;
  };

  return {
    archive: (ids: string[]) => run(ids, archiveChat, "archived"),
    unarchive: (ids: string[]) => run(ids, unarchiveChat, "unarchived"),
    remove: (ids: string[]) => run(ids, async (id) => {
      await deleteChat(id);
      chatDeleted(id);
    }, "deleted"),
  };
}
