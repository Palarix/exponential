import { useEffect, useCallback, type ReactNode } from 'react';
import { createPortal } from 'react-dom';

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  children: ReactNode;
  title?: string;
  size?: 'sm' | 'md' | 'lg' | 'xl' | '2xl';
  showCloseButton?: boolean;
}

const sizeStyles: Record<string, { minWidth: string; maxWidth: string }> = {
  sm: { minWidth: '320px', maxWidth: '384px' },
  md: { minWidth: '400px', maxWidth: '448px' },
  lg: { minWidth: '480px', maxWidth: '512px' },
  xl: { minWidth: '600px', maxWidth: '672px' },
  '2xl': { minWidth: '760px', maxWidth: '860px' },
};

export default function Modal({
  isOpen,
  onClose,
  children,
  title,
  size = 'md',
  showCloseButton = true,
}: ModalProps) {
  const handleEscape = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    },
    [onClose]
  );

  useEffect(() => {
    if (isOpen) {
      document.addEventListener('keydown', handleEscape);
      document.body.style.overflow = 'hidden';
    }
    return () => {
      document.removeEventListener('keydown', handleEscape);
      document.body.style.overflow = '';
    };
  }, [isOpen, handleEscape]);

  if (!isOpen) return null;

  return createPortal(
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50 animate-fade-in"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Modal Content */}
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={title ? 'modal-title' : undefined}
        className="relative bg-[var(--color-bg-elevated)] border border-[var(--color-border-default)] rounded-[var(--radius-lg)] shadow-[var(--shadow-lg)] animate-fade-in flex flex-col"
        style={{
          minWidth: sizeStyles[size].minWidth,
          maxWidth: sizeStyles[size].maxWidth,
          maxHeight: '85vh',
          width: '100%',
        }}
      >
        {/* Header */}
        {(title || showCloseButton) && (
          <div className="flex items-center justify-between px-5 py-3 border-b border-[var(--color-border-subtle)]">
            {title && (
              <h2
                id="modal-title"
                className="text-sm font-semibold text-[var(--color-text-primary)]"
              >
                {title}
              </h2>
            )}
            {showCloseButton && (
              <button
                onClick={onClose}
                className="p-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-hover)] transition-colors duration-[var(--duration-fast)] ml-auto"
                aria-label="Close modal"
              >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M18 6 6 18" />
                  <path d="m6 6 12 12" />
                </svg>
              </button>
            )}
          </div>
        )}

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-5 py-4">
          {children}
        </div>
      </div>
    </div>,
    document.body
  );
}

export function ModalFooter({ children }: { children: ReactNode }) {
  return (
    <div className="flex items-center justify-end gap-2 px-5 py-3 border-t border-[var(--color-border-subtle)]">
      {children}
    </div>
  );
}
