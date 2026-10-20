import { useCallback, useMemo, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import type { Issue, Release } from "../../api/types";
import { useIssueList, useReleases } from "../../api/queries";
import { useAssignRelease } from "../../api/mutations";
import { Modal, StatusIcon, useToast } from "../ui";
import ReleasePicker from "./ReleasePicker";
import { ReleaseAssignContext, type ReleaseAssign } from "./release-assign-context";
import { childrenPrompt, pickerOptions, planReleaseMoves, type ChildrenPrompt } from "./release-assign-utils";

const plural = (n: number, one: string, many: string) => `${n} ${n === 1 ? one : many}`;

/**
 * Owns release assignment for the whole app: the "Move sub-issues too?"
 * prompt and the centred picker that `r` and the command palette open.
 */
export default function ReleaseAssignProvider({ children }: { children: ReactNode }) {
  const issues = useIssueList();
  const { data: board } = useReleases();
  const { mutate } = useAssignRelease();
  const showToast = useToast();
  const [prompt, setPrompt] = useState<{ ids: string[]; target: Release | null; plan: ChildrenPrompt } | null>(null);
  const [pickerIds, setPickerIds] = useState<string[] | null>(null);

  const run = useCallback((ids: string[], target: Release | null, withChildren: boolean) => {
    const { moves, optimistic } = planReleaseMoves(issues, ids, target, withChildren);
    if (moves.length === 0) return;
    const message = target
      ? `Moved ${optimistic.length === 1 ? "to" : `${optimistic.length} issues to`} ${target.name}`
      : `Removed ${optimistic.length === 1 ? "from release" : `${optimistic.length} issues from their release`}`;
    mutate({ moves, optimistic }, { onSuccess: () => showToast(message) });
  }, [issues, mutate, showToast]);

  const assign = useCallback((ids: string[], target: Release | null) => {
    const plan = childrenPrompt(issues, ids, target);
    if (plan) setPrompt({ ids, target, plan });
    else run(ids, target, false);
  }, [issues, run]);

  const api = useMemo<ReleaseAssign>(() => ({ assign, openPicker: setPickerIds }), [assign]);

  const pickerCurrent = pickerIds?.length === 1 ? issues.find((i) => i.id === pickerIds[0])?.release_id : undefined;

  const decide = (withChildren: boolean) => {
    if (!prompt) return;
    setPrompt(null);
    run(prompt.ids, prompt.target, withChildren);
  };

  return (
    <ReleaseAssignContext.Provider value={api}>
      {children}
      {pickerIds && board && createPortal(
        <div className="fixed inset-0 z-[200] flex items-start justify-center pt-[20vh]">
          <div className="absolute inset-0 bg-black/30" onClick={() => setPickerIds(null)} aria-hidden="true" />
          <div className="relative w-72 bg-[var(--color-surface-3)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-popover)] animate-fade-in">
            <ReleasePicker
              options={pickerOptions(board, pickerCurrent)}
              current={pickerCurrent}
              onSelect={(release) => assign(pickerIds, release)}
              onClose={() => setPickerIds(null)}
            />
          </div>
        </div>,
        document.body,
      )}
      <Modal isOpen={!!prompt} onClose={() => setPrompt(null)} title="Move sub-issues too?" size="xl" showCloseButton={false}>
        {prompt && (
          <MoveChildrenBody
            prompt={prompt.plan}
            target={prompt.target}
            onAbort={() => setPrompt(null)}
            onJustThis={() => decide(false)}
            onMoveAll={() => decide(true)}
          />
        )}
      </Modal>
    </ReleaseAssignContext.Provider>
  );
}

/** The prompt's text, its sub-issues and the choices, as the status prompt shows them. */
export function MoveChildrenBody({ prompt, target, onAbort, onJustThis, onMoveAll }: {
  prompt: ChildrenPrompt;
  target: Release | null;
  onAbort: () => void;
  onJustThis: () => void;
  onMoveAll: () => void;
}) {
  const { epics, children, doneCount } = prompt;
  const owner = epics.length === 1 ? epics[0].title : `${epics.length} issues`;
  return (
    <>
      <p className="text-sm text-[var(--color-text-secondary)] mb-5 leading-6">
        <span className="text-[var(--color-text-primary)]">{owner}</span>{" "}
        {epics.length === 1 ? "has" : "have"} {plural(children.length, "open sub-issue", "open sub-issues")}{" "}
        {target ? (
          <>not in <strong className="text-[var(--color-text-primary)]">{target.name}</strong>. Move them to {target.name} too?</>
        ) : (
          <>in a release. Remove them from their release too?</>
        )}
        {doneCount > 0 && (
          <>{" "}{doneCount === 1 ? "1 completed issue stays where it is." : `${doneCount} completed issues stay where they are.`}</>
        )}
      </p>
      <div className="rounded-[var(--radius-md)] border border-[var(--color-border-default)] overflow-hidden mb-8 max-h-64 overflow-y-auto">
        {children.map((child: Issue, i) => (
          <div key={child.id} className={`flex items-center gap-3 px-4 py-2.5 text-sm ${i > 0 ? "border-t border-[var(--color-border-subtle)]" : ""}`}>
            <StatusIcon status={child.status} size={14} />
            <span className="text-[var(--color-text-primary)] truncate min-w-0 flex-1">{child.title}</span>
            {child.release && <span className="text-xs text-[var(--color-text-muted)] shrink-0">{child.release}</span>}
          </div>
        ))}
      </div>
      <div className="flex items-center gap-4">
        <button
          className="px-3 py-1.5 text-sm rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface)] transition-colors"
          onClick={onAbort}
        >
          Abort
        </button>
        <div className="flex-1" />
        <button
          className="px-3 py-1.5 text-sm rounded-[var(--radius-sm)] text-[var(--color-text-secondary)] hover:bg-[var(--color-hover-surface)] transition-colors"
          onClick={onJustThis}
        >
          {epics.length === 1 ? "Just this issue" : "Just these issues"}
        </button>
        <button
          className="px-3 py-1.5 text-sm rounded-[var(--radius-sm)] bg-[var(--color-accent-primary)] text-white hover:opacity-90 transition-colors"
          onClick={onMoveAll}
        >
          Move all
        </button>
      </div>
    </>
  );
}
