import { pointerWithin, closestCenter, type CollisionDetection } from "@dnd-kit/core";

export const boardCollision: CollisionDetection = (args) => {
  const pointer = pointerWithin(args);
  const column = pointer.find((c) => String(c.id).startsWith("column-"));
  if (column) {
    const cards = pointer.filter((c) => !String(c.id).startsWith("column-"));
    return cards.length > 0 ? cards : [column];
  }

  const all = closestCenter(args);
  const cards = all.filter((c) => !String(c.id).startsWith("column-"));
  return cards.length > 0 ? cards : all;
};
