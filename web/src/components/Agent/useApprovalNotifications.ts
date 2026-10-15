import { useEffect, useRef } from "react";
import type { ChatMeta } from "../../api/chat";
import { notificationFor, notificationPlan } from "./agent-state-utils";

/**
 * Shows a desktop notification when a chat starts waiting on the viewer while
 * the tab is hidden, and closes it once the chat stops waiting or the tab is
 * visible again. chats is null until the chat list has loaded, so waits that
 * were there on page load do not notify.
 */
export function useApprovalNotifications(chats: ChatMeta[] | null, enabled: boolean, openChat: (id: string) => void): void {
  const prev = useRef<ChatMeta[] | null>(null);
  const open = useRef(new Map<string, Notification>());
  const openRef = useRef(openChat);
  useEffect(() => { openRef.current = openChat; }, [openChat]);

  useEffect(() => {
    if (chats === null) return;
    const before = prev.current;
    prev.current = chats;
    if (before === null || before === chats) return;
    const plan = notificationPlan(before, chats, document.visibilityState === "hidden");
    for (const id of plan.close) {
      open.current.get(id)?.close();
      open.current.delete(id);
    }
    if (!enabled || typeof Notification === "undefined" || Notification.permission !== "granted") return;
    for (const c of plan.show) {
      const { title, options } = notificationFor(c);
      const n = new Notification(title, options);
      n.onclick = () => {
        window.focus();
        openRef.current(c.id);
        n.close();
      };
      n.onclose = () => {
        if (open.current.get(c.id) === n) open.current.delete(c.id);
      };
      open.current.set(c.id, n);
    }
  }, [chats, enabled]);

  useEffect(() => {
    const shown = open.current;
    const onChange = () => {
      if (document.visibilityState !== "visible") return;
      for (const n of shown.values()) n.close();
      shown.clear();
    };
    document.addEventListener("visibilitychange", onChange);
    return () => document.removeEventListener("visibilitychange", onChange);
  }, []);
}
