import type { ReactNode, HTMLAttributes } from 'react';

interface CardProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode;
  variant?: 'default' | 'elevated' | 'glass';
  interactive?: boolean;
  glow?: boolean;
  padding?: 'none' | 'sm' | 'md' | 'lg';
}

const variantStyles: Record<string, string> = {
  default: `
    bg-[var(--color-surface)] 
    border border-[var(--color-border-subtle)]
  `,
  elevated: `
    bg-[var(--color-surface-elevated)] 
    border border-[var(--color-border-default)]
    shadow-[var(--shadow-md)]
  `,
  glass: `
    glass
  `,
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
  glow = false,
  padding = 'md',
  className = '',
  ...props
}: CardProps) {
  return (
    <div
      className={`
        rounded-[var(--radius-lg)]
        transition-all duration-[var(--duration-normal)]
        ${variantStyles[variant]}
        ${paddingStyles[padding]}
        ${interactive ? 'cursor-pointer hover:bg-[var(--color-bg-hover)] hover:border-[var(--color-border-accent)]' : ''}
        ${glow ? 'hover:shadow-[var(--shadow-glow-sm)]' : ''}
        ${className}
      `.trim().replace(/\s+/g, ' ')}
      {...props}
    >
      {children}
    </div>
  );
}
