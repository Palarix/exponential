import { useLayoutEffect, useRef, useState, type PointerEvent as ReactPointerEvent } from "react";
import { MousePointer2, Maximize2, Minus, X } from "lucide-react";
import { useIssueList } from "../../api/queries";
import { useAppNav } from "../../app/hooks";
import { useAgentChat } from "./agent-context";
import { clampPopupSize, loadPopupSize, popupLeft, safeLocalStorage, savePopupSize, SUGGESTIONS, type PopupSize } from "./agent-state-utils";
import ChatComposer from "./ChatComposer";
import ChatThread from "./ChatThread";
import ChatTitleBar from "./ChatTitleBar";
import { isEmptyThread } from "./chat-thread-utils";
import { useChatSession } from "./useChatSession";
import { useWorkMode } from "./useWorkMode";

const headerButton = "p-1.5 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-ai-1)]";

/**
 * The chat popup. It floats centered above anchor(): the open chat's pill,
 * or the Agent button for a new chat.
 */
export default function ChatPopup({ anchor }: { anchor(): HTMLElement | null }) {
  const agent = useAgentChat();
  const ref = useRef<HTMLDivElement>(null);
  const [storage] = useState(safeLocalStorage);
  const [size, setSize] = useState(() => loadPopupSize(storage));

  // Position after every render, and on resize: pills come and go, and their
  // widths change with titles. Writing the style directly avoids a re-render.
  useLayoutEffect(() => {
    const place = () => {
      const pop = ref.current;
      const a = anchor()?.getBoundingClientRect();
      if (!pop || !a) return;
      pop.style.left = `${popupLeft(a.left + a.width / 2, pop.offsetWidth, window.innerWidth)}px`;
      pop.style.bottom = `${window.innerHeight - a.top + 8}px`;
      pop.style.visibility = "visible";
    };
    place();
    window.addEventListener("resize", place);
    return () => window.removeEventListener("resize", place);
  });
  const { chatId, chats, context, prefsFor, setPrefsFor, chatCreated, draftFor, clearDraft, pendingIssueFor, consumeContext } = agent;
  const { openChatView } = useAppNav();
  const { thread, running, send, stop, expireApproval } = useChatSession(chatId, chatCreated);
  const issues = useIssueList();

  const meta = chats.find((c) => c.id === chatId);
  const { work, onLeaveWork, onOpenIssue } = useWorkMode(meta);
  // The chip is the context the next message carries; sending uses it up.
  const issueId = pendingIssueFor(chatId);
  const issueTitle = issues.find((i) => i.id === issueId)?.title;
  const sendText = (text: string) => {
    consumeContext(chatId);
    return send(text, issueId ? { view: context.view, issue_id: issueId, issue_title: issueTitle } : { view: context.view });
  };
  const empty = isEmptyThread(thread);

  return (
    <div
      ref={ref}
      role="dialog"
      aria-label="Agent chat"
      style={{ visibility: "hidden", width: size.width, height: size.height }}
      onKeyDown={(e) => {
        if (e.key === "Escape") {
          e.stopPropagation();
          agent.minimize();
        }
      }}
      className="surface-ai fixed z-40 max-w-[calc(100vw-1.5rem)] max-h-[calc(100vh-4rem)] flex flex-col rounded-[var(--radius-xl)] border border-[var(--color-border-ai)] bg-[var(--color-surface-ai-1)] shadow-[var(--shadow-popover)] overflow-hidden"
    >
      <ResizeHandles size={size} onResize={setSize} onDone={(next) => savePopupSize(storage, next)} />
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
            agent.maximize();
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
                className="px-3 py-1.5 rounded-full border border-[var(--color-border-ai)] text-xs text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface-ai-1)] hover:text-[var(--color-text-primary)] disabled:opacity-50 transition-colors"
              >
                {s.label}
              </button>
            ))}
          </div>
        </div>
      ) : (
        <ChatThread thread={thread} chatId={chatId} onApprovalExpired={expireApproval} cacheExpiresAt={meta?.cache_expires_at} />
      )}

      <ChatComposer
        running={running}
        disabled={!agent.status?.available}
        placeholder={empty ? "Ask about issues, or the code behind them…" : "Reply…"}
        contextIssue={issueId ? { id: issueId, title: issueTitle } : undefined}
        onDismissContext={() => consumeContext(chatId)}
        status={agent.status}
        prefs={prefsFor(chatId)}
        providerLocked={thread.turns.length > 0}
        onPrefsChange={(next) => setPrefsFor(chatId, next)}
        onSend={sendText}
        onStop={stop}
        work={work}
        onLeaveWork={onLeaveWork}
        onOpenIssue={onOpenIssue}
        draft={draftFor(chatId)}
        onDraftUsed={() => chatId && clearDraft(chatId)}
      />
    </div>
  );
}

// Each handle grows the popup in its directions: x is -1 (left edge), 1
// (right edge) or 0; y is 1 for the top edge, 0 otherwise. The popup stays
// centered on its anchor, so a side drag grows both sides.
const HANDLES: { x: -1 | 0 | 1; y: 0 | 1; className: string }[] = [
  { x: 0, y: 1, className: "top-0 inset-x-3 h-1.5 cursor-ns-resize" },
  { x: -1, y: 0, className: "left-0 inset-y-3 w-1.5 cursor-ew-resize" },
  { x: 1, y: 0, className: "right-0 inset-y-3 w-1.5 cursor-ew-resize" },
  { x: -1, y: 1, className: "top-0 left-0 w-3 h-3 cursor-nwse-resize" },
  { x: 1, y: 1, className: "top-0 right-0 w-3 h-3 cursor-nesw-resize" },
];

/** Drag handles on the popup's top edge, sides and top corners. */
function ResizeHandles({ size, onResize, onDone }: { size: PopupSize; onResize(s: PopupSize): void; onDone(s: PopupSize): void }) {
  const start = (e: ReactPointerEvent<HTMLDivElement>, x: number, y: number) => {
    e.preventDefault();
    const el = e.currentTarget;
    el.setPointerCapture(e.pointerId);
    const from = { x: e.clientX, y: e.clientY, ...size };
    let next = size;
    const move = (ev: PointerEvent) => {
      next = clampPopupSize(
        { width: from.width + x * 2 * (ev.clientX - from.x), height: from.height - y * (ev.clientY - from.y) },
        { width: window.innerWidth, height: window.innerHeight },
      );
      onResize(next);
    };
    const up = () => {
      el.removeEventListener("pointermove", move);
      el.removeEventListener("pointerup", up);
      onDone(next);
    };
    el.addEventListener("pointermove", move);
    el.addEventListener("pointerup", up);
  };
  return (
    <>
      {HANDLES.map((h) => (
        <div key={h.className} aria-hidden onPointerDown={(e) => start(e, h.x, h.y)} className={`absolute z-10 ${h.className}`} />
      ))}
    </>
  );
}
