import { useEffect, useState, type ReactNode } from "react";
import { KeyboardNavContext } from "./context";
import { KeyboardRegistry } from "./registry";

export function KeyboardNavProvider({ children }: { children: ReactNode }) {
  const [registry] = useState(() => new KeyboardRegistry());

  useEffect(() => {
    document.addEventListener("keydown", registry.dispatch);
    return () => document.removeEventListener("keydown", registry.dispatch);
  }, [registry]);

  return (
    <KeyboardNavContext.Provider value={registry}>
      {children}
    </KeyboardNavContext.Provider>
  );
}
