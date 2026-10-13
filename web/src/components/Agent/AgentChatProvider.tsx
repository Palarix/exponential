import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { fetchAgentStatus, fetchChats, type ChatContext } from "../../api/chat";
import { useKeyboardShortcuts } from "../../keyboard";
import { AgentChatContext, agentQueryKeys, type AgentChat } from "./agent-context";
import { addPill, chatListPollMs, chatPrefs, loadPills, loadPrefs, normalizePrefs, removePill, safeLocalStorage, savePills, savePrefs, type ChatPrefs } from "./agent-state-utils";

const isMac = navigator.platform.includes("Mac");

export default function AgentChatProvider({ context, children }: { context: ChatContext; children: ReactNode }) {
  const [storage] = useState(safeLocalStorage);
  const [popupOpen, setPopupOpen] = useState(false);
  const [chatId, setChatId] = useState<string | null>(null);
  const [pills, setPills] = useState(() => loadPills(storage));
  const [savedPrefs, setPrefsState] = useState(() => loadPrefs(storage));
  const [drafts, setDrafts] = useState<Record<string, string>>({});

  const status = useQuery({ queryKey: agentQueryKeys.status, queryFn: fetchAgentStatus, staleTime: Infinity }).data;
  // Picks made in open chats, by chat ID. They last until the next reload,
  // when the chat's meta holds what its last message used.
  const [picked, setPicked] = useState<Record<string, ChatPrefs>>({});
  const chatsQuery = useQuery({
    queryKey: agentQueryKeys.chats,
    queryFn: fetchChats,
    enabled: !!status?.available,
    // Pills and issue chat icons show running chats; poll while one runs.
    refetchInterval: (q) => chatListPollMs(q.state.data),
  });
  const chats = useMemo(() => chatsQuery.data ?? [], [chatsQuery.data]);

  // Pills whose chat file is gone drop out once the chat list has loaded.
  const visiblePills = useMemo(() => {
    if (!chatsQuery.isSuccess) return pills;
    const known = new Set(chats.map((c) => c.id));
    return pills.filter((id) => known.has(id));
  }, [chatsQuery.isSuccess, chats, pills]);
  useEffect(() => savePills(storage, visiblePills), [storage, visiblePills]);

  // Saved prefs can name a provider or model that is gone; use the defaults then.
  const prefsFor = useCallback((id: string | null) => {
    const meta = id ? chats.find((c) => c.id === id) : undefined;
    return normalizePrefs(chatPrefs(meta, id ? picked[id] : undefined, savedPrefs), status);
  }, [chats, picked, savedPrefs, status]);

  const setPrefsFor = useCallback((id: string | null, next: ChatPrefs) => {
    if (id) setPicked((p) => ({ ...p, [id]: next }));
    setPrefsState(next);
    savePrefs(storage, next);
  }, [storage]);

  const value = useMemo<AgentChat>(() => ({
    status,
    chats,
    popupOpen,
    chatId,
    pills: visiblePills,
    context,
    prefsFor,
    setPrefsFor,
    openNew: () => { setChatId(null); setPopupOpen(true); },
    openChat: (id, draft) => {
      if (draft) setDrafts((d) => ({ ...d, [id]: draft }));
      setChatId(id);
      // Every open chat gets a pill, so minimizing never loses the way back.
      setPills((p) => addPill(p, id));
      setPopupOpen(true);
    },
    draftFor: (id) => (id ? drafts[id] : undefined),
    clearDraft: (id) => setDrafts((d) => {
      if (!(id in d)) return d;
      const next = { ...d };
      delete next[id];
      return next;
    }),
    toggle: () => setPopupOpen((open) => !open),
    minimize: () => setPopupOpen(false),
    close: () => {
      setPopupOpen(false);
      if (chatId) setPills((p) => removePill(p, chatId));
      setChatId(null);
    },
    chatCreated: (id) => {
      setChatId(id);
      setPills((p) => addPill(p, id));
    },
    chatDeleted: (id) => {
      setPills((p) => removePill(p, id));
      if (chatId === id) setChatId(null);
    },
  }), [status, chats, popupOpen, chatId, visiblePills, context, prefsFor, setPrefsFor, drafts]);

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
