import type { Issue } from "./types";
import type { fetchConfig } from "./client";

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

export function patchIssueList(issues: Issue[] | undefined, issueId: string, patch: Partial<Issue>): Issue[] | undefined {
  return issues?.map((i) => (i.id === issueId ? { ...i, ...patch } : i));
}
