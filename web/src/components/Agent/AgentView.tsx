import { useRef, useState, type ReactNode } from "react";
import { Bug, ChevronDown, Dock, ListChecks, MessageSquare, MousePointer2, SquarePen, Workflow } from "lucide-react";
import type { ChatMeta } from "../../api/chat";
import { useConfig } from "../../api/queries";
import { formatRelativeTime } from "../../utils/format";
import { useAppNav } from "../../app/hooks";
import { Button, Menu, MenuDivider, MenuItem, Popover } from "../ui";import { useAgentChat } from "./agent-context";
import { SUGGESTIONS } from "./agent-state-utils";
import ChatComposer from "./ChatComposer";
import ChatThread from "./ChatThread";
import ChatTitleBar from "./ChatTitleBar";
import { chatStateIcon, recentChatItems } from "./recent-chats";
import { isEmptyThread } from "./chat-thread-utils";
import { useChatSession } from "./useChatSession";
import { useWorkMode } from "./useWorkMode";

const SUGGESTION_ICONS = [<Bug key="bug" size={16} />, <Workflow key="epic" size={16} />, <ListChecks key="list" size={16} />];

/** The header title: opens a menu with "New chat" and the recent chats. */
function ChatSwitcher({ title }: { title: string }) {
  const { chats } = useAgentChat();
  const { chatId, openChatView } = useAppNav();
  const btnRef = useRef<HTMLButtonElement>(null);
  const [open, setOpen] = useState(false);
  const pick = (id: string | null) => {
    setOpen(false);
    openChatView(id);
  };

  return (
    <>
      <button
        ref={btnRef}
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex items-center gap-1.5 min-w-0 h-7 px-2 -ml-2 rounded-[var(--radius-md)] text-sm font-medium text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-3)] transition-colors"
      >
        <span className="truncate">{title}</span>
        <ChevronDown className="w-3.5 h-3.5 shrink-0 text-[var(--color-text-muted)]" />
      </button>
      {open && (
        <Popover anchorRef={btnRef} onClose={() => setOpen(false)}>
          <Menu onClose={() => setOpen(false)} aria-label="Chats" className="w-96" maxHeight="24rem">
            <MenuItem label="New chat" icon={<SquarePen size={14} />} onClick={() => pick(null)} />
            {chats.length > 0 && <MenuDivider />}
            {recentChatItems(chats, chatId, pick)}
          </Menu>
        </Popover>
      )}
    </>
  );
}

const HOME_RECENT_LIMIT = 5;

