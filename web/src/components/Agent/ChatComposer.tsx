import { useEffect, useRef, useState } from "react";
import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import { ArrowUp, CornerDownRight, FileText, Paperclip, Pencil, Square, X } from "lucide-react";
import type { AgentStatus, BackgroundTask } from "../../api/chat";
import type { ChatPrefs } from "./agent-state-utils";
import Tooltip from "../ui/Tooltip";
import ComposerPickers, { type WorkModeInfo } from "./ComposerPickers";
import { composerArrowUp, composerEscapeBlurs } from "./chat-thread-utils";
import { formatSize } from "./attachment-utils";
import { useComposerDraft, type ComposerDrafts, type PendingAttachment } from "./composer-drafts";
import type { ComposerQueue } from "./useChatSession";

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
  /**
   * Sends the message with its attachments; resolves false when it failed.
   * While a turn runs, the message waits in the queue. edit replaces the
   * chat's last message.
   */
  onSend(text: string, files: File[], opts?: { edit?: boolean }): Promise<boolean>;
  onStop(): void;
  /** The messages sent while the turn ran. */
  queue?: ComposerQueue;
  /** Loads the last message to edit it; undefined when it cannot be edited. ↑ in an empty input runs it. */
  onEditLast?(): void;
  /** The full-page variant: a centered column and a taller input. */
  wide?: boolean;
  /** Set in Work Mode. */
  work?: WorkModeInfo;
  onLeaveWork?(): void;
  onOpenIssue?(issueId: string): void;
  /** Every unsent draft; the composer shows and edits draftKey's. */
  drafts: ComposerDrafts;
  /** The chat's ID, or the new chat's key in this place (NEW_CHAT_POPUP or NEW_CHAT_VIEW). */
  draftKey: string;
  /** The input takes focus each time this changes, as when the last approval card is answered. */
  focusSignal?: number;
  /** Esc gives up focus, keeping the draft. The popup leaves Esc to minimize instead. */
  blurOnEscape?: boolean;
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

/** The composer's pending attachments: image thumbnails and file chips, each with a remove button. */
export function ComposerAttachments({ items, onRemove }: { items: PendingAttachment[]; onRemove(key: string): void }) {
  return (
    <div className="flex flex-wrap gap-1.5 px-2 pt-2">
      {items.map(({ key, file, preview }) => (
        <div key={key} title={file.name} className="relative group flex items-center gap-1.5 max-w-56 h-10 pl-1 pr-6 rounded-[var(--radius-md)] bg-[var(--color-surface-ai-3)] text-xs">
          {preview ? (
            <img src={preview} alt="" className="w-8 h-8 shrink-0 rounded-[var(--radius-sm)] object-cover" />
          ) : (
            <span className="w-8 h-8 shrink-0 flex items-center justify-center text-[var(--color-text-muted)]">
              <FileText className="w-4 h-4" aria-hidden />
            </span>
          )}
          <span className="min-w-0 flex flex-col leading-tight">
            <span className="truncate text-[var(--color-text-primary)]">{file.name}</span>
            <span className="text-[var(--color-text-muted)]">{formatSize(file.size)}</span>
          </span>
          <button
            type="button"
            aria-label={`Remove ${file.name}`}
            onClick={() => onRemove(key)}
            className="absolute top-1 right-1 p-0.5 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-ai-3)]"
          >
            <X className="w-3 h-3" />
          </button>
        </div>
      ))}
    </div>
  );
}

