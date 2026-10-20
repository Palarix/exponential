export type BoardPopoverType = "status" | "labels" | "estimate" | "assignee" | "release";

/**
 * Where a card's quick-edit popover anchors: the card itself, or for the
 * assignee and release pickers the card's avatar and release chip. A card
 * without a release has no chip, so the caller falls back to the card.
 */
export function cardPopoverAnchorSelector(issueId: string, type: BoardPopoverType): string {
  const card = `[data-board-card="${issueId}"]`;
  if (type === "assignee") return `${card} [data-assignee-trigger]`;
  if (type === "release") return `${card} [data-release-trigger]`;
  return card;
}
