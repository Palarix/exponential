import { useSyncExternalStore } from "react";

/**
 * User preferences, kept in this browser. One localStorage key holds every
 * setting; `parsePreferences` defaults each field on its own, so a new setting
 * needs no migration.
 */

export type MotionPreference = "system" | "on" | "off";

/**
 * What the app does with animations, after the preference meets the OS:
 * - `full`: everything plays.
 * - `reduced`: movement becomes a plain fade; fades and status animations play.
 * - `off`: no movement and no fades; status animations still play.
 * The tiers themselves live in index.css, keyed on `<html data-motion>`.
 */
export type ResolvedMotion = "full" | "reduced" | "off";

/** How the Changes tab shows diffs. */
export type DiffView = "unified" | "split";

export interface Preferences {
  motion: MotionPreference;
  diffView: DiffView;
}

export const DEFAULT_PREFERENCES: Preferences = { motion: "system", diffView: "unified" };

const STORAGE_KEY = "exponential-preferences";
const MOTIONS: readonly MotionPreference[] = ["system", "on", "off"];
const DIFF_VIEWS: readonly DiffView[] = ["unified", "split"];
const REDUCED_QUERY = "(prefers-reduced-motion: reduce)";

export function parsePreferences(raw: string | null): Preferences {
  let data: unknown = null;
  try { data = raw ? JSON.parse(raw) : null; } catch { /* bad JSON: defaults */ }
  if (!data || typeof data !== "object" || Array.isArray(data)) return DEFAULT_PREFERENCES;
  const { motion, diffView } = data as Record<string, unknown>;
  return {
    motion: oneOf(MOTIONS, motion, DEFAULT_PREFERENCES.motion),
    diffView: oneOf(DIFF_VIEWS, diffView, DEFAULT_PREFERENCES.diffView),
  };
}

function oneOf<T extends string>(allowed: readonly T[], value: unknown, fallback: T): T {
  return allowed.includes(value as T) ? (value as T) : fallback;
}

export function resolveMotion(pref: MotionPreference, systemReduced: boolean): ResolvedMotion {
  if (pref === "on") return "full";
  if (pref === "off") return "off";
  return systemReduced ? "reduced" : "full";
}

function storage(): Storage | null {
  try { return window.localStorage; } catch { return null; }
}

function readStored(): Preferences {
  try { return parsePreferences(storage()?.getItem(STORAGE_KEY) ?? null); } catch { return DEFAULT_PREFERENCES; }
}

// Read lazily: tests import this module without a window.
let current: Preferences | null = null;
const listeners = new Set<() => void>();

function emit(): void {
  for (const l of listeners) l();
}

export function getPreferences(): Preferences {
  current ??= readStored();
  return current;
}

export function setPreference<K extends keyof Preferences>(key: K, value: Preferences[K]): void {
  current = { ...getPreferences(), [key]: value };
  try { storage()?.setItem(STORAGE_KEY, JSON.stringify(current)); } catch { /* blocked: session only */ }
  emit();
}

export function subscribe(listener: () => void): () => void {
  if (listeners.size === 0) window.addEventListener("storage", onStorage);
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
    if (listeners.size === 0) window.removeEventListener("storage", onStorage);
  };
}

/** Another tab changed the preferences. */
function onStorage(e: StorageEvent): void {
  if (e.key !== STORAGE_KEY && e.key !== null) return;
  current = readStored();
  emit();
}

export function usePreferences(): Preferences {
  return useSyncExternalStore(subscribe, getPreferences, getPreferences);
}

// --- Motion ---

function systemReducedQuery(): MediaQueryList | null {
  return typeof window !== "undefined" && typeof window.matchMedia === "function" ? window.matchMedia(REDUCED_QUERY) : null;
}

function currentMotion(): ResolvedMotion {
  return resolveMotion(getPreferences().motion, systemReducedQuery()?.matches ?? false);
}

function subscribeMotion(listener: () => void): () => void {
  const query = systemReducedQuery();
  query?.addEventListener("change", listener);
  const unsubscribe = subscribe(listener);
  return () => {
    query?.removeEventListener("change", listener);
    unsubscribe();
  };
}

/** Whether the OS asks for reduced motion, live. */
export function useSystemReducedMotion(): boolean {
  return useSyncExternalStore(
    (listener) => {
      const query = systemReducedQuery();
      query?.addEventListener("change", listener);
      return () => query?.removeEventListener("change", listener);
    },
    () => systemReducedQuery()?.matches ?? false,
    () => false,
  );
}

/** The resolved motion tier, live. */
export function useMotion(): ResolvedMotion {
  return useSyncExternalStore(subscribeMotion, currentMotion, currentMotion);
}

/** Whether movement (slides, scales, smooth scroll) may play right now. For non-React code. */
export function motionAllowed(): boolean {
  return currentMotion() === "full";
}

/** Keeps `<html data-motion>` in step with the preference and the OS. Call once, before the first render. */
export function startMotionSync(): () => void {
  const apply = () => { document.documentElement.dataset.motion = currentMotion(); };
  apply();
  return subscribeMotion(apply);
}

/** `smooth` when movement may play, else `auto` (an instant jump). */
export function scrollBehavior(): ScrollBehavior {
  return motionAllowed() ? "smooth" : "auto";
}
