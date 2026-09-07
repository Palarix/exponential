import { createContext, useContext } from "react";

interface ToastOptions {
  variant?: "success" | "error" | "info";
  duration?: number;
  persistent?: boolean;
}

interface ToastContextValue {
  showToast: (message: string, opts?: ToastOptions) => void;
}

export const ToastContext = createContext<ToastContextValue | null>(null);

export function useToast() {
  const ctx = useContext(ToastContext);
  if (!ctx) throw new Error("useToast must be used within ToastProvider");
  return ctx.showToast;
}