/** The newest chats under the suggestion cards on the empty Agent view. */
function RecentChatsList({ chats, onOpen }: { chats: ChatMeta[]; onOpen: (id: string) => void }) {
  if (chats.length === 0) return null;
  return (
    <div className="w-full max-w-3xl px-4">
      <div className="px-1 mb-2 text-xs font-medium text-[var(--color-text-muted)]">Recent chats</div>
      <ul className="flex flex-col">
        {chats.slice(0, HOME_RECENT_LIMIT).map((c) => (
          <li key={c.id}>
            <button
              type="button"
              onClick={() => onOpen(c.id)}
              className="flex items-center gap-3 w-full h-9 px-3 rounded-[var(--radius-md)] text-left hover:bg-[var(--color-hover-surface-2)] transition-colors"
            >
              {chatStateIcon(c, "w-3.5 h-3.5", <MessageSquare className="shrink-0 w-3.5 h-3.5 text-[var(--color-text-muted)]" />)}
              <span className="flex-1 min-w-0 truncate text-sm text-[var(--color-text-secondary)]">{c.title || "Untitled chat"}</span>
              <span className="shrink-0 whitespace-nowrap text-xs text-[var(--color-text-muted)]">{formatRelativeTime(c.updated_at)}</span>
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}

function Centered({ children }: { children: ReactNode }) {
  return <div className="h-full flex flex-col items-center justify-center gap-4 px-6 text-center">{children}</div>;
}

export default function AgentView() {
  const agent = useAgentChat();
  const { chatId, openChatView, leaveChatView } = useAppNav();
  const { projectName } = useConfig();
  const { thread, missing, running, send, stop, expireApproval } = useChatSession(chatId, (id) => openChatView(id));
  const meta = agent.chats.find((c) => c.id === chatId);
  const { work, onLeaveWork, onOpenIssue } = useWorkMode(meta);

  if (!agent.status) return null;
  if (!agent.status.available) {
    return (
      <Centered>
        <MousePointer2 className="w-6 h-6 text-[var(--color-text-muted)]" />
        <div className="text-sm text-[var(--color-text-secondary)] max-w-md">{agent.status.reason ?? "Agent chat is not available."}</div>
      </Centered>
    );
  }
  if (missing) {
    return (
      <Centered>
        <div className="text-sm text-[var(--color-text-secondary)]">Chat not found. It may have been deleted.</div>
        <Button variant="secondary" size="sm" icon={<SquarePen size={14} />} onClick={() => openChatView(null)}>New chat</Button>
      </Centered>
    );
  }

  const empty = isEmptyThread(thread);
  const sendText = (text: string) => send(text, { view: "Agent" });
  const composer = (
    <ChatComposer
      wide
      running={running}
      placeholder={empty ? `Ask ${projectName || "xpo"} anything…` : "Reply…"}
      onDismissContext={() => {}}
      status={agent.status}
      prefs={agent.prefsFor(chatId)}
      providerLocked={thread.turns.length > 0}
      onPrefsChange={(next) => agent.setPrefsFor(chatId, next)}
      onSend={sendText}
      onStop={stop}
      work={work}
      onLeaveWork={onLeaveWork}
      onOpenIssue={onOpenIssue}
      draft={agent.draftFor(chatId)}
      onDraftUsed={() => chatId && agent.clearDraft(chatId)}
    />
  );

  return (
    <div className="h-full flex flex-col">
      <div className="flex items-center h-12 px-6 shrink-0 border-b border-[var(--color-border-subtle)]">
        <ChatTitleBar
          key={chatId ?? "new"}
          chat={meta}
          title={<ChatSwitcher title={meta?.title || "New chat"} />}
          onDeleted={() => openChatView(null)}
        />
        {chatId && (
          <button
            type="button"
            aria-label="Dock chat"
            onClick={() => {
              agent.openChat(chatId);
              leaveChatView();
            }}
            className="shrink-0 p-1.5 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-3)]"
          >
            <Dock className="w-4 h-4" />
          </button>
        )}
      </div>

      {empty && !chatId ? (
        <div className="flex-1 overflow-y-auto flex flex-col items-center justify-center gap-8 py-10">
          <h1 className="text-2xl font-semibold text-[var(--color-text-primary)] px-4 text-center">
            What do you want to know about {projectName || "this project"}?
          </h1>
          {composer}
          {/* -mt-6 cancels the composer's bottom padding, so the gap matches the others. */}
          <div className="w-full max-w-3xl px-4 -mt-6 grid grid-cols-1 sm:grid-cols-3 gap-3">
            {SUGGESTIONS.map((s, i) => (
              <button
                key={s.label}
                type="button"
                onClick={() => void sendText(s.prompt)}
                className="flex flex-col items-start gap-3 p-4 text-left rounded-[var(--radius-lg)] border border-[var(--color-border-subtle)] hover:bg-[var(--color-hover-surface-2)] hover:border-[var(--color-border-default)] transition-colors"
              >
                <span className="text-[var(--color-text-muted)]">{SUGGESTION_ICONS[i]}</span>
                <span>
                  <span className="block text-sm font-medium text-[var(--color-text-primary)]">{s.label}</span>
                  <span className="block mt-1 text-xs text-[var(--color-text-muted)]">{s.description}</span>
                </span>
              </button>
            ))}
          </div>
          <RecentChatsList chats={agent.chats} onOpen={(id) => openChatView(id)} />
        </div>
      ) : (
        <>
          <ChatThread thread={thread} wide chatId={chatId} onApprovalExpired={expireApproval} cacheExpiresAt={meta?.cache_expires_at} />
          {composer}
        </>
      )}
    </div>
  );
}
