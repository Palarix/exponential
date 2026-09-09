export function truncate(s: string, max: number): string {
  return s.length > max ? s.slice(0, max) + "…" : s;
}

const KIND_COLORS: Record<string, string> = {
  blocks: "var(--color-error)",
  blocked_by: "var(--color-error)",
  depends_on: "var(--color-error)",
  dependency_of: "var(--color-error)",
  relates_to: "var(--color-info)",
  related: "var(--color-info)",
  duplicates: "var(--color-text-muted)",
  duplicated_by: "var(--color-text-muted)",
};

export function kindColor(kind: string): string {
  return KIND_COLORS[kind] ?? "var(--color-text-muted)";
}

export function edgePath(points: { x: number; y: number }[]): string {
  if (points.length === 0) return "";
  if (points.length === 1) return `M${points[0].x},${points[0].y}`;
  let d = `M${points[0].x},${points[0].y}`;
  for (let i = 1; i < points.length; i++) {
    d += `L${points[i].x},${points[i].y}`;
  }
  return d;
}
