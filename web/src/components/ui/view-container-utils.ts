export type ViewMaxWidth = "7xl" | "detail";

const MAX_WIDTH_CLASS: Record<ViewMaxWidth, string> = {
  "7xl": "max-w-7xl",
  detail: "max-w-[76rem]",
};

export function viewContentClass(scroll: boolean): string {
  return `flex-1 min-h-0 ${scroll ? "scroll-stable" : "overflow-hidden"}`;
}

export function viewInnerClass(maxWidth: ViewMaxWidth | undefined): string | null {
  return maxWidth ? `mx-auto w-full ${MAX_WIDTH_CLASS[maxWidth]}` : null;
}
