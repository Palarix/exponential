import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { invalidateOnServerEvent, queryKeys } from "../api/query-keys";
import { agentQueryKeys } from "../components/Agent/agent-context";
import { live } from ".";

/**
 * Opens the tab's event stream and refetches what each signal topic
 * covers. Each (re)connect calls every handler, since changes may have been
 * missed while away.
 */
export function useLive() {
  const qc = useQueryClient();
  useEffect(() => {
    const off = [
      live.onSignal("issues", () => { invalidateOnServerEvent(qc); }),
      live.onSignal("drafts", () => { qc.invalidateQueries({ queryKey: queryKeys.issueDrafts }); }),
      live.onSignal("chats", () => { qc.invalidateQueries({ queryKey: agentQueryKeys.chats }); }),
    ];
    live.start();
    return () => {
      for (const fn of off) fn();
      live.stop();
    };
  }, [qc]);
}
