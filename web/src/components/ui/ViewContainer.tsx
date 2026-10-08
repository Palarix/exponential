import type { HTMLAttributes, ReactNode, Ref } from "react";
import { cn } from "../../utils/cn";
import { type ViewMaxWidth, viewContentClass, viewInnerClass } from "./view-container-utils";

interface ViewContainerProps {
  topBar: ReactNode;
  /** Rendered between the TopBar and the content (banners, tab bars). */
  header?: ReactNode;
  /** `false` clips the content instead of scrolling it (canvases, empty states). */
  scroll?: boolean;
  /** Centers a constrained column inside the full-width scroll container,
   *  keeping the scrollbar at the right edge of the view. */
  maxWidth?: ViewMaxWidth;
  className?: string;
  contentClassName?: string;
  /** Extra classes for the centered column; only applies with `maxWidth`. */
  innerClassName?: string;
  contentRef?: Ref<HTMLDivElement>;
  contentProps?: Omit<HTMLAttributes<HTMLDivElement>, "className" | "children">;
  ref?: Ref<HTMLDivElement>;
  children: ReactNode;
}

export function ViewContainer({
  topBar,
  header,
  scroll = true,
  maxWidth,
  className,
  contentClassName,
  innerClassName,
  contentRef,
  contentProps,
  ref,
  children,
}: ViewContainerProps) {
  const inner = viewInnerClass(maxWidth);
  return (
    <div ref={ref} className={cn("h-full flex flex-col", className)}>
      {topBar}
      {header}
      <div {...contentProps} ref={contentRef} className={cn(viewContentClass(scroll), contentClassName)}>
        {inner ? <div className={cn(inner, innerClassName)}>{children}</div> : children}
      </div>
    </div>
  );
}
