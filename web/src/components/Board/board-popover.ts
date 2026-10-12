export type BoardPopoverType = "status" | "labels" | "estimate" | "assignee";

/**
 * Where a card's quick-edit popover anchors: the card itself, or for the
 * assignee picker the card's avatar trigger.
 */
export function cardPopoverAnchorSelector(issueId: string, type: BoardPopoverType): string {
  const card = `[data-board-card="${issueId}"]`;
  return type === "assignee" ? `${card} [data-assignee-trigger]` : card;
}
