import type { ReactNode } from 'react';
import { headingClass, headingWrapperClass } from './heading-utils';

interface HeadingProps {
  title: string;
  as?: 'span' | 'h1' | 'h2' | 'h3' | 'p';
  subtitle?: ReactNode;
  leading?: 'snug';
  truncate?: boolean;
  clamp?: number;
}

export default function Heading({
  title,
  as: Tag = 'span',
  subtitle,
  leading,
  truncate,
  clamp,
}: HeadingProps) {
  const cls = headingClass({ leading, truncate, clamp });
  return subtitle ? (
    <div className={headingWrapperClass()}>
      <Tag className={`${cls} shrink-0`}>{title}</Tag>
      {subtitle}
    </div>
  ) : (
    <Tag className={cls}>{title}</Tag>
  );
}
