import type { Milestone } from "../../api/types";
import type { MoveOption } from "./milestones-utils";

export type MilestoneAction = "complete" | "delete";

interface MilestoneDialogCopy {
  title: string;
  /** The move picker's label; undefined when no issue would move, so there is no picker. */
  pickerLabel?: string;
  confirm: string;
}

/**
 * The title, picker label and button of the complete and delete dialogs.
 * `count` is what would move: the open issues for a complete (done ones
 * stay), all issues for a delete.
 */
export function milestoneDialogCopy(action: MilestoneAction, count: number): MilestoneDialogCopy {
  const complete = action === "complete";
  return {
    title: complete ? "Complete milestone?" : "Delete milestone?",
    pickerLabel: count > 0 ? (complete ? "Move open issues to" : "Move issues to") : undefined,
    confirm: complete ? "Complete milestone" : "Delete milestone",
  };
}

/** A run of help text; `strong` names the target in bold. */
export type HelpPart = string | { strong: string };

/**
 * The intro above the picker: the question, naming the milestone (with its
 * goal) in bold, and what the picker decides.
 */
export function milestoneIntro(action: MilestoneAction, milestone: Milestone, count: number): HelpPart[] {
  const name = milestone.goal ? `${milestone.name} – ${milestone.goal}` : milestone.name;
  const question: HelpPart[] = [`Do you really want to ${action} `, { strong: name }, "?"];
  if (count === 0) return question;
  const open = action === "complete" ? "open " : "";
  const issues = count === 1 ? `the ${open}issue` : `the ${count} ${open}issues`;
  return [...question, ` Decide what happens to ${issues} currently assigned to this milestone.`];
}

/**
 * The help under the picker: how many issues go where, whether done issues
 * move, and that the action can't be undone. `option` is the picked move
 * option, absent when nothing moves.
 */
export function moveHelp(action: MilestoneAction, milestone: Milestone, count: number, option?: MoveOption): HelpPart[] {
  const where = (subject: string, tail = ""): HelpPart[] => {
    if (option?.value === "none") return [`${subject} will be taken out of the milestone${tail}. `];
    return [`${subject} will move to `, { strong: option?.value === "new" ? "a new milestone" : option?.label ?? "" }, `${tail}. `];
  };
  if (action === "complete") {
    const final = `Done issues stay with ${milestone.name}. Completing is final: ${milestone.name} moves to Past and can't be reopened.`;
    if (count === 0 || !option) return [final];
    return [...where(`${count} open ${count === 1 ? "issue" : "issues"}`), final];
  }
  const undo = "This can't be undone.";
  if (count === 0 || !option) return [undo];
  const moved = count === 1 ? where("The issue", ", even if it's done") : where(`All ${count} issues, including done ones,`);
  return [...moved, undo];
}

/** The `⋯` menu items of a saved open milestone, in order. Completing is the main action of Current, so it comes first there. */
export function milestoneMenuItems(milestone: Milestone): (MilestoneAction | "edit")[] {
  return milestone.slot === "current" ? ["complete", "edit", "delete"] : ["edit", "complete", "delete"];
}
