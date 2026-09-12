import type { ReactNode } from 'react';
import { textClass, type TextColor, type TextSize, type TextWeight } from './text-utils';

interface TextProps {
  children: ReactNode;
  as?: 'span' | 'p' | 'label';
  color?: TextColor;
  size?: TextSize;
  weight?: TextWeight;
  mono?: boolean;
  tabular?: boolean;
  truncate?: boolean;
}

export default function Text({
  children,
  as: Tag = 'span',
  color,
  size,
  weight,
  mono,
  tabular,
  truncate,
}: TextProps) {
  return (
    <Tag className={textClass({ color, size, weight, mono, tabular, truncate })}>
      {children}
    </Tag>
  );
}
