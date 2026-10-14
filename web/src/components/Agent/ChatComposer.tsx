import { useEffect, useRef, useState } from "react";
import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import { ArrowUp, Square, X } from "lucide-react";
import type { AgentStatus, BackgroundTask } from "../../api/chat";
import type { ChatPrefs } from "./agent-state-utils";
import ComposerPickers, { type WorkModeInfo } from "./ComposerPickers";

/** The banner for work the agent keeps running between turns. */
export interface BackgroundBannerInfo {
  /** Such as "Monitoring" or "2 agents working". */
  title: string;
  /** A subagent works, rather than only shells and monitors. */
  working: boolean;
  /** Listed in the card the title opens. */
  tasks: BackgroundTask[];
  /** Stop was pressed; the tasks have not ended yet. */
  stopping: boolean;
  onStop(): void;
}

interface ChatComposerProps {
  running: boolean;
  /** Set while the agent runs background tasks. Shown between turns. */
  background?: BackgroundBannerInfo;
  disabled?: boolean;
  placeholder: string;
  /** The open issue, shown as a removable chip; undefined hides it. */
  contextIssue?: { id: string; title?: string };
  onDismissContext(): void;
  /** The providers and their models; the pickers are empty until it loads. */
  status: AgentStatus | undefined;
  prefs: ChatPrefs;
  /** The chat has started: its session belongs to the picked provider. */
  providerLocked?: boolean;
  onPrefsChange(prefs: ChatPrefs): void;
  onSend(text: string): Promise<boolean>;
  onStop(): void;
  /** The full-page variant: a centered column and a taller input. */
  wide?: boolean;
  /** Set in Work Mode. */
  work?: WorkModeInfo;
  onLeaveWork?(): void;
  onOpenIssue?(issueId: string): void;
  /** Text to put in the input, such as Start Work's first message. onDraftUsed runs once it is in. */
  draft?: string;
  onDraftUsed?(): void;
}

/** The background tasks, one per line: what each one runs. */
export function BackgroundTaskList({ tasks }: { tasks: BackgroundTask[] }) {
  return (
    <ul className="flex flex-col gap-1">
      {tasks.map((t) => (
        <li key={t.id} className="flex gap-2 min-w-0">
          <span className="shrink-0 text-[var(--color-text-muted)]">{t.kind === "agent" ? "Agent" : "Task"}</span>
          <span className="min-w-0 break-words text-[var(--color-text-primary)]">{t.description || (t.kind === "agent" ? "Agent" : "Background task")}</span>
        </li>
      ))}
    </ul>
  );
}

/** The banner's title. Hover, focus or click opens a card with the tasks. */
function BackgroundTitle({ background }: { background: BackgroundBannerInfo }) {
  const [open, setOpen] = useState(false);
  return (
    <TooltipPrimitive.Root open={open} onOpenChange={setOpen} delayDuration={150}>
      <TooltipPrimitive.Trigger asChild>
        <button
          type="button"
          aria-label="Show background tasks"
          onClick={() => setOpen(true)}
          className="min-w-0 truncate rounded-[var(--radius-sm)] text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] focus-visible:outline focus-visible:outline-1 focus-visible:outline-[var(--color-border-focus)]"
        >
          {background.title}
        </button>
      </TooltipPrimitive.Trigger>
      <TooltipPrimitive.Portal>
        <TooltipPrimitive.Content
          side="top"
          align="start"
          sideOffset={6}
          collisionPadding={12}
          className="z-50 max-w-sm max-h-64 overflow-auto px-3 py-2 rounded-[var(--radius-md)] bg-[var(--color-surface-ai-2)] border border-[var(--color-border-ai)] shadow-[var(--shadow-md)] text-xs leading-relaxed animate-in fade-in-0 zoom-in-95"
        >
          <BackgroundTaskList tasks={background.tasks} />
        </TooltipPrimitive.Content>
      </TooltipPrimitive.Portal>
    </TooltipPrimitive.Root>
  );
}

