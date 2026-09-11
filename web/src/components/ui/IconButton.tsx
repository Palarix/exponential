import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from 'react';
import Tooltip from './Tooltip';
import { iconButtonClass } from './icon-button-utils';

interface IconButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  icon: ReactNode;
  active?: boolean;
  tooltip?: string;
}

const IconButton = forwardRef<HTMLButtonElement, IconButtonProps>(
  function IconButton({ icon, active, tooltip, className, ...props }, ref) {
    const btn = (
      <button ref={ref} className={iconButtonClass(active, className)} {...props}>
        {icon}
        {active && (
          <span className="absolute -top-0.5 -right-0.5 w-2 h-2 rounded-full bg-[var(--color-accent-primary)]" />
        )}
      </button>
    );

    return tooltip ? <Tooltip content={tooltip}>{btn}</Tooltip> : btn;
  },
);

export default IconButton;
