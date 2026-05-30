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
  const barCount = priority === 0 ? 0 : 5 - priority;

  if (barCount === 0) {
    return (
      <svg width={size} height={size} viewBox="0 0 14 14" fill="none" className={`shrink-0 ${className}`}>
        <line x1="3" y1="7" x2="5.5" y2="7" stroke={color} strokeWidth="1.5" strokeLinecap="round" />
        <line x1="8.5" y1="7" x2="11" y2="7" stroke={color} strokeWidth="1.5" strokeLinecap="round" />
      </svg>
    );
  }

  const barWidth = 2;
  const barGap = 1;
  const totalWidth = barCount * barWidth + (barCount - 1) * barGap;
  const startX = (14 - totalWidth) / 2;

  return (
    <svg width={size} height={size} viewBox="0 0 14 14" fill="none" className={`shrink-0 ${className}`}>
      {Array.from({ length: barCount }, (_, i) => {
        const x = startX + i * (barWidth + barGap);
        const height = 4 + i * 2;
        const y = 12 - height;
        return (
          <rect key={i} x={x} y={y} width={barWidth} height={height} rx="0.5" fill={color} />
        );
      })}
    </svg>
  );
}
