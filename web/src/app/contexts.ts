import { createContext } from "react";
import type { ViewId } from "../views";
import type { BacklogFilters } from "../components/Backlog/filters";
import type { ViewStateStore } from "./view-state-utils";

/** One-shot payload handed to the target view of a `navigate` call. */
export interface NavIntent {
  /** Merged into the target view's filters for its active tab. */
  filters?: Partial<BacklogFilters>;
}

export interface AppNav {
  view: ViewId;
  navigate(view: ViewId, intent?: NavIntent): void;
  openIssue(issueId: string): void;
  closeIssue(): void;
  newIssue(): void;
  cycleId: string | null;
  setCycleId(cycleId: string | null): void;
  depFocusId: string | null;
  setDepFocusId(issueId: string | null): void;
  /** Sets the order IssueDetail's prev/next walks; `navigate` clears it. */
  publishNavOrder(ids: string[]): void;
  /** Returns and forgets the pending intent for `view`. */
  takeIntent(view: ViewId): NavIntent | undefined;
}

export const AppNavContext = createContext<AppNav | null>(null);
export const ViewStateContext = createContext<ViewStateStore | null>(null);
export const StatusBarSlotContext = createContext<HTMLElement | null>(null);
