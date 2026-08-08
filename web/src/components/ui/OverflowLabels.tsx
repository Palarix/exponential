import { useRef, useState, useLayoutEffect, useContext, useMemo } from "react";
import { LabelBadge, DefaultLabelsContext } from "./Badge";
import { splitLabels } from "../../utils/labels";

export default function OverflowLabels({ labels }: { labels: string[] }) {
  const defaultLabels = useContext(DefaultLabelsContext);
  const containerRef = useRef<HTMLDivElement>(null);
  const measureRef = useRef<HTMLDivElement>(null);
  const [maxVisible, setMaxVisible] = useState(labels.length);

  const { primary, metadata } = splitLabels(labels, defaultLabels);
  const all = useMemo(() => [...metadata, ...primary], [labels, defaultLabels]);

  useLayoutEffect(() => {
    const container = containerRef.current;
    const measure = measureRef.current;
    if (!container || !measure || all.length === 0) return;

    const row = container.closest("[data-backlog-row]") as HTMLElement | null;
    if (!row) {
      setMaxVisible(all.length);
      return;
    }

    const compute = () => {
      const budget = Math.max(100, row.clientWidth * 0.35);
      const items = measure.children;
      const gap = 12;
      const overflowWidth = 28;

      let used = 0;
      let count = 0;

      for (let i = 0; i < items.length; i++) {
        const w = (items[i] as HTMLElement).offsetWidth;
        const withGap = count > 0 ? w + gap : w;
        const hasMore = i < items.length - 1;

        if (used + withGap + (hasMore ? gap + overflowWidth : 0) <= budget) {
          used += withGap;
          count++;
        } else {
          break;
        }
      }

      setMaxVisible(Math.max(count, 1));
    };

    compute();
    const observer = new ResizeObserver(compute);
    observer.observe(row);
    return () => observer.disconnect();
  }, [all]);

  if (all.length === 0) return null;

  const visible = Math.min(maxVisible, all.length);
  const extra = all.length - visible;

  return (
    <div ref={containerRef} className="flex items-center gap-3">
      <div
        ref={measureRef}
        className="flex items-center gap-3 pointer-events-none"
        style={{ position: "fixed", top: -9999, left: -9999, visibility: "hidden" }}
        aria-hidden="true"
      >
        {all.map((label) => (
          <LabelBadge key={label} label={label} />
        ))}
      </div>
      {all.slice(0, visible).map((label) => (
        <LabelBadge key={label} label={label} />
      ))}
      {extra > 0 && (
        <span className="text-xs text-[var(--color-text-muted)] shrink-0">
          +{extra}
        </span>
      )}
    </div>
  );
}
