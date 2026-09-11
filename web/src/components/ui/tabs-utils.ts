export function cycleTabs<T extends string>(
  keys: readonly T[],
  activeId: string,
  direction: 1 | -1,
): T | null {
  if (keys.length <= 1) return null;
  const idx = keys.indexOf(activeId as T);
  if (idx < 0) return null;
  return keys[(idx + direction + keys.length) % keys.length];
}
