import type { ReactNode } from "react";

export interface CommandItem {
  id: string;
  group: string;
  icon: ReactNode;
  label: string;
  meta?: ReactNode;
  shortcut?: string;
  action: () => void;
}
