import { fuzzyMatch } from "../../utils/format";
import { VIEWS, type ViewId } from "../../views";
import { VIEW_ICONS } from "../Layout/view-icons";
import type { CommandItem } from "./types";

/**
 * The palette's Navigation rows: the sidebar's views, in sidebar order, with
 * the sidebar's names and icons. Cycles shows only when cycles are enabled.
 * Agent and Drafts stay out (the palette lists chats itself), and so do views
 * the sidebar does not list.
 */
export function navCommandItems(query: string, cyclesEnabled: boolean, onSelect: (view: ViewId) => void): CommandItem[] {
  return VIEWS
    .filter((v) => v.section !== "none" && (!v.requires || (v.requires === "cycles" && cyclesEnabled)))
    .filter((v) => !query || fuzzyMatch(v.navLabel, query))
    .map((v) => ({
      id: `nav:${v.id}`,
      group: "Navigation",
      icon: VIEW_ICONS[v.id],
      label: v.navLabel,
      action: () => onSelect(v.id),
    }));
}