export default function ChatComposer({
  running,
  background,
  disabled,
  placeholder,
  contextIssue,
  onDismissContext,
  status,
  prefs,
  providerLocked,
  onPrefsChange,
  onSend,
  onStop,
  wide,
  work,
  onLeaveWork,
  onOpenIssue,
  draft,
  onDraftUsed,
}: ChatComposerProps) {
  const [text, setText] = useState(draft ?? "");
  const [sending, setSending] = useState(false);
  const inputRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => inputRef.current?.focus(), []);

  // A draft fills the input but sends nothing: the user can add to it first.
  const [shownDraft, setShownDraft] = useState(draft);
  if (draft !== shownDraft) {
    setShownDraft(draft);
    if (draft) setText(draft);
  }
  useEffect(() => {
    if (!draft) return;
    inputRef.current?.focus();
    onDraftUsed?.();
  }, [draft, onDraftUsed]);

  // Grow with the text, up to about eight lines.
  useEffect(() => {
    const el = inputRef.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = `${Math.min(el.scrollHeight, 180)}px`;
  }, [text]);

  const canSend = !running && !sending && !disabled && text.trim() !== "";
  const send = async () => {
    if (!canSend) return;
    setSending(true);
    const ok = await onSend(text.trim());
    setSending(false);
    if (ok) setText("");
    inputRef.current?.focus();
  };

  return (
    <div className={wide ? "w-full max-w-3xl mx-auto px-4 pb-6" : "px-3 pb-3"}>
      {contextIssue && (
        <div className="flex items-center gap-2 px-3 py-1.5 mb-1 text-xs min-w-0">
          <span className="font-mono text-[var(--color-text-muted)] shrink-0">{contextIssue.id}</span>
          <span className="text-[var(--color-text-secondary)] truncate">{contextIssue.title}</span>
          <button
            type="button"
            aria-label="Remove issue context"
            onClick={onDismissContext}
            className="ml-auto shrink-0 p-0.5 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-ai-1)]"
          >
            <X className="w-3 h-3" />
          </button>
        </div>
      )}
      {background && !running && (
        <div className="flex items-center gap-2 pl-3 pr-1 py-1 mb-1 text-xs min-w-0" role="status">
          <span className={`size-1.5 shrink-0 rounded-full bg-[var(--color-text-secondary)] ${background.working ? "animate-pulse" : ""}`} aria-hidden />
          <BackgroundTitle background={background} />
          <button
            type="button"
            aria-label="Stop background tasks"
            onClick={background.onStop}
            disabled={background.stopping}
            className="ml-auto shrink-0 px-2 py-0.5 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-ai-1)] disabled:opacity-60 disabled:hover:bg-transparent"
          >
            {background.stopping ? "Stopping…" : "Stop"}
          </button>
        </div>
      )}
      <div className="rounded-[var(--radius-lg)] border border-[var(--color-border-ai)] bg-[var(--color-surface-ai-2)] focus-within:border-[var(--color-border-focus)] transition-colors">
        <textarea
          ref={inputRef}
          rows={1}
          value={text}
          placeholder={placeholder}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey && !e.nativeEvent.isComposing) {
              e.preventDefault();
              void send();
            }
          }}
          className={`block w-full resize-none bg-transparent px-3 pt-3 pb-1 text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none ${wide ? "min-h-16" : ""}`}
        />
        <div className="flex items-center gap-1 px-1.5 pb-1.5 min-w-0">
          <ComposerPickers
            status={status}
            prefs={prefs}
            providerLocked={providerLocked}
            onPrefsChange={onPrefsChange}
            work={work}
            running={running}
            onLeaveWork={onLeaveWork}
            onOpenIssue={onOpenIssue}
          />
          {running ? (
            <button
              type="button"
              aria-label="Stop"
              onClick={onStop}
              className="ml-auto w-6 h-6 shrink-0 flex items-center justify-center rounded-full bg-[var(--color-surface-ai-3)] text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-ai-3)] transition-colors"
            >
              <Square className="w-2.5 h-2.5 fill-current" />
            </button>
          ) : (
            <button
              type="button"
              aria-label="Send"
              onClick={() => void send()}
              disabled={!canSend}
              className="ml-auto w-6 h-6 shrink-0 flex items-center justify-center rounded-full bg-[var(--color-button-ai)] text-white hover:bg-[var(--color-button-ai-hover)] disabled:opacity-40 disabled:hover:bg-[var(--color-button-ai)] transition-colors"
            >
              <ArrowUp className="w-3.5 h-3.5" />
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
