/** Which rail segments a timeline row draws, based on its display position. */
export function timelineConnectors(
  index: number,
  count: number,
): { above: boolean; below: boolean } {
  return { above: index > 0, below: index < count - 1 };
}
