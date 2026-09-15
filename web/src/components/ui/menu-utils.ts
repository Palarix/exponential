export function nextIndex(
  current: number,
  count: number,
  direction: 1 | -1,
  skip: Set<number>,
): number {
  if (count === 0) return -1;

  if (current === -1) {
    const start = direction === 1 ? 0 : count - 1;
    if (!skip.has(start)) return start;
    current = start;
  }

  for (let i = 1; i < count; i++) {
    const candidate = ((current + direction * i) % count + count) % count;
    if (!skip.has(candidate)) return candidate;
  }
  return -1;
}

export function matchShortcut(
  key: string,
  shortcuts: Map<number, string>,
): number {
  const lower = key.toLowerCase();
  let result = -1;
  shortcuts.forEach((shortcut, index) => {
    if (result === -1 && shortcut.toLowerCase() === lower) result = index;
  });
  return result;
}

/**
 * Check if point (px,py) is inside the triangle formed by three vertices.
 * Uses barycentric coordinate method.
 */
export function pointInTriangle(
  px: number, py: number,
  x1: number, y1: number,
  x2: number, y2: number,
  x3: number, y3: number,
): boolean {
  const d = (x2 - x1) * (y3 - y1) - (x3 - x1) * (y2 - y1);
  if (d === 0) return false;
  const a = ((x2 - px) * (y3 - py) - (x3 - px) * (y2 - py)) / d;
  const b = ((x3 - px) * (y1 - py) - (x1 - px) * (y3 - py)) / d;
  const c = 1 - a - b;
  return a >= 0 && b >= 0 && c >= 0;
}
