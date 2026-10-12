type Containers = Record<string, string[] | undefined>;

/** Columns keyboard navigation can visit: rendered (not hidden) and expanded. */
export function navigableColumnIds(
  columns: { id: string }[],
  hidden: Set<string>,
  collapsed: Set<string>,
): string[] {
  return columns.map((c) => c.id).filter((id) => !hidden.has(id) && !collapsed.has(id));
}

/** The next column in `dir` that has cards, or `from` if there is none. */
export function stepColumn(colIds: string[], from: number, dir: 1 | -1, containers: Containers): number {
  for (let n = from + dir; n >= 0 && n < colIds.length; n += dir) {
    if ((containers[colIds[n]]?.length ?? 0) > 0) return n;
  }
  return from;
}

/**
 * The column focus should sit in: `focusCol` while it exists and has cards,
 * otherwise the first column with cards, or -1 when the board is empty.
 */
export function resolveFocusColumn(colIds: string[], focusCol: number, containers: Containers): number {
  if ((containers[colIds[focusCol]]?.length ?? 0) > 0) return focusCol;
  return colIds.findIndex((id) => (containers[id]?.length ?? 0) > 0);
}
