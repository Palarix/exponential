import { createContext } from "react";
import type { ViewId } from "../views";
import type { BacklogFilters } from "../components/Backlog/filters";
import type { ViewStateStore } from "./view-state-utils";
import type { IssueDraft } from "../api/types";

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
  /** Opens the new-issue modal filled from an issue draft. */
  openDraft(draft: IssueDraft): void;
  cycleId: string | null;
  setCycleId(cycleId: string | null): void;
  depFocusId: string | null;
  setDepFocusId(issueId: string | null): void;
  /** The chat shown in the Agent view; null is a new chat. */
  chatId: string | null;
  /** Opens the Agent view on a chat, or on a new chat. */
  openChatView(chatId: string | null): void;
  /** Leaves the Agent view for the view the user was on before it. */
  leaveChatView(): void;
  /** Sets the order IssueDetail's prev/next walks; `navigate` clears it. */
  publishNavOrder(ids: string[]): void;
  /** Returns and forgets the pending intent for `view`. */
  takeIntent(view: ViewId): NavIntent | undefined;
}

export const AppNavContext = createContext<AppNav | null>(null);
export const ViewStateContext = createContext<ViewStateStore | null>(null);
export const StatusBarSlotContext = createContext<HTMLElement | null>(null);
