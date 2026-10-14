import { useEffect } from "react";
import { useHiddenSince } from "../../hooks/useHiddenSince";
import { faviconFor } from "../../favicon";
import { useAgentChat } from "./agent-context";
import { tabAttention } from "./agent-state-utils";

/**
 * Sets the tab title, and badges the favicon when a chat waits on an
 * approval or a reply came in while the tab was hidden.
 */
export default function TabStatus({ title }: { title: string }) {
  const { chats } = useAgentChat();
  const hiddenSince = useHiddenSince();
  const attention = tabAttention(chats, hiddenSince);

  useEffect(() => {
    document.title = title;
  }, [title]);

  useEffect(() => {
    const link = document.querySelector<HTMLLinkElement>('link[rel="icon"]');
    if (link) link.href = faviconFor(attention);
  }, [attention]);

  return null;
}
