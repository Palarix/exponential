/**
 * Stacking layers of the overlays that portal to the body. A popover opens
 * above a dialog, so dropdowns inside dialogs show; toasts and context menus
 * (z-[100]) and the command palette (z-[200]) stay above both.
 */
export const DIALOG_LAYER = "z-[60]";
export const POPOVER_LAYER = "z-[70]";
