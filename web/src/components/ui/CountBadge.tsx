import { formatCount, countBadgeClass } from './count-badge-utils';

interface CountBadgeProps {
  count: number;
  label?: string;
  short?: boolean;
  rounded?: boolean;
}

export default function CountBadge({
  count,
  label = 'issue',
  short = false,
  rounded = false,
}: CountBadgeProps) {
  return (
    <span className={countBadgeClass(rounded)}>
      {formatCount(count, label, short)}
    </span>
  );
}
