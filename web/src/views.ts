export type ViewId =
  | "dashboard"
  | "inbox"
  | "backlog"
  | "board"
  | "cycles"
  | "dependencies"
  | "labels"
  | "my-issues"
  | "timeline";

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
  /** Hidden from the sidebar unless the feature is enabled. */
  requires?: "cycles";
}

/** The single source of view metadata, in sidebar order. */
export const VIEWS: readonly ViewDef[] = [
  { id: "dashboard", route: "dashboard", title: "Dashboard", navLabel: "Overview", goKey: "o", section: "main" },
  { id: "backlog", route: "issues", title: "Issues", navLabel: "Issues", goKey: "i", section: "main" },
  { id: "board", route: "board", title: "Board", navLabel: "Board", goKey: "b", section: "main" },
  { id: "cycles", route: "cycles", title: "Cycles", navLabel: "Cycles", goKey: "c", section: "main", requires: "cycles" },
  { id: "timeline", route: "timeline", title: "Timeline", navLabel: "Timeline", goKey: "t", section: "main" },
  { id: "labels", route: "labels", title: "Labels", navLabel: "Labels", goKey: "l", section: "main" },
  { id: "dependencies", route: "dependencies", title: "Dependencies", navLabel: "Dependencies", goKey: "d", section: "main" },
  { id: "my-issues", route: "my-issues", title: "My Issues", navLabel: "My Issues", goKey: "m", section: "personal" },
  { id: "inbox", route: "inbox", title: "Inbox", navLabel: "Notifications", goKey: "n", section: "personal" },
];

export const VIEW_BY_ID = Object.fromEntries(VIEWS.map((v) => [v.id, v])) as Record<ViewId, ViewDef>;
export const VIEW_BY_GO_KEY: Record<string, ViewId> = Object.fromEntries(VIEWS.map((v) => [v.goKey, v.id]));
const VIEW_BY_ROUTE: Record<string, ViewId> = Object.fromEntries(VIEWS.map((v) => [v.route, v.id]));

export interface Route {
  view: ViewId;
  issueId: string | null;
  cycleId: string | null;
  depFocusId: string | null;
}

export function parseRoute(hash: string): Route {
  const [head, param] = hash.replace(/^#\/?/, "").split("/");
  const empty: Route = { view: "dashboard", issueId: null, cycleId: null, depFocusId: null };
  if (head === "issues" && param) return { ...empty, view: "backlog", issueId: param };
  if (head === "cycles" && param) return { ...empty, view: "cycles", cycleId: param };
  if (head === "dependencies" && param) return { ...empty, view: "dependencies", depFocusId: param };
  return { ...empty, view: VIEW_BY_ROUTE[head] ?? "dashboard" };
}

export function formatRoute({ view, issueId, cycleId, depFocusId }: Route): string {
  if (issueId) return `#/issues/${issueId}`;
  if (view === "cycles" && cycleId) return `#/cycles/${cycleId}`;
  if (view === "dependencies" && depFocusId) return `#/dependencies/${depFocusId}`;
  return `#/${VIEW_BY_ID[view].route}`;
}
