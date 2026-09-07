import { useMemo, useContext } from "react";
import { LabelColorsContext } from "../components/ui/BadgeContexts";
import { mergeAndSort } from "../utils/labels";
import type { Issue } from "../api/client";

export function useAllLabels(issues: Issue[]): string[] {
  const configLabels = useContext(LabelColorsContext);

  return useMemo(
    () => mergeAndSort(configLabels, issues.flatMap((i) => i.labels || [])),
    [issues, configLabels],
  );
}
