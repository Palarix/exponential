import { pointerWithin, closestCenter, type CollisionDetection } from "@dnd-kit/core";

export const backlogCollision: CollisionDetection = (args) => {
  const pointer = pointerWithin(args);
  const directGroup = pointer.find((c) => String(c.id).startsWith("group-"));
  if (directGroup) return [directGroup];

  const all = closestCenter(args);
  const rows = all.filter((c) => !String(c.id).startsWith("group-"));
  return rows.length > 0 ? rows : all;
};
