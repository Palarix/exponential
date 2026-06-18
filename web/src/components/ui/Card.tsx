import type { ReactNode, HTMLAttributes } from 'react';

interface CardProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode;
  variant?: 'default' | 'elevated';
  interactive?: boolean;
  padding?: 'none' | 'sm' | 'md' | 'lg';
}

const variantStyles: Record<string, string> = {
  default: 'bg-[var(--color-surface-1)] border border-[var(--color-border-subtle)]',
  elevated: 'bg-[var(--color-surface-2)] border border-[var(--color-border-default)]',
};

const paddingStyles: Record<string, string> = {
  none: '',
  sm: 'p-3',
  md: 'p-4',
  lg: 'p-6',
};

export default function Card({
  children,
  variant = 'default',
  interactive = false,
  padding = 'md',
  className = '',
  ...props
}: CardProps) {
  return (
    <div
      className={`
        rounded-[var(--radius-md)]
        transition-colors duration-[var(--duration-fast)]
        ${variantStyles[variant]}
        ${paddingStyles[padding]}
        ${interactive ? 'cursor-pointer hover:bg-[var(--color-hover-surface-2)]' : ''}
        ${className}
      `.trim().replace(/\s+/g, ' ')}
      {...props}
    >
      {children}
    </div>
  );
}
