import { MousePointer2 } from "lucide-react";
import Tooltip from "../ui/Tooltip";
import { useAgentChat } from "./agent-context";
import ChatPopup from "./ChatPopup";

const shortcut = navigator.platform.includes("Mac") ? "⌘J" : "Ctrl+J";

/**
 * The status bar's agent area: a pill per open chat, then the Agent button.
 * Also mounts the chat popup, which floats above the status bar.
 */
export default function AgentStatusBar() {
  const agent = useAgentChat();
  const { status, chats, pills, popupOpen, chatId } = agent;
  if (!status) return null;

  const available = status.available;
  const pillChats = pills.map((id) => chats.find((c) => c.id === id)).filter((c) => c !== undefined);

  return (
    <>
      <div className="flex items-center gap-1.5 shrink-0 min-w-0">
        {pillChats.map((c) => {
          const active = popupOpen && c.id === chatId;
          return (
            <button
              key={c.id}
              type="button"
              onClick={() => (active ? agent.minimize() : agent.openChat(c.id))}
              className={`flex items-center gap-1.5 max-w-48 h-6 px-2.5 rounded-[var(--radius-md)] text-xs transition-colors ${active ? "bg-[var(--color-surface-3)] text-[var(--color-text-primary)]" : "bg-[var(--color-surface-2)] text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface-3)] hover:text-[var(--color-text-primary)]"}`}
            >
              {c.state === "running" && (
                <span className="shrink-0 w-2.5 h-2.5 border-[1.5px] border-[var(--color-text-muted)] border-t-transparent rounded-full animate-spin" />
              )}
              <span className="truncate">{c.title || "New chat"}</span>
            </button>
          );
        })}
        <Tooltip content={available ? `Agent (${shortcut})` : status.reason ?? "Agent chat is not available"}>
          {/* A span keeps the tooltip working while the button is disabled. */}
          <span>
            <button
              type="button"
              disabled={!available}
              onClick={() => (popupOpen && chatId === null ? agent.minimize() : agent.openNew())}
              className="flex items-center gap-1.5 h-6 px-2 rounded-[var(--radius-md)] text-xs text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface-3)] hover:text-[var(--color-text-primary)] disabled:opacity-40 disabled:hover:bg-transparent disabled:hover:text-[var(--color-text-secondary)] transition-colors"
            >
              <MousePointer2 className="w-3.5 h-3.5" />
              Agent
            </button>
          </span>
        </Tooltip>
      </div>
      {popupOpen && available && <ChatPopup />}
    </>
  );
}
