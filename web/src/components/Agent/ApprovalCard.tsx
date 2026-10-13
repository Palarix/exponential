import { useState } from "react";
import { decideApproval, type ApprovalDecision } from "../../api/chat";
import { ApiError } from "../../api/client";
import { IssueRefText } from "../IssueRef/IssueRef";
import { approvalArgs, approvalTitle, toolLabel, type StepApproval, type ToolStep } from "./chat-thread-utils";

function Spinner() {
  return <span className="inline-block w-3 h-3 border-[1.5px] border-current border-t-transparent rounded-full animate-spin" />;
}

// Footer buttons share the composer's control size.
const BUTTON = "flex items-center justify-center gap-1.5 h-7 px-2.5 rounded-[var(--radius-md)] text-xs font-medium transition-colors disabled:opacity-50 disabled:pointer-events-none";
const BUTTON_STYLES: Record<ApprovalDecision, string> = {
  deny: "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-3)]",
  allow_session: "border border-[var(--color-border-default)] text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-3)]",
  allow: "bg-[var(--color-accent-primary)] text-white hover:bg-[var(--color-accent-primary-hover)]",
};
const BUTTON_LABELS: Record<ApprovalDecision, string> = { deny: "Deny", allow_session: "Allow for this chat", allow: "Allow" };

/**
 * An agent asks before a tool call runs. The card stays until the decision
 * arrives on the chat stream. onExpired runs when the server no longer has
 * the request open.
 */
export default function ApprovalCard({
  chatId,
  step,
  onExpired,
}: {
  chatId: string;
  step: ToolStep & { approval: StepApproval };
  onExpired?: (approvalId: string) => void;
}) {
  const [sending, setSending] = useState<ApprovalDecision | null>(null);
  const [error, setError] = useState<string | null>(null);

  const decide = (decision: ApprovalDecision) => {
    setSending(decision);
    setError(null);
    decideApproval(chatId, step.approval.id, decision)
      .catch((err: unknown) => {
        if (err instanceof ApiError && err.status === 404) {
          onExpired?.(step.approval.id);
          return;
        }
        setError(err instanceof Error ? err.message : "Could not send the decision");
      })
      .finally(() => setSending(null));
  };

  const { action, detail } = approvalTitle(step.name, step.arg);
  const { command, fields } = approvalArgs(step.name, step.approval.input, step.arg);
  const buttons = (
    <div className="ml-auto flex shrink-0 items-center gap-1">
      {(["deny", "allow_session", "allow"] as const).map((d) => (
        <button key={d} type="button" disabled={sending !== null} onClick={() => decide(d)} className={`${BUTTON} ${BUTTON_STYLES[d]}`}>
          {sending === d && <Spinner />}
          {BUTTON_LABELS[d]}
        </button>
      ))}
    </div>
  );
  const extras = command || fields.length > 0;
  return (
    <div role="group" aria-label="Approval needed" className="rounded-[var(--radius-lg)] border border-[var(--color-border-default)] bg-[var(--color-surface-2)]">
      {/* The headline and the buttons share a row; the buttons wrap below a long headline. */}
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5 pl-3 pr-1.5 py-1.5 min-w-0">
        <div className="min-w-0 py-0.5 text-sm text-[var(--color-text-primary)] break-words">
          {action}
          {detail && (
            <>
              {" "}
              <IssueRefText text={detail} />
            </>
          )}
          <span className="ml-1.5 text-xs text-[var(--color-text-muted)]">
            via <span className="font-mono text-[11px]">{toolLabel(step.name)}</span>
          </span>
        </div>
        {buttons}
      </div>
      {(extras || error) && (
        <div className="px-3 pb-2.5 min-w-0">
          {command && (
            <pre className="max-h-40 overflow-auto px-2.5 py-1.5 rounded-[var(--radius-md)] bg-[var(--color-surface-1)] border border-[var(--color-border-subtle)] text-xs font-mono whitespace-pre-wrap break-words text-[var(--color-text-primary)]">
              {command}
            </pre>
          )}
          {fields.length > 0 && (
            <dl className={`grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5 text-xs min-w-0 ${command ? "mt-1.5" : ""}`}>
              {fields.map(([key, value]) => (
                <div key={key} className="contents">
                  <dt className="text-[var(--color-text-muted)]">{key}</dt>
                  <dd className="min-w-0 text-[var(--color-text-secondary)] break-words line-clamp-3" title={value}>
                    {value}
                  </dd>
                </div>
              ))}
            </dl>
          )}
          {error && <div className={`text-xs text-[var(--color-error)] ${extras ? "mt-1.5" : ""}`}>{error}</div>}
        </div>
      )}
    </div>
  );
}
