interface PriorityIconProps {
  priority: number;
  size?: number;
  className?: string;
}

const colors: Record<number, string> = {
  0: 'var(--color-text-muted)',
  1: 'var(--color-error)',
  2: 'var(--color-warning)',
  3: 'var(--color-text-secondary)',
  4: 'var(--color-text-muted)',
};

export default function PriorityIcon({ priority, size = 14, className = '' }: PriorityIconProps) {
  const color = colors[priority] || colors[0];
  if (priority === 0) {
    return (
      <svg width={size} height={size} viewBox="0 0 16 16" fill="none" className={`shrink-0 ${className}`}>
        <line x1="4" y1="8" x2="6.5" y2="8" stroke={color} strokeWidth="1.5" strokeLinecap="round" />
        <line x1="9.5" y1="8" x2="12" y2="8" stroke={color} strokeWidth="1.5" strokeLinecap="round" />
      </svg>
    );
  }

  if (priority === 1) {
    return (
      <svg width={size} height={size} viewBox="0 0 16 16" fill="none" className={`shrink-0 ${className}`}>
        <rect x="2.5" y="2.5" width="11" height="11" rx="2.5" stroke={color} strokeWidth="1.5" fill={color} fillOpacity="0.15" />
        <line x1="8" y1="5" x2="8" y2="9" stroke={color} strokeWidth="1.5" strokeLinecap="round" />
        <circle cx="8" cy="11" r="0.75" fill={color} />
      </svg>
    );
  }

  const totalBars = 4;
  const filledBars = 5 - priority;
  const barWidth = 2;
  const barGap = 1;
  const totalWidth = totalBars * barWidth + (totalBars - 1) * barGap;
  const startX = (16 - totalWidth) / 2;

  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="none" className={`shrink-0 ${className}`}>
      {Array.from({ length: totalBars }, (_, i) => {
        const x = startX + i * (barWidth + barGap);
        const height = 4 + i * 2;
        const y = 13 - height;
        return (
          <rect key={i} x={x} y={y} width={barWidth} height={height} rx="0.5" fill={color} opacity={i < filledBars ? 1 : 0.2} />
        );
      })}
    </svg>
  );
}
