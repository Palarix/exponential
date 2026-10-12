/** ARIA attributes for a button that opens one of the sidebar's menu popovers. */
export function popoverTriggerProps(key: string, openPopover: string | null) {
  return {
    "aria-haspopup": "menu",
    "aria-expanded": openPopover === key,
  } as const;
}
