export type BoardPopoverType = "status" | "labels" | "estimate" | "assignee" | "milestone";

/**
 * Where a card's quick-edit popover anchors: the card itself, or for the
 * assignee and milestone pickers the card's avatar and milestone chip. A card
 * without a milestone has no chip, so the caller falls back to the card.
 */
export function cardPopoverAnchorSelector(issueId: string, type: BoardPopoverType): string {
  const card = `[data-board-card="${issueId}"]`;
  if (type === "assignee") return `${card} [data-assignee-trigger]`;
  if (type === "milestone") return `${card} [data-milestone-trigger]`;
  return card;
}
