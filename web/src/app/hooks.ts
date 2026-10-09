import { useCallback, useContext, useEffect, useRef, useSyncExternalStore } from "react";
import type { Issue } from "../api/types";
import type { ViewId } from "../views";
import { AppNavContext, ViewStateContext, type NavIntent } from "./contexts";
import type { Codec } from "./view-state-utils";

export function useAppNav() {
  const nav = useContext(AppNavContext);
  if (!nav) throw new Error("useAppNav requires AppNavContext");
  return nav;
}

/**
 * View-local state that survives the view unmounting (e.g. while IssueDetail
 * is open). Pass an initializer for session-only state, or a codec to also
 * persist it to localStorage under `key`; a codec must be referentially stable
 * (define it at module level).
 */
export function useViewState<T>(key: string, source: (() => T) | Codec<T>): [T, (value: T) => void] {
  const store = useContext(ViewStateContext);
  if (!store) throw new Error("useViewState requires ViewStateContext");
  const codec = typeof source === "function" ? undefined : source;
  const sourceRef = useRef(source);
  useEffect(() => { sourceRef.current = source; });
  const value = useSyncExternalStore(store.subscribe, () => {
    const s = sourceRef.current;
    return typeof s === "function" ? store.get(key, s) : store.get(key, () => s.parse(null), s);
  });
  const setValue = useCallback((next: T) => store.set(key, next, codec), [store, key, codec]);
  return [value, setValue];
}

/** A stable `onIssueClick` handler that opens IssueDetail. */
export function useIssueClick(): (issue: Issue) => void {
  const { openIssue } = useAppNav();
  return useCallback((issue: Issue) => openIssue(issue.id), [openIssue]);
}

/** Publishes the view's visible issue order for IssueDetail's prev/next. Pass a memoized array. */
export function useIssueNavOrder(ids: string[]): void {
  const { publishNavOrder } = useAppNav();
  useEffect(() => publishNavOrder(ids), [publishNavOrder, ids]);
}

/** Applies the pending navigation intent for `view`, once, after mount. */
export function useNavIntent(view: ViewId, apply: (intent: NavIntent) => void): void {
  const { takeIntent } = useAppNav();
  const applyRef = useRef(apply);
  useEffect(() => { applyRef.current = apply; });
  useEffect(() => {
    const intent = takeIntent(view);
    if (intent) applyRef.current(intent);
  }, [takeIntent, view]);
}
