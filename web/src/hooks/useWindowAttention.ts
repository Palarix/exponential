import { useSyncExternalStore } from "react";

function subscribe(onChange: () => void): () => void {
  window.addEventListener("focus", onChange);
  window.addEventListener("blur", onChange);
  document.addEventListener("visibilitychange", onChange);
  return () => {
    window.removeEventListener("focus", onChange);
    window.removeEventListener("blur", onChange);
    document.removeEventListener("visibilitychange", onChange);
  };
}

function hasAttention(): boolean {
  return document.visibilityState === "visible" && document.hasFocus();
}

/**
 * Whether the user can be looking at the page: its tab is visible and its
 * window has focus. Another focused window, such as a terminal, means no.
 */
export function useWindowAttention(): boolean {
  return useSyncExternalStore(subscribe, hasAttention, () => true);
}
