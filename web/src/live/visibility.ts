/** Calls fn with the tab's visibility on each change; returns an unsubscribe. */
export function watchDocumentVisibility(fn: (visible: boolean) => void): () => void {
  const onChange = () => fn(document.visibilityState === "visible");
  document.addEventListener("visibilitychange", onChange);
  return () => document.removeEventListener("visibilitychange", onChange);
}

/**
 * Calls onVisible when the tab becomes visible after being hidden. Returns
 * an unsubscribe.
 */
export function onVisibleAgain(watch: (fn: (visible: boolean) => void) => () => void, onVisible: () => void): () => void {
  let hidden = false;
  return watch((visible) => {
    if (!visible) hidden = true;
    else if (hidden) {
      hidden = false;
      onVisible();
    }
  });
}
