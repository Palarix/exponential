import type { Cycle, CyclesResponse, Issue } from "./types";
import { ApiError, type fetchConfig } from "./client";

export type RawConfig = Awaited<ReturnType<typeof fetchConfig>>;

export interface AppConfig {
  prefix: string;
  version: string;
  projectName: string;
  configLabels: Record<string, string>;
  contributors: string[];
  hideDefaultLabels: boolean;
  cyclesEnabled: boolean;
  defaultLabels: { name: string; color: string }[];
}

export function deriveAppConfig(c: RawConfig | undefined): AppConfig {
  const labels = c?.labels || {};
  return {
    prefix: c?.prefix ?? "issue-",
    version: c?.version || "",
    projectName: c?.name || "",
    configLabels: labels,
    contributors: c?.contributors || [],
    hideDefaultLabels: !!c?.hide_default_labels,
    cyclesEnabled: !!c?.cycles?.enabled,
    defaultLabels: (c?.default_labels || []).map((name) => ({
      name,
      color: labels[name] || labels[name.toLowerCase()] || "",
    })),
  };
}

export function effectiveLabelColors(c: AppConfig): Record<string, string> {
  if (c.hideDefaultLabels) return c.configLabels;
  const defaults = Object.fromEntries(c.defaultLabels.map((d) => [d.name, d.color]));
  return { ...defaults, ...c.configLabels };
}

export function withConfigLabels(c: RawConfig | undefined, labels: Record<string, string>): RawConfig | undefined {
  return c && { ...c, labels };
}

/** The cycles to show, or a stable `[]` when cycles are disabled or not loaded. */
export function cyclesFromResponse(r: CyclesResponse | undefined): Cycle[] {
  return r?.enabled && r.cycles ? r.cycles : NO_CYCLES;
}
const NO_CYCLES: Cycle[] = [];

export function renameLabel(labels: Record<string, string>, oldName: string, newName: string, color: string): Record<string, string> {
  return { ...removeLabel(labels, oldName), [newName]: color };
}

export function removeLabel(labels: Record<string, string>, name: string): Record<string, string> {
  return Object.fromEntries(Object.entries(labels).filter(([n]) => n !== name));
}

export interface IssuePatch {
  issueId: string;
  patch: Partial<Issue>;
}

/**
 * Merges `patches` into the issue list, in order. `undo` restores the
 * previous value of only the patched fields, so rolling back one mutation
 * keeps a concurrent edit to other fields.
 */
export function applyIssuePatches(
  issues: Issue[] | undefined,
  patches: IssuePatch[],
): { next: Issue[] | undefined; undo: IssuePatch[] } {
  if (!issues) return { next: undefined, undo: [] };
  const byId = new Map(issues.map((i) => [i.id, i]));
  const undo: IssuePatch[] = [];
  for (const { issueId, patch } of patches) {
    const current = byId.get(issueId);
    if (!current) continue;
    const previous = Object.fromEntries(Object.keys(patch).map((k) => [k, current[k as keyof Issue]]));
    undo.unshift({ issueId, patch: previous });
    byId.set(issueId, { ...current, ...patch });
  }
  return { next: issues.map((i) => byId.get(i.id) ?? i), undo };
}

/**
 * The message of the first failed query among `results` (query results or
 * refetch results; `false` entries are skipped), or null if none failed.
 */
export function firstQueryError(results: ReadonlyArray<{ status: string; error: unknown } | false>): string | null {
  for (const r of results) {
    if (r && r.status === "error") return r.error instanceof ApiError ? r.error.message : "Failed to load";
  }
  return null;
}
