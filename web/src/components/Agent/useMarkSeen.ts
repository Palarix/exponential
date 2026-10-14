import { useEffect, useState } from "react";
import type { ChatMeta } from "../../api/chat";
import { useWindowAttention } from "../../hooks/useWindowAttention";
import { useAgentChat } from "./agent-context";

/** How long a chat must be watched before its unseen dot clears, so the user sees which chat finished. */
export const SEEN_GRACE_MS = 1500;

/**
 * Marks a shown chat seen while the window has the user's attention. It
 * waits SEEN_GRACE_MS after the chat becomes watched (opened, or the window
 * regains focus); after that, every change to the chat is seen at once.
 */
export function useMarkSeen(meta: ChatMeta | undefined) {
  const { markSeen } = useAgentChat();
  const attention = useWindowAttention();
  const id = meta?.id;
  // The chat watched for longer than the grace delay.
  const [settled, setSettled] = useState<string | null>(null);

  useEffect(() => {
    if (!attention || !id) return;
    const timer = setTimeout(() => setSettled(id), SEEN_GRACE_MS);
    return () => {
      clearTimeout(timer);
      setSettled(null);
    };
  }, [attention, id]);

  const updatedAt = meta?.updated_at;
  useEffect(() => {
    if (attention && id && settled === id && updatedAt) markSeen(id, updatedAt);
  }, [attention, id, settled, updatedAt, markSeen]);
}
