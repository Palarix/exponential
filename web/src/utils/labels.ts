export function canonicalLabel(
  label: string,
  configLabels: Record<string, string>,
): string {
  const lk = label.toLowerCase();
  for (const k of Object.keys(configLabels)) {
    if (k.toLowerCase() === lk) return k;
  }
  return label;
}

export function labelColor(
  label: string,
  configLabels: Record<string, string>,
): string {
  const lk = label.toLowerCase();
  for (const [k, v] of Object.entries(configLabels)) {
    if (k.toLowerCase() === lk) return v;
  }
  return "var(--color-text-muted)";
}

export function toggleLabel(
  current: string[],
  label: string,
): string[] {
  const lk = label.toLowerCase();
  return current.some((l) => l.toLowerCase() === lk)
    ? current.filter((l) => l.toLowerCase() !== lk)
    : [...current, label];
}

export function deduplicateLabels(
  labels: string[],
  configLabels: Record<string, string>,
): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const l of labels) {
    const key = l.toLowerCase();
    if (seen.has(key)) continue;
    seen.add(key);
    out.push(canonicalLabel(l, configLabels));
  }
  return out;
}

export function mergeAndSort(
  configLabels: Record<string, string>,
  issueLabels: string[],
): string[] {
  return deduplicateLabels(issueLabels, configLabels).sort((a, b) =>
    a.localeCompare(b, undefined, { sensitivity: "base" }),
  );
}

export function splitLabels(
  issueLabels: string[],
  defaultLabels: { name: string; color: string }[],
): { primary: string[]; metadata: string[] } {
  const defaultSet = new Set(defaultLabels.map((d) => d.name.toLowerCase()));
  const defaultOrder = new Map(defaultLabels.map((d, i) => [d.name.toLowerCase(), i]));
  const primary: string[] = [];
  const metadata: string[] = [];
  for (const l of issueLabels) {
    if (defaultSet.has(l.toLowerCase())) {
      primary.push(l);
    } else {
      metadata.push(l);
    }
  }
  primary.sort((a, b) => (defaultOrder.get(a.toLowerCase()) ?? 0) - (defaultOrder.get(b.toLowerCase()) ?? 0));
  metadata.sort((a, b) => a.localeCompare(b, undefined, { sensitivity: "base" }));
  return { primary, metadata };
}

export function contrastTextColor(hex: string): string {
  const raw = hex.replace("#", "");
  const r = parseInt(raw.substring(0, 2), 16) / 255;
  const g = parseInt(raw.substring(2, 4), 16) / 255;
  const b = parseInt(raw.substring(4, 6), 16) / 255;
  const luminance =
    0.2126 * (r <= 0.03928 ? r / 12.92 : ((r + 0.055) / 1.055) ** 2.4) +
    0.7152 * (g <= 0.03928 ? g / 12.92 : ((g + 0.055) / 1.055) ** 2.4) +
    0.0722 * (b <= 0.03928 ? b / 12.92 : ((b + 0.055) / 1.055) ** 2.4);
  return luminance > 0.179 ? "#000000" : "#ffffff";
}

export function filterLabelsByName<T extends { name: string }>(
  labels: T[],
  query: string,
): T[] {
  const q = query.trim().toLowerCase();
  if (!q) return labels;
  return labels.filter((l) => l.name.toLowerCase().includes(q));
}

/** Whether `name` is one of the project's built-in (default) labels. */
export function isBuiltinLabel(
  name: string,
  defaultLabels: { name: string }[],
): boolean {
  const lk = name.toLowerCase();
  return defaultLabels.some((d) => d.name.toLowerCase() === lk);
}

/**
 * Splits labels into built-ins, in `defaultLabels` order, and custom labels,
 * in their original order.
 */
export function partitionBuiltinLabels<T extends { name: string }>(
  labels: T[],
  defaultLabels: { name: string }[],
): { builtin: T[]; custom: T[] } {
  const rank = new Map(defaultLabels.map((d, i) => [d.name.toLowerCase(), i]));
  const builtin: T[] = [];
  const custom: T[] = [];
  for (const l of labels) {
    (rank.has(l.name.toLowerCase()) ? builtin : custom).push(l);
  }
  builtin.sort((a, b) => rank.get(a.name.toLowerCase())! - rank.get(b.name.toLowerCase())!);
  return { builtin, custom };
}

/** The `limit` most recently updated issues carrying `label` (case-insensitive). */
export function recentIssuesWithLabel<T extends { labels?: string[]; updated_at: string }>(
  issues: T[],
  label: string,
  limit: number,
): T[] {
  const lk = label.toLowerCase();
  return issues
    .filter((i) => (i.labels || []).some((l) => l.toLowerCase() === lk))
    .sort((a, b) => b.updated_at.localeCompare(a.updated_at))
    .slice(0, limit);
}
