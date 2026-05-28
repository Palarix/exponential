export function PriorityIcon({
  priority,
  size = 14,
}: {
  priority: number;
  size?: number;
}) {
  const colors: Record<number, string> = {
    0: "var(--color-text-muted)",
    1: "var(--color-error)",
    2: "var(--color-warning)",
    3: "var(--color-text-secondary)",
    4: "var(--color-text-muted)",
  };
  const color = colors[priority] || colors[0];

  if (priority === 0) {
    return (
      <svg
        width={size}
        height={size}
        viewBox="0 0 16 16"
        fill="none"
        style={{ color }}
      >
        <rect x="1" y="7" width="3" height="2" rx="0.5" fill="currentColor" opacity="0.4" />
        <rect x="5" y="7" width="3" height="2" rx="0.5" fill="currentColor" opacity="0.4" />
        <rect x="9" y="7" width="3" height="2" rx="0.5" fill="currentColor" opacity="0.4" />
        <rect x="13" y="7" width="2" height="2" rx="0.5" fill="currentColor" opacity="0.4" />
      </svg>
    );
  }
  if (priority === 1) {
    return (
      <svg
        width={size}
        height={size}
        viewBox="0 0 16 16"
        fill="none"
        style={{ color }}
      >
        <path
          d="M3 2.5L8 1l5 1.5v6c0 3-2.5 5-5 6.5-2.5-1.5-5-3.5-5-6.5v-6z"
          stroke="currentColor"
          strokeWidth="1.5"
          fill="currentColor"
          fillOpacity="0.15"
        />
        <path
          d="M7.5 4.5v4M7.5 10.5v0"
          stroke="currentColor"
          strokeWidth="1.5"
          strokeLinecap="round"
        />
      </svg>
    );
  }
  const filled = priority === 2 ? 3 : priority === 3 ? 2 : 1;
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 16 16"
      fill="none"
      style={{ color }}
    >
      {[0, 1, 2].map((i) => (
        <rect
          key={i}
          x={1 + i * 5}
          y={11 - (i + 1) * 3}
          width="4"
          height={(i + 1) * 3}
          rx="1"
          fill="currentColor"
          opacity={i < filled ? 1 : 0.2}
        />
      ))}
    </svg>
  );
}

export function EstimateIcon() {
  return (
    <svg
      className="w-3.5 h-3.5 text-[var(--color-text-muted)]"
      viewBox="0 0 16 16"
      fill="none"
    >
      <path
        d="M8 2L14 14H2L8 2Z"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinejoin="round"
      />
    </svg>
  );
}
