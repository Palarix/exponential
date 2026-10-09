import { useContext, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { StatusBarSlotContext } from "./contexts";

/** Renders its children into the left side of Layout's status bar. */
export function StatusBarSlot({ children }: { children: ReactNode }) {
  const target = useContext(StatusBarSlotContext);
  return target ? createPortal(children, target) : null;
}
