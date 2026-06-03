interface StatusIconProps {
  status: string;
  size?: number;
  className?: string;
  isInferred?: boolean;
}

/**
 * SVG circle-based status icons, standardized across all views.
 *
 * - BACKLOG:  dashed circle (muted)
 * - PLANNED:  solid circle outline (light)
 * - DOING:    half-filled circle (blue/amber)
 * - DONE:     filled circle with checkmark (green)
 */
export default function StatusIcon({ status, size = 16, className = '', isInferred }: StatusIconProps) {
  const s = status.toUpperCase();

  switch (s) {
    case 'BACKLOG':
      return (
        <svg width={size} height={size} viewBox="0 0 16 16" fill="none" className={className}>
          <circle
            cx="8" cy="8" r="6.5"
            stroke="var(--color-status-backlog)"
            strokeWidth="1.5"
            strokeDasharray="3 2.5"
            fill="none"
          />
        </svg>
      );

    case 'PLANNED':
      return (
        <svg width={size} height={size} viewBox="0 0 16 16" fill="none" className={className}>
          <circle
            cx="8" cy="8" r="6.5"
            stroke="var(--color-status-planned)"
            strokeWidth="1.5"
            fill="none"
          />
        </svg>
      );

    case 'DOING':
      if (isInferred) {
        return (
          <svg width={size} height={size} viewBox="0 0 16 16" fill="none" className={className}>
            <circle
              cx="8" cy="8" r="6.5"
              stroke="var(--color-status-doing)"
              strokeWidth="1.5"
              strokeDasharray="3 2.5"
              fill="none"
            />
            <path
              d="M8 1.5 A6.5 6.5 0 0 1 8 14.5"
              fill="var(--color-status-doing)"
              opacity="0.5"
            />
          </svg>
        );
      }
      return (
        <svg width={size} height={size} viewBox="0 0 16 16" fill="none" className={className}>
          <circle
            cx="8" cy="8" r="6.5"
            stroke="var(--color-status-doing)"
            strokeWidth="1.5"
            fill="none"
          />
          <path
            d="M8 1.5 A6.5 6.5 0 0 1 8 14.5"
            fill="var(--color-status-doing)"
          />
        </svg>
      );

    case 'BLOCKED':
      return (
        <svg width={size} height={size} viewBox="0 0 16 16" fill="none" className={className}>
          <circle
            cx="8" cy="8" r="7"
            fill="var(--color-status-blocked)"
          />
          <path
            d="M5.5 5.5 L10.5 10.5 M10.5 5.5 L5.5 10.5"
            stroke="white"
            strokeWidth="1.8"
            strokeLinecap="round"
          />
        </svg>
      );

    case 'DONE':
      return (
        <svg width={size} height={size} viewBox="0 0 16 16" fill="none" className={className}>
          <circle
            cx="8" cy="8" r="7"
            fill="var(--color-status-done)"
          />
          <path
            d="M5 8.2 L7.2 10.4 L11 5.6"
            stroke="white"
            strokeWidth="1.8"
            strokeLinecap="round"
            strokeLinejoin="round"
            fill="none"
          />
        </svg>
      );

    default:
      return (
        <svg width={size} height={size} viewBox="0 0 16 16" fill="none" className={className}>
          <circle
            cx="8" cy="8" r="6.5"
            stroke="var(--color-text-muted)"
            strokeWidth="1.5"
            strokeDasharray="3 2.5"
            fill="none"
          />
        </svg>
      );
  }
}
