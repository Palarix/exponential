export type ViewId =
  | "dashboard"
  | "inbox"
  | "backlog"
  | "board"
  | "cycles"
  | "releases"
  | "dependencies"
  | "labels"
  | "my-issues"
  | "timeline"
  | "drafts"
  | "agent";

export interface ViewDef {
  id: ViewId;
  /** First hash segment, e.g. `#/issues`. */
  route: string;
  /** Used for `document.title` and the "Go to …" shortcut labels. */
  title: string;
  /** Used in the sidebar. */
  navLabel: string;
  /** Second key of the `g` chord. */
  goKey: string;
  /** Sidebar section; `personal` views render below the divider. */
  section: "main" | "personal";
  /** Hidden from the sidebar unless cycles are enabled, issue drafts exist, or agent chat is available. */
  requires?: "cycles" | "drafts" | "agent";
}

/** The single source of view metadata, in sidebar order. */
export const VIEWS: readonly ViewDef[] = [
  { id: "dashboard", route: "dashboard", title: "Dashboard", navLabel: "Overview", goKey: "o", section: "main" },
  { id: "backlog", route: "issues", title: "Issues", navLabel: "Issues", goKey: "i", section: "main" },
  { id: "board", route: "board", title: "Board", navLabel: "Board", goKey: "b", section: "main" },
  { id: "cycles", route: "cycles", title: "Cycles", navLabel: "Cycles", goKey: "c", section: "main", requires: "cycles" },
  { id: "releases", route: "releases", title: "Releases", navLabel: "Releases", goKey: "v", section: "main" },
  { id: "timeline", route: "timeline", title: "Timeline", navLabel: "Timeline", goKey: "t", section: "main" },
  { id: "labels", route: "labels", title: "Labels", navLabel: "Labels", goKey: "l", section: "main" },
  { id: "dependencies", route: "dependencies", title: "Dependencies", navLabel: "Dependencies", goKey: "d", section: "main" },
  { id: "agent", route: "agent", title: "Agent", navLabel: "Agent", goKey: "a", section: "personal", requires: "agent" },
  { id: "my-issues", route: "my-issues", title: "My Issues", navLabel: "My Issues", goKey: "m", section: "personal" },
  { id: "inbox", route: "inbox", title: "Inbox", navLabel: "Notifications", goKey: "n", section: "personal" },
  { id: "drafts", route: "drafts", title: "Drafts", navLabel: "Drafts", goKey: "r", section: "personal", requires: "drafts" },
];

export const VIEW_BY_ID = Object.fromEntries(VIEWS.map((v) => [v.id, v])) as Record<ViewId, ViewDef>;
export const VIEW_BY_GO_KEY: Record<string, ViewId> = Object.fromEntries(VIEWS.map((v) => [v.goKey, v.id]));
const VIEW_BY_ROUTE: Record<string, ViewId> = Object.fromEntries(VIEWS.map((v) => [v.route, v.id]));

export interface Route {
  view: ViewId;
  issueId: string | null;
  cycleId: string | null;
  depFocusId: string | null;
  chatId: string | null;
  /** The Agent view's chat history page, optionally filtered to one issue's chats. */
  chatHistory: { issueId: string | null } | null;
}

/** The Agent view segment of the chat history page; chat IDs are UUIDs, so it never clashes. */
const CHAT_HISTORY = "history";

export function parseRoute(hash: string): Route {
  const [path, query = ""] = hash.replace(/^#\/?/, "").split("?");
  const [head, param] = path.split("/");
  const empty: Route = { view: "dashboard", issueId: null, cycleId: null, depFocusId: null, chatId: null, chatHistory: null };
  if (head === "agent" && param === CHAT_HISTORY) {
    return { ...empty, view: "agent", chatHistory: { issueId: new URLSearchParams(query).get("issue") || null } };
  }
  if (head === "issues" && param) return { ...empty, view: "backlog", issueId: param };
  if (head === "cycles" && param) return { ...empty, view: "cycles", cycleId: param };
  if (head === "dependencies" && param) return { ...empty, view: "dependencies", depFocusId: param };
  if (head === "agent" && param) return { ...empty, view: "agent", chatId: param };
  return { ...empty, view: VIEW_BY_ROUTE[head] ?? "dashboard" };
}

/** The browser tab label for a route: the open issue, the chat history page, or the view. */
export function routeTitle(route: Route): string {
  if (route.issueId) return route.issueId;
  if (route.view === "agent" && route.chatHistory) return "All Chats";
  return VIEW_BY_ID[route.view].title;
}

export function formatRoute({ view, issueId, cycleId, depFocusId, chatId, chatHistory }: Route): string {
  if (issueId) return `#/issues/${issueId}`;
  if (view === "cycles" && cycleId) return `#/cycles/${cycleId}`;
  if (view === "dependencies" && depFocusId) return `#/dependencies/${depFocusId}`;
  if (view === "agent" && chatHistory) {
    return chatHistory.issueId ? `#/agent/${CHAT_HISTORY}?issue=${encodeURIComponent(chatHistory.issueId)}` : `#/agent/${CHAT_HISTORY}`;
  }
  if (view === "agent" && chatId) return `#/agent/${chatId}`;
  return `#/${VIEW_BY_ID[view].route}`;
}
