import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { MousePointer2, Minus, X } from "lucide-react";
import { ApiError } from "../../api/client";
import { cancelChat, createChat, sendChatMessage } from "../../api/chat";
import { useToast } from "../ui";
import { agentQueryKeys, useAgentChat } from "./agent-context";
import { SUGGESTIONS } from "./agent-state-utils";
import { isRunning } from "./chat-thread-utils";
import ChatComposer from "./ChatComposer";
import ChatThread from "./ChatThread";
import { useChatThread } from "./useChatThread";

function errorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    try {
      return (JSON.parse(err.body) as { error?: string }).error ?? err.message;
    } catch {
      return err.message;
    }
  }
  return err instanceof Error ? err.message : String(err);
}

export default function ChatPopup() {
  const agent = useAgentChat();
  const { chatId, chats, context, prefs, setPrefs, chatCreated } = agent;
  const thread = useChatThread(chatId);
  const running = isRunning(thread);
  const qc = useQueryClient();
  const showToast = useToast();
  const [dismissedIssue, setDismissedIssue] = useState<string | null>(null);

  const meta = chats.find((c) => c.id === chatId);

  const issueId = context.issue_id && context.issue_id !== dismissedIssue ? context.issue_id : undefined;

  const send = async (text: string): Promise<boolean> => {
    try {
      let id = chatId;
      if (!id) {
        id = (await createChat()).id;
        chatCreated(id);
      }
      await sendChatMessage(id, {
        text,
        context: issueId ? context : { view: context.view },
        model: prefs.model,
        effort: prefs.effort,
      });
      qc.invalidateQueries({ queryKey: agentQueryKeys.chats });
      return true;
    } catch (err) {
      showToast(errorMessage(err), { variant: "error" });
      return false;
    }
  };

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
      <div className="flex items-center gap-2 pl-4 pr-2 h-12 shrink-0">
        <span className="flex-1 truncate text-sm font-medium text-[var(--color-text-primary)]">
          {meta?.title || "New chat"}
        </span>
        <button type="button" aria-label="Minimize" onClick={agent.minimize} className="p-1.5 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-3)]">
          <Minus className="w-4 h-4" />
        </button>
        <button type="button" aria-label="Close" onClick={agent.close} className="p-1.5 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-3)]">
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
                onClick={() => void send(s.prompt)}
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
        onSend={send}
        onStop={() => {
          if (chatId) cancelChat(chatId).catch(() => {});
        }}
      />
    </div>
  );
}
