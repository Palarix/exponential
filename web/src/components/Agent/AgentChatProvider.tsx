import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { fetchAgentStatus, fetchChats, type ChatContext } from "../../api/chat";
import { useKeyboardShortcuts } from "../../keyboard";
import { AgentChatContext, agentQueryKeys, type AgentChat } from "./agent-context";
import { addPill, loadPills, loadPrefs, normalizePrefs, removePill, savePills, savePrefs, type ChatPrefs } from "./agent-state-utils";

function safeLocalStorage(): Storage | null {
  try { return window.localStorage; } catch { return null; }
}

const isMac = navigator.platform.includes("Mac");

export default function AgentChatProvider({ context, children }: { context: ChatContext; children: ReactNode }) {
  const [storage] = useState(safeLocalStorage);
  const [popupOpen, setPopupOpen] = useState(false);
  const [chatId, setChatId] = useState<string | null>(null);
  const [pills, setPills] = useState(() => loadPills(storage));
  const [savedPrefs, setPrefsState] = useState(() => loadPrefs(storage));

  const status = useQuery({ queryKey: agentQueryKeys.status, queryFn: fetchAgentStatus, staleTime: Infinity }).data;
  // Saved prefs can name a provider or model that is gone; use the defaults then.
  const prefs = useMemo(() => normalizePrefs(savedPrefs, status), [savedPrefs, status]);
  const anyPillRunning = (chats: { id: string; state: string }[] | undefined) =>
    !!chats?.some((c) => c.state === "running" && pills.includes(c.id));
  const chatsQuery = useQuery({
    queryKey: agentQueryKeys.chats,
    queryFn: fetchChats,
    enabled: !!status?.available,
    // Pills of chats in the background show a spinner; poll while one runs.
    refetchInterval: (q) => (anyPillRunning(q.state.data) ? 2000 : false),
  });
  const chats = useMemo(() => chatsQuery.data ?? [], [chatsQuery.data]);

  // Pills whose chat file is gone drop out once the chat list has loaded.
  const visiblePills = useMemo(() => {
    if (!chatsQuery.isSuccess) return pills;
    const known = new Set(chats.map((c) => c.id));
    return pills.filter((id) => known.has(id));
  }, [chatsQuery.isSuccess, chats, pills]);
  useEffect(() => savePills(storage, visiblePills), [storage, visiblePills]);

  const setPrefs = useCallback((next: ChatPrefs) => {
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
    prefs,
    setPrefs,
    openNew: () => { setChatId(null); setPopupOpen(true); },
    openChat: (id) => {
      // An existing chat opens with the agent, model and effort it last used.
      const meta = chats.find((c) => c.id === id);
      if (meta) setPrefs({ provider: meta.agent || prefs.provider, model: meta.model ?? "", effort: meta.effort ?? "" });
      setChatId(id);
      setPopupOpen(true);
    },
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
  }), [status, chats, popupOpen, chatId, visiblePills, context, prefs, setPrefs]);

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
