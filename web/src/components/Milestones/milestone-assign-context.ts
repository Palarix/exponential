import { createContext, useContext } from "react";
import type { Milestone } from "../../api/types";

/** Every way to put issues into a milestone goes through here, so all of them ask about children alike. */
export interface MilestoneAssign {
  /**
   * Moves `issueIds` to `target` (null is No milestone). When they have open
   * children not in the target, asks first whether to move those too.
   */
  assign(issueIds: string[], target: Milestone | null): void;
  /** Opens the milestone picker for `issueIds`, centred on the screen. */
  openPicker(issueIds: string[]): void;
}

const noop: MilestoneAssign = { assign: () => {}, openPicker: () => {} };

/** Provided by `MilestoneAssignProvider`; a no-op outside it (e.g. in markup tests). */
export const MilestoneAssignContext = createContext<MilestoneAssign>(noop);

export function useMilestoneAssign(): MilestoneAssign {
  return useContext(MilestoneAssignContext);
}
