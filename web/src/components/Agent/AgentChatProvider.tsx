import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { fetchAgentStatus, fetchChats, type ChatContext } from "../../api/chat";
import { useKeyboardShortcuts } from "../../keyboard";
import { AgentChatContext, agentQueryKeys, type AgentChat } from "./agent-context";
import { activeChats } from "./chat-history-utils";
import { createComposerDrafts } from "./composer-drafts";
import { useApprovalNotifications } from "./useApprovalNotifications";
import { addPill, adoptNewChatContext, chatPrefs, consumeContext, consumedFor, loadPills, loadSeen, markSeen, nextViewed, pruneSeen, pendingContextIssue, loadNotify, loadPrefs, normalizePrefs, notifyState, saveNotify, removePill, safeLocalStorage, savePills, savePrefs, saveSeen, type ChatPrefs, type ConsumedContext } from "./agent-state-utils";

const isMac = navigator.platform.includes("Mac");

/** The browser's notification permission; undefined when it has no Notification API. */
function notificationPermission(): NotificationPermission | undefined {
  return typeof Notification === "undefined" ? undefined : Notification.permission;
}

export default function AgentChatProvider({ context, children }: { context: ChatContext; children: ReactNode }) {
  const [storage] = useState(safeLocalStorage);
  const [popupOpen, setPopupOpen] = useState(false);
  const [chatId, setChatId] = useState<string | null>(null);
  const [pills, setPills] = useState(() => loadPills(storage));
  const [seen, setSeen] = useState(() => loadSeen(storage));
  const [savedPrefs, setPrefsState] = useState(() => loadPrefs(storage));
  const [notifyOn, setNotifyOn] = useState(() => loadNotify(storage));
  const [permission, setPermission] = useState(notificationPermission);
  const [drafts] = useState(createComposerDrafts);
  // Each issue the user opens becomes pending context for every chat, until
  // that chat sends or removes it. This outlives the popup, not a reload.
  const [viewed, setViewed] = useState(() => nextViewed({ seq: 0 }, context.issue_id));
  const [consumed, setConsumed] = useState<ConsumedContext>({});
  const nextView = nextViewed(viewed, context.issue_id);
  if (nextView !== viewed) setViewed(nextView);

  const status = useQuery({ queryKey: agentQueryKeys.status, queryFn: fetchAgentStatus, staleTime: Infinity }).data;
  // Picks made in open chats, by chat ID. They last until the next reload,
  // when the chat's meta holds what its last message used.
  const [picked, setPicked] = useState<Record<string, ChatPrefs>>({});
  const chatsQuery = useQuery({
    queryKey: agentQueryKeys.chats,
    queryFn: fetchChats,
    // No polling: the event stream's chats signal refetches it (useLive).
    enabled: !!status?.available,
  });
  const allChats = useMemo(() => chatsQuery.data ?? [], [chatsQuery.data]);
  const chats = useMemo(() => activeChats(allChats), [allChats]);

  // Pills whose chat file is gone or archived drop out once the chat list has loaded.
  const visiblePills = useMemo(() => {
    if (!chatsQuery.isSuccess) return pills;
    const known = new Set(chats.map((c) => c.id));
    return pills.filter((id) => known.has(id));
  }, [chatsQuery.isSuccess, chats, pills]);
  useEffect(() => savePills(storage, visiblePills), [storage, visiblePills]);

  // Seen records of deleted chats drop out once the chat list has loaded.
  const visibleSeen = useMemo(() => (chatsQuery.isSuccess ? pruneSeen(seen, allChats.map((c) => c.id)) : seen), [chatsQuery.isSuccess, allChats, seen]);
  useEffect(() => saveSeen(storage, visibleSeen), [storage, visibleSeen]);
  const markChatSeen = useCallback((id: string, updatedAt: string) => setSeen((s) => markSeen(s, id, updatedAt)), []);

  // Saved prefs can name a provider or model that is gone; use the defaults then.
  const prefsFor = useCallback((id: string | null) => {
    const meta = id ? allChats.find((c) => c.id === id) : undefined;
    return normalizePrefs(chatPrefs(meta, id ? picked[id] : undefined, savedPrefs), status);
  }, [allChats, picked, savedPrefs, status]);

  const setPrefsFor = useCallback((id: string | null, next: ChatPrefs) => {
    if (id) setPicked((p) => ({ ...p, [id]: next }));
    setPrefsState(next);
    savePrefs(storage, next);
  }, [storage]);

  const notify = notifyState(notifyOn, permission);
  const toggleNotify = useCallback(() => {
    const turnOn = (on: boolean) => {
      setNotifyOn(on);
      saveNotify(storage, on);
    };
    if (notify === "on") return turnOn(false);
    if (notify !== "off") return;
    // Browsers only allow the permission prompt from a user gesture, like this click.
    if (Notification.permission === "granted") return turnOn(true);
    void Notification.requestPermission().then((p) => {
      setPermission(p);
      turnOn(p === "granted");
    });
  }, [notify, storage]);

  const value = useMemo<AgentChat>(() => {
    const closeChat = () => {
      setPopupOpen(false);
      if (chatId) setPills((p) => removePill(p, chatId));
      setChatId(null);
    };
    return {
      status,
      chats,
      allChats,
      popupOpen,
      chatId,
      pills: visiblePills,
      seen: visibleSeen,
      markSeen: markChatSeen,
      context,
      prefsFor,
      setPrefsFor,
      notify,
      toggleNotify,
      openNew: () => { setChatId(null); setPopupOpen(true); },
      openChat: (id, draft) => {
        if (draft) drafts.prefill(id, draft);
        setChatId(id);
        // Every open chat gets a pill, so minimizing never loses the way back.
        setPills((p) => addPill(p, id));
        setPopupOpen(true);
      },
      drafts,
      pendingIssueFor: (id) => {
        const meta = id ? chats.find((c) => c.id === id) : undefined;
        return pendingContextIssue(viewed, consumedFor(consumed, id), meta?.mode === "work" ? meta.work?.issue_id : undefined);
      },
      consumeContext: (id) => setConsumed((c) => consumeContext(c, id, viewed.seq)),
      toggle: () => setPopupOpen((open) => !open),
      minimize: () => setPopupOpen(false),
      close: closeChat,
      // The Agent view shows the chat now; its dock button brings the pill back.
      maximize: closeChat,
      chatCreated: (id) => {
        setChatId(id);
        setConsumed((c) => adoptNewChatContext(c, id));
        setPills((p) => addPill(p, id));
      },
      chatDeleted: (id) => {
        setPills((p) => removePill(p, id));
        if (chatId === id) setChatId(null);
      },
    };
  }, [status, chats, allChats, popupOpen, chatId, visiblePills, visibleSeen, markChatSeen, context, prefsFor, setPrefsFor, notify, toggleNotify, drafts, viewed, consumed]);

  useApprovalNotifications(chatsQuery.isSuccess ? chats : null, notify === "on", value.openChat);

  useKeyboardShortcuts({
    scope: "global.agent",
    priority: "global",
    enabled: !!status?.available,
    shortcuts: [
      { id: "agent.toggle", key: "j", label: "Toggle agent chat", group: "Global", modifiers: isMac ? { meta: true } : { ctrl: true }, allowInEditable: true, run: value.toggle },
    ],
  });

  return <AgentChatContext.Provider value={value}>{children}</AgentChatContext.Provider>;
}
