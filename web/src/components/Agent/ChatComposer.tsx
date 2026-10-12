import { useEffect, useRef, useState } from "react";
import { ArrowUp, Square, X } from "lucide-react";
import { CHAT_EFFORTS, CHAT_MODELS } from "../../api/chat";
import { InlineDropdown } from "../ui";
import { EFFORT_LABELS, MODEL_LABELS, type ChatPrefs } from "./agent-state-utils";

const modelOptions = CHAT_MODELS.map((m) => ({ value: m, label: MODEL_LABELS[m] }));
const effortOptions = CHAT_EFFORTS.map((e) => ({ value: e, label: EFFORT_LABELS[e] }));

interface ChatComposerProps {
  running: boolean;
  disabled?: boolean;
  placeholder: string;
  /** The open issue, shown as a removable chip; undefined hides it. */
  contextIssue?: { id: string; title?: string };
  onDismissContext(): void;
  prefs: ChatPrefs;
  onPrefsChange(prefs: ChatPrefs): void;
  onSend(text: string): Promise<boolean>;
  onStop(): void;
}

export default function ChatComposer({
  running,
  disabled,
  placeholder,
  contextIssue,
  onDismissContext,
  prefs,
  onPrefsChange,
  onSend,
  onStop,
}: ChatComposerProps) {
  const [text, setText] = useState("");
  const [sending, setSending] = useState(false);
  const inputRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => inputRef.current?.focus(), []);

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
    <div className="px-3 pb-3">
      {contextIssue && (
        <div className="flex items-center gap-2 px-3 py-1.5 mb-1 text-xs min-w-0">
          <span className="font-mono text-[var(--color-text-muted)] shrink-0">{contextIssue.id}</span>
          <span className="text-[var(--color-text-secondary)] truncate">{contextIssue.title}</span>
          <button
            type="button"
            aria-label="Remove issue context"
            onClick={onDismissContext}
            className="ml-auto shrink-0 p-0.5 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-3)]"
          >
            <X className="w-3 h-3" />
          </button>
        </div>
      )}
      <div className="rounded-[var(--radius-lg)] border border-[var(--color-border-default)] bg-[var(--color-surface-2)] focus-within:border-[var(--color-border-focus)] transition-colors">
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
          className="block w-full resize-none bg-transparent px-3 pt-3 pb-1 text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none"
        />
        <div className="flex items-center gap-1 px-1.5 pb-1.5">
          <InlineDropdown
            borderless
            placeholder="Model"
            options={modelOptions}
            value={prefs.model}
            onChange={(v) => onPrefsChange({ ...prefs, model: v as ChatPrefs["model"] })}
          />
          <InlineDropdown
            borderless
            placeholder="Effort"
            options={effortOptions}
            value={prefs.effort}
            onChange={(v) => onPrefsChange({ ...prefs, effort: v as ChatPrefs["effort"] })}
          />
          {running ? (
            <button
              type="button"
              aria-label="Stop"
              onClick={onStop}
              className="ml-auto w-8 h-8 flex items-center justify-center rounded-full bg-[var(--color-surface-4)] text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-4)] transition-colors"
            >
              <Square className="w-3 h-3 fill-current" />
            </button>
          ) : (
            <button
              type="button"
              aria-label="Send"
              onClick={() => void send()}
              disabled={!canSend}
              className="ml-auto w-8 h-8 flex items-center justify-center rounded-full bg-[var(--color-accent-primary)] text-white hover:bg-[var(--color-accent-primary-hover)] disabled:opacity-40 disabled:hover:bg-[var(--color-accent-primary)] transition-colors"
            >
              <ArrowUp className="w-4 h-4" />
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
