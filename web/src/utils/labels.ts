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
