import { useEffect, useState } from "react";

function now(): string | null {
  return document.visibilityState === "hidden" ? new Date().toISOString() : null;
}

/** When the tab was last hidden, as an ISO time; null while it is visible. */
export function useHiddenSince(): string | null {
  const [since, setSince] = useState(now);
  useEffect(() => {
    const onChange = () => setSince((prev) => (document.visibilityState === "hidden" ? prev ?? now() : null));
    document.addEventListener("visibilitychange", onChange);
    return () => document.removeEventListener("visibilitychange", onChange);
  }, []);
  return since;
}