/** The queued messages, one chip each: Edit puts one back in the input, Steer sends it into the running turn. */
function QueuedMessages({ queue, inputEmpty }: { queue: ComposerQueue; inputEmpty: boolean }) {
  const chipButton = "shrink-0 p-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-ai-1)] disabled:opacity-40 disabled:hover:bg-transparent disabled:hover:text-[var(--color-text-muted)]";
  return (
    <ul aria-label="Queued messages" className="flex flex-col gap-1 mb-1">
      {queue.items.map((m) => (
        <li key={m.id} data-queued-message className="flex items-center gap-1 pl-3 pr-1 py-1 rounded-[var(--radius-md)] bg-[var(--color-surface-ai-2)] border border-dashed border-[var(--color-border-ai)] text-xs min-w-0">
          <span className="min-w-0 flex-1 truncate text-[var(--color-text-secondary)]" title={m.text}>
            {m.text || m.attachments?.map((a) => a.name ?? "attachment").join(", ")}
          </span>
          {!!m.attachments?.length && m.text && (
            <span className="shrink-0 text-[var(--color-text-muted)]">
              {m.attachments.length} {m.attachments.length === 1 ? "file" : "files"}
            </span>
          )}
          <Tooltip content={inputEmpty ? "Edit" : "Clear the input first"}>
            <button type="button" aria-label="Edit queued message" disabled={!inputEmpty} onClick={() => queue.edit(m.id)} className={chipButton}>
              <Pencil className="w-3 h-3" />
            </button>
          </Tooltip>
          <Tooltip content="Steer: send it into the running turn">
            <button type="button" aria-label="Steer: send it into the running turn" disabled={!queue.canSteer} onClick={() => queue.steer(m.id)} className={chipButton}>
              <CornerDownRight className="w-3 h-3" />
            </button>
          </Tooltip>
          <Tooltip content="Remove">
            <button type="button" aria-label="Remove queued message" onClick={() => queue.remove(m.id)} className={chipButton}>
              <X className="w-3 h-3" />
            </button>
          </Tooltip>
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
  queue,
  onEditLast,
  wide,
  work,
  onLeaveWork,
  onOpenIssue,
  drafts,
  draftKey,
  focusSignal,
  blurOnEscape,
}: ChatComposerProps) {
  const { text, attachments, prefilled, editing } = useComposerDraft(drafts, draftKey);
  const [sending, setSending] = useState(false);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const fileRef = useRef<HTMLInputElement>(null);
  const [attachError, setAttachError] = useState<string | undefined>();
  const [dragging, setDragging] = useState(false);

  // The attach error belongs to the draft it came up in.
  const [errorKey, setErrorKey] = useState(draftKey);
  if (draftKey !== errorKey) {
    setErrorKey(draftKey);
    setAttachError(undefined);
  }
  const addFiles = (files: File[]) => {
    if (files.length === 0) return;
    setAttachError(drafts.addFiles(draftKey, files));
  };
  const removeAttachments = (keys: Set<string>) => {
    drafts.removeAttachments(draftKey, keys);
    setAttachError(undefined);
  };

  useEffect(() => inputRef.current?.focus(), []);
  useEffect(() => {
    if (focusSignal) inputRef.current?.focus();
  }, [focusSignal]);

  // Another chat's draft, or this one's in a new place: the caret goes to its end.
  useEffect(() => {
    const el = inputRef.current;
    el?.setSelectionRange(el.value.length, el.value.length);
  }, [draftKey]);

  // A prefill fills the input but sends nothing: the user can add to it first.
  useEffect(() => {
    if (!prefilled) return;
    const el = inputRef.current;
    el?.focus();
    el?.setSelectionRange(el.value.length, el.value.length);
    drafts.prefillUsed(draftKey);
  }, [prefilled, draftKey, drafts]);

  // Grow with the text, up to about eight lines.
  useEffect(() => {
    const el = inputRef.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = `${Math.min(el.scrollHeight, 180)}px`;
  }, [text]);

  const hasContent = text.trim() !== "" || attachments.length > 0;
  const canSend = !sending && !disabled && hasContent;
  const send = async () => {
    if (!canSend) return;
    setSending(true);
    // The draft to clear is the one sent, even if the chat changes meanwhile.
    const from = draftKey;
    const sent = attachments;
    const ok = await onSend(text.trim(), sent.map((a) => a.file), editing ? { edit: true } : undefined);
    setSending(false);
    if (ok) {
      drafts.sent(from, new Set(sent.map((a) => a.key)));
      setAttachError(undefined);
    }
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
      {queue && queue.items.length > 0 && <QueuedMessages queue={queue} inputEmpty={!hasContent} />}
      {editing && (
        <div className="flex items-center gap-2 pl-3 pr-1 py-1 mb-1 text-xs min-w-0" role="status">
          <Pencil className="w-3 h-3 shrink-0 text-[var(--color-text-muted)]" aria-hidden />
          <span className="min-w-0 truncate text-[var(--color-text-secondary)]">Editing your last message. Sending replaces it and its reply.</span>
          <button
            type="button"
            aria-label="Cancel editing"
            onClick={() => {
              drafts.cancelEdit(draftKey);
              inputRef.current?.focus();
            }}
            className="ml-auto shrink-0 px-2 py-0.5 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-ai-1)]"
          >
            Cancel
          </button>
        </div>
      )}
      <div
        onDragOver={(e) => {
          if (!e.dataTransfer.types.includes("Files")) return;
          e.preventDefault();
          setDragging(true);
        }}
        onDragLeave={(e) => {
          if (!e.currentTarget.contains(e.relatedTarget as Node | null)) setDragging(false);
        }}
        onDrop={(e) => {
          if (e.dataTransfer.files.length === 0) return;
          e.preventDefault();
          setDragging(false);
          addFiles(Array.from(e.dataTransfer.files));
          inputRef.current?.focus();
        }}
        className={`rounded-[var(--radius-lg)] border bg-[var(--color-surface-ai-2)] focus-within:border-[var(--color-border-focus)] transition-colors ${dragging ? "border-[var(--color-border-focus)]" : "border-[var(--color-border-ai)]"}`}
      >
        {attachments.length > 0 && <ComposerAttachments items={attachments} onRemove={(key) => removeAttachments(new Set([key]))} />}
        {attachError && (
          <p role="alert" className="px-3 pt-2 text-xs text-[var(--color-error)]">
            {attachError}
          </p>
        )}
        <textarea
          ref={inputRef}
          rows={1}
          value={text}
          placeholder={placeholder}
          onChange={(e) => drafts.setText(draftKey, e.target.value)}
          onPaste={(e) => {
            // Files on the clipboard become attachments; text pastes as text.
            const files = Array.from(e.clipboardData.files);
            if (files.length === 0) return;
            e.preventDefault();
            addFiles(files);
          }}
          onKeyDown={(e) => {
            const up = composerArrowUp({
              key: e.key,
              empty: !hasContent,
              composing: e.nativeEvent.isComposing,
              modified: e.shiftKey || e.altKey || e.metaKey || e.ctrlKey,
              queued: !!queue?.items.length,
              canEdit: !!onEditLast && !running,
            });
            if (up) {
              e.preventDefault();
              if (up === "pull" && queue) queue.edit(queue.items[queue.items.length - 1].id);
              else onEditLast?.();
            } else if (e.key === "Enter" && !e.shiftKey && !e.nativeEvent.isComposing) {
              e.preventDefault();
              void send();
            } else if (editing && e.key === "Escape" && !e.nativeEvent.isComposing) {
              // Esc ends edit mode first; the popup stays open.
              e.preventDefault();
              e.stopPropagation();
              drafts.cancelEdit(draftKey);
            } else if (composerEscapeBlurs({ key: e.key, enabled: !!blurOnEscape, composing: e.nativeEvent.isComposing, handled: e.defaultPrevented })) {
              inputRef.current?.blur();
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
          <input
            ref={fileRef}
            type="file"
            multiple
            hidden
            onChange={(e) => {
              addFiles(Array.from(e.target.files ?? []));
              e.target.value = "";
              inputRef.current?.focus();
            }}
          />
          <Tooltip content="Attach files">
            <button
              type="button"
              aria-label="Attach files"
              onClick={() => fileRef.current?.click()}
              disabled={disabled}
              className="ml-auto w-6 h-6 shrink-0 flex items-center justify-center rounded-full text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-ai-1)] disabled:opacity-40 disabled:hover:bg-transparent transition-colors"
            >
              <Paperclip className="w-3.5 h-3.5" />
            </button>
          </Tooltip>
          {running && hasContent && (
            <Tooltip content="Queue: send when the reply ends">
              <button
                type="button"
                aria-label="Queue message"
                onClick={() => void send()}
                disabled={!canSend}
                className="w-6 h-6 shrink-0 flex items-center justify-center rounded-full bg-[var(--color-button-ai)] text-white hover:bg-[var(--color-button-ai-hover)] disabled:opacity-40 disabled:hover:bg-[var(--color-button-ai)] transition-colors"
              >
                <ArrowUp className="w-3.5 h-3.5" />
              </button>
            </Tooltip>
          )}
          {running ? (
            <button
              type="button"
              aria-label="Stop"
              onClick={onStop}
              className="w-6 h-6 shrink-0 flex items-center justify-center rounded-full bg-[var(--color-surface-ai-3)] text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-ai-3)] transition-colors"
            >
              <Square className="w-2.5 h-2.5 fill-current" />
            </button>
          ) : (
            <button
              type="button"
              aria-label="Send"
              onClick={() => void send()}
              disabled={!canSend}
              className="w-6 h-6 shrink-0 flex items-center justify-center rounded-full bg-[var(--color-button-ai)] text-white hover:bg-[var(--color-button-ai-hover)] disabled:opacity-40 disabled:hover:bg-[var(--color-button-ai)] transition-colors"
            >
              <ArrowUp className="w-3.5 h-3.5" />
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
