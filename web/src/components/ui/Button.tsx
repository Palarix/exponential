import { forwardRef, type ReactNode, type ButtonHTMLAttributes } from 'react';

type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger' | 'warning';
type ButtonTone = 'neutral' | 'danger' | 'warning';
type ButtonSize = 'sm' | 'md' | 'lg';

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  /** Colors the text of a secondary or ghost button; its tint follows the text. */
  tone?: ButtonTone;
  /** Adds a border in the text color's tint, so it reads on any surface. */
  bordered?: boolean;
  size?: ButtonSize;
  children: ReactNode;
  loading?: boolean;
  icon?: ReactNode;
}

const variantStyles: Record<ButtonVariant, string> = {
  primary: 'bg-[var(--color-accent-primary)] hover:bg-[var(--color-accent-primary-hover)] active:bg-[var(--color-accent-primary-active)] text-white',
  // A wash of the text color: it lifts off whatever surface the button sits on.
  secondary: 'bg-[var(--color-tint-rest)] hover:bg-[var(--color-tint-hover)] active:bg-[var(--color-tint-active)]',
  // No fill at rest; on hover it takes secondary's resting wash, so it never outweighs secondary.
  ghost: 'bg-transparent hover:bg-[var(--color-tint-rest)] active:bg-[var(--color-tint-hover)]',
  // Solid status buttons rest a little below their base color and hover a little above it.
  danger: 'bg-[var(--color-error-solid)] hover:bg-[var(--color-error-solid-hover)] active:bg-[var(--color-error-solid-active)] text-white',
  warning: 'bg-[var(--color-warning-solid)] hover:bg-[var(--color-warning-solid-hover)] active:bg-[var(--color-warning-solid-active)] text-[var(--color-text-on-warning)]',
};

// The text color of the tinted variants; the tint is a wash of it.
const toneStyles: Record<'secondary' | 'ghost', Record<ButtonTone, string>> = {
  secondary: {
    neutral: 'text-[var(--color-text-primary)]',
    danger: 'text-[var(--color-error)]',
    warning: 'text-[var(--color-warning)]',
  },
  ghost: {
    neutral: 'text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]',
    danger: 'text-[var(--color-error)]',
    warning: 'text-[var(--color-warning)]',
  },
};

const sizeStyles: Record<ButtonSize, string> = {
  sm: 'h-8 px-3 text-sm gap-2',
  md: 'h-9 px-4 text-sm gap-2',
  lg: 'h-10 px-5 text-sm gap-2',
};

const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  function Button({
    variant = 'primary',
    tone = 'neutral',
    bordered = false,
    size = 'md',
    children,
    loading = false,
    icon,
    className = '',
    disabled,
    ...props
  }, ref) {
    return (
      <button
        ref={ref}
        className={`
          inline-flex items-center justify-center font-medium
          rounded-[var(--radius-md)]
          transition-colors duration-[var(--duration-fast)]
          focus-visible:outline-none focus-visible:ring-2
          focus-visible:ring-[var(--color-border-focus)]
          disabled:opacity-40 disabled:cursor-not-allowed disabled:pointer-events-none
          ${variantStyles[variant]}
          ${variant === 'secondary' || variant === 'ghost' ? toneStyles[variant][tone] : ''}
          ${bordered ? 'border border-[var(--color-tint-border)]' : ''}
          ${sizeStyles[size]}
          ${className}
        `.trim().replace(/\s+/g, ' ')}
        disabled={disabled || loading}
        {...props}
      >
        {loading ? (
          <svg className="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
          </svg>
        ) : icon ? (
          <span className="flex-shrink-0">{icon}</span>
        ) : null}
        {children}
      </button>
    );
  },
);

export default Button;
