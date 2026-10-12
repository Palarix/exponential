import { useState } from "react";
import { MousePointer2, Maximize2, Minus, X } from "lucide-react";
import { useAppNav } from "../../app/hooks";
import { useAgentChat } from "./agent-context";
import { SUGGESTIONS } from "./agent-state-utils";
import ChatComposer from "./ChatComposer";
import ChatThread from "./ChatThread";
import ChatTitleBar from "./ChatTitleBar";
import { useChatSession } from "./useChatSession";

const headerButton = "p-1.5 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-3)]";

export default function ChatPopup() {
  const agent = useAgentChat();
  const { chatId, chats, context, prefs, setPrefs, chatCreated } = agent;
  const { openChatView } = useAppNav();
  const { thread, running, send, stop } = useChatSession(chatId, chatCreated);
  const [dismissedIssue, setDismissedIssue] = useState<string | null>(null);

  const meta = chats.find((c) => c.id === chatId);
  const issueId = context.issue_id && context.issue_id !== dismissedIssue ? context.issue_id : undefined;
  const sendText = (text: string) => send(text, issueId ? context : { view: context.view });
  const empty = thread.turns.length === 0;

  return (
    <div
      role="dialog"
      aria-label="Agent chat"
      onKeyDown={(e) => {
        if (e.key === "Escape") {
          e.stopPropagation();
          agent.minimize();
        }
      }}
      className="absolute right-3 bottom-10 z-40 w-[420px] max-w-[calc(100vw-1.5rem)] h-[600px] max-h-[calc(100vh-4rem)] flex flex-col rounded-[var(--radius-xl)] border border-[var(--color-border-overlay)] bg-[var(--color-surface-1)] shadow-[var(--shadow-popover)] overflow-hidden"
    >
      <div className="flex items-center gap-1 pl-4 pr-2 h-12 shrink-0">
        <ChatTitleBar
          chat={meta}
          title={<span className="truncate text-sm font-medium text-[var(--color-text-primary)]">{meta?.title || "New chat"}</span>}
        />
        <button
          type="button"
          aria-label="Open in Agent view"
          onClick={() => {
            openChatView(chatId);
            agent.minimize();
          }}
          className={headerButton}
        >
          <Maximize2 className="w-4 h-4" />
        </button>
        <button type="button" aria-label="Minimize" onClick={agent.minimize} className={headerButton}>
          <Minus className="w-4 h-4" />
        </button>
        <button type="button" aria-label="Close" onClick={agent.close} className={headerButton}>
          <X className="w-4 h-4" />
        </button>
      </div>

      {empty ? (
        <div className="flex-1 flex flex-col items-center justify-center gap-4 px-6 text-center">
          <MousePointer2 className="w-6 h-6 text-[var(--color-text-muted)]" />
          <div>
            <div className="text-sm font-medium text-[var(--color-text-primary)]">Chat with your agent</div>
            <div className="text-sm text-[var(--color-text-muted)]">Ask anything or tell xpo what you need</div>
          </div>
          <div className="flex flex-wrap justify-center gap-2">
            {SUGGESTIONS.map((s) => (
              <button
                key={s.label}
                type="button"
                disabled={!agent.status?.available}
                onClick={() => void sendText(s.prompt)}
                className="px-3 py-1.5 rounded-full border border-[var(--color-border-default)] text-xs text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface-3)] hover:text-[var(--color-text-primary)] disabled:opacity-50 transition-colors"
              >
                {s.label}
              </button>
            ))}
          </div>
        </div>
      ) : (
        <ChatThread thread={thread} />
      )}

      <ChatComposer
        running={running}
        disabled={!agent.status?.available}
        placeholder={empty ? "Ask about issues, or the code behind them…" : "Reply…"}
        contextIssue={issueId ? { id: issueId, title: context.issue_title } : undefined}
        onDismissContext={() => setDismissedIssue(context.issue_id ?? null)}
        prefs={prefs}
        onPrefsChange={setPrefs}
        onSend={sendText}
        onStop={stop}
      />
    </div>
  );
}
