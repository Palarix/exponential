interface ToggleProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label?: string;
  size?: 'sm' | 'md';
}

const sizes = {
  sm: { track: 'w-7 h-4', thumb: 'w-3 h-3', translate: 'translate-x-3' },
  md: { track: 'w-9 h-5', thumb: 'w-3.5 h-3.5', translate: 'translate-x-4' },
};

export default function Toggle({ checked, onChange, label, size = 'sm' }: ToggleProps) {
  const s = sizes[size];
  return (
    <label className="inline-flex items-center gap-2 cursor-pointer select-none">
      <button
        role="switch"
        aria-checked={checked}
        onClick={() => onChange(!checked)}
        className={`
          relative inline-flex items-center shrink-0 rounded-full
          transition-colors duration-[var(--duration-fast)]
          focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--color-border-focus)]
          ${s.track}
          ${checked ? 'bg-[var(--color-accent-primary)]' : 'bg-[var(--color-border-default)]'}
        `.trim().replace(/\s+/g, ' ')}
      >
        <span
          className={`
            inline-block rounded-full bg-white shadow-sm
            transition-transform duration-[var(--duration-fast)]
            ${s.thumb}
            ${checked ? s.translate : 'translate-x-0.5'}
          `.trim().replace(/\s+/g, ' ')}
        />
      </button>
      {label && (
        <span className="text-xs text-[var(--color-text-muted)]">{label}</span>
      )}
    </label>
  );
}
