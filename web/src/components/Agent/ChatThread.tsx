import { useEffect, useRef, useState } from "react";
import Markdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import { ChevronRight, Copy, Check } from "lucide-react";
import { useConfig } from "../../api/queries";
import { linkifyIssueIds } from "../../utils/format";
import { splitTurn, turnSeconds, type ChatStep, type ChatThread as Thread, type ChatTurn } from "./chat-thread-utils";

/** "mcp__xpo__list" reads as "xpo list"; built-in tools keep their name. */
function toolLabel(name: string): string {
  const parts = name.split("__");
  return parts.length === 3 && parts[0] === "mcp" ? `${parts[1]} ${parts[2]}` : name;
}

function Spinner() {
  return <span className="inline-block w-3 h-3 border-[1.5px] border-[var(--color-text-muted)] border-t-transparent rounded-full animate-spin" />;
}

function useNow(active: boolean): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!active) return;
    const timer = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(timer);
  }, [active]);
  return now;
}

function AnswerText({ text }: { text: string }) {
  const { prefix } = useConfig();
  return (
    <div className="prose-exponential prose-activity">
      <Markdown remarkPlugins={[remarkGfm, remarkBreaks]}>{linkifyIssueIds(text, prefix)}</Markdown>
    </div>
  );
}

function WorkStep({ step }: { step: ChatStep }) {
  if (step.kind === "text") {
    return <div className="text-xs text-[var(--color-text-muted)] whitespace-pre-wrap">{step.text}</div>;
  }
  return (
    <div className="flex items-center gap-2 text-xs min-w-0">
      {step.done ? <span className="w-3 text-center text-[var(--color-text-muted)]">·</span> : <Spinner />}
      <span className="text-[var(--color-text-secondary)] shrink-0">{toolLabel(step.name)}</span>
      {step.arg && <span className="font-mono text-[var(--color-text-muted)] truncate">{step.arg}</span>}
    </div>
  );
}

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <button
      type="button"
      aria-label="Copy answer"
      onClick={() => {
        navigator.clipboard?.writeText(text).then(() => {
          setCopied(true);
          setTimeout(() => setCopied(false), 1500);
        }).catch(() => {});
      }}
      className="p-1 -ml-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-3)] transition-colors"
    >
      {copied ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
    </button>
  );
}

function Turn({ turn }: { turn: ChatTurn }) {
  const [expanded, setExpanded] = useState(false);
  const running = turn.status === "running";
  const now = useNow(running);
  const { work, answer } = splitTurn(turn);
  const answerText = answer.map((s) => (s.kind === "text" ? s.text : "")).join("\n\n").trim();
  const seconds = turnSeconds(turn, now);
  const showWork = work.length > 0 || (running && answer.length === 0);

  return (
    <div className="flex flex-col gap-3">
      <div className="self-end max-w-[85%] px-3 py-2 rounded-[var(--radius-lg)] bg-[var(--color-surface-3)] text-sm text-[var(--color-text-primary)] whitespace-pre-wrap break-words">
        {turn.user.text}
      </div>

      {showWork && (
        <div>
          <button
            type="button"
            onClick={() => setExpanded((e) => !e)}
            disabled={work.length === 0}
            className="flex items-center gap-1.5 text-xs text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] disabled:hover:text-[var(--color-text-muted)] transition-colors"
          >
            {running && <Spinner />}
            <span>
              {running ? (turn.thinking ? "Thinking" : "Working") + ` for ${seconds}s` : `Worked for ${seconds} second${seconds === 1 ? "" : "s"}`}
            </span>
            {work.length > 0 && <ChevronRight className={`w-3 h-3 transition-transform ${expanded ? "rotate-90" : ""}`} />}
          </button>
          {expanded && (
            <div className="mt-2 ml-1 pl-3 border-l border-[var(--color-border-subtle)] flex flex-col gap-1.5">
              {work.map((step, i) => <WorkStep key={i} step={step} />)}
            </div>
          )}
        </div>
      )}

      {answerText && (
        <div className="flex flex-col gap-1">
          <AnswerText text={answerText} />
          {!running && <div><CopyButton text={answerText} /></div>}
        </div>
      )}

      {turn.status === "stopped" && <div className="text-xs text-[var(--color-text-muted)]">Stopped</div>}
      {turn.status === "interrupted" && (
        <div className="text-xs text-[var(--color-text-muted)]">Interrupted. Send a message to continue.</div>
      )}
      {turn.status === "error" && (
        <div className="px-3 py-2 rounded-[var(--radius-md)] bg-[var(--color-error-bg)] text-xs text-[var(--color-error)] whitespace-pre-wrap break-words">
          {turn.error}
        </div>
      )}
    </div>
  );
}

function dayLabel(iso: string): string {
  const d = new Date(iso);
  const today = new Date();
  const time = d.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });
  return d.toDateString() === today.toDateString() ? `Today ${time}` : `${d.toLocaleDateString()} ${time}`;
}

export default function ChatThread({ thread }: { thread: Thread }) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const stickRef = useRef(true);

  // Follow new output while the user is at the bottom.
  useEffect(() => {
    const el = scrollRef.current;
    if (el && stickRef.current) el.scrollTop = el.scrollHeight;
  }, [thread]);

  const first = thread.turns[0];
  return (
    <div
      ref={scrollRef}
      onScroll={(e) => {
        const el = e.currentTarget;
        stickRef.current = el.scrollHeight - el.scrollTop - el.clientHeight < 40;
      }}
      className="flex-1 overflow-y-auto px-4 py-3"
    >
      {first && <div className="text-center text-xs text-[var(--color-text-muted)] mb-4">{dayLabel(first.user.time)}</div>}
      <div className="flex flex-col gap-6">
        {thread.turns.map((turn, i) => <Turn key={i} turn={turn} />)}
      </div>
    </div>
  );
}
