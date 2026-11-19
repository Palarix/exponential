import { createContext, useContext } from "react";

/**
 * What a Menu gives its own rows: the handlers that act on the menu. A Menu
 * provides it, and tags each direct child with `_ownerId` when it clones it.
 */
export interface MenuOwner {
  ownerId: string;
  onItemMouseEnter(index: number): void;
  onItemClick(index: number): void;
  closeSubMenu(): void;
  /** The mouse reached a submenu's flyout: drop a pending hover switch. */
  onFlyoutEnter(): void;
}

export const MenuOwnerContext = createContext<MenuOwner | null>(null);

/**
 * The menu that owns a row, or null. Only a direct child of a Menu carries
 * that menu's `ownerId`; a row nested deeper (say, in a submenu's custom
 * panel) isn't one of its rows and runs its own `onClick` instead.
 */
export function useMenuOwner(ownerId: string | undefined): MenuOwner | null {
  const owner = useContext(MenuOwnerContext);
  return owner && ownerId !== undefined && owner.ownerId === ownerId ? owner : null;
}
