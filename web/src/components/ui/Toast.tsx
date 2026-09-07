import { useState, useCallback, useRef, useEffect } from "react";
import { ToastContext } from "./ToastContext";

type ToastVariant = "success" | "error" | "info";

interface ToastOptions {
  variant?: ToastVariant;
  duration?: number;
  persistent?: boolean;
}

interface ToastState {
  message: string;
  variant: ToastVariant;
  persistent: boolean;
}

const VARIANT_STYLES: Record<ToastVariant, string> = {
  success: "bg-[var(--color-surface-3)] border-[var(--color-border-default)] text-[var(--color-text-primary)]",
  info: "bg-[var(--color-surface-3)] border-[var(--color-border-default)] text-[var(--color-text-primary)]",
  error: "bg-[var(--color-error)]/10 border-[var(--color-error)]/40 text-[var(--color-error)]",
};

const VARIANT_ICONS: Record<ToastVariant, React.ReactNode> = {
  success: (
    <svg className="w-4 h-4 text-[var(--color-success)] shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
  ),
  info: null,
  error: (
    <svg className="w-4 h-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
    </svg>
  ),
};

export default function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toast, setToast] = useState<ToastState | null>(null);
  const [visible, setVisible] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const dismiss = useCallback(() => {
    if (timerRef.current) clearTimeout(timerRef.current);
    setVisible(false);
    setTimeout(() => setToast(null), 200);
  }, []);

  const showToast = useCallback((message: string, opts?: ToastOptions) => {
    if (timerRef.current) clearTimeout(timerRef.current);
    const variant = opts?.variant ?? "success";
    const persistent = opts?.persistent ?? (variant === "error");
    setToast({ message, variant, persistent });
    setVisible(true);
    if (!persistent) {
      timerRef.current = setTimeout(() => {
        setVisible(false);
        setTimeout(() => setToast(null), 200);
      }, opts?.duration ?? 2500);
    }
  }, []);

  useEffect(() => () => { if (timerRef.current) clearTimeout(timerRef.current); }, []);

  return (
    <ToastContext.Provider value={{ showToast }}>
      {children}
      {toast && (
        <div
          onClick={toast.persistent ? dismiss : undefined}
          className={`fixed bottom-6 left-1/2 -translate-x-1/2 z-[100] flex items-center gap-2 px-3 py-2 rounded-[var(--radius-md)] border shadow-[var(--shadow-md)] text-sm transition-opacity duration-200 ${toast.persistent ? "cursor-pointer" : ""} ${VARIANT_STYLES[toast.variant]} ${visible ? "opacity-100" : "opacity-0"}`}
        >
          {VARIANT_ICONS[toast.variant]}
          {toast.message}
          {toast.persistent && (
            <svg className="w-3.5 h-3.5 ml-1 shrink-0 opacity-60" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          )}
        </div>
      )}
    </ToastContext.Provider>
  );
}
