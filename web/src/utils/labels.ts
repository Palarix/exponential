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
