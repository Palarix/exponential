import { createContext, useContext } from "react";
import type { Release } from "../../api/types";

/** Every way to put issues into a release goes through here, so all of them ask about children alike. */
export interface ReleaseAssign {
  /**
   * Moves `issueIds` to `target` (null is No release). When they have open
   * children not in the target, asks first whether to move those too.
   */
  assign(issueIds: string[], target: Release | null): void;
  /** Opens the release picker for `issueIds`, centred on the screen. */
  openPicker(issueIds: string[]): void;
}

const noop: ReleaseAssign = { assign: () => {}, openPicker: () => {} };

/** Provided by `ReleaseAssignProvider`; a no-op outside it (e.g. in markup tests). */
export const ReleaseAssignContext = createContext<ReleaseAssign>(noop);

export function useReleaseAssign(): ReleaseAssign {
  return useContext(ReleaseAssignContext);
}
