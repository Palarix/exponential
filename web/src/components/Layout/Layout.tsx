import { useState, type ComponentProps, type ReactNode } from "react";
import Sidebar from "./Sidebar";
import { StatusBarSlotContext } from "../../app/contexts";

interface LayoutProps {
  children: ReactNode;
  sidebar: ComponentProps<typeof Sidebar>;
  version: string;
  connected: boolean;
}

export default function Layout({
  children,
  sidebar,
  version,
  connected,
}: LayoutProps) {
  const [statusSlot, setStatusSlot] = useState<HTMLDivElement | null>(null);

  return (
    <div className="flex h-screen overflow-hidden bg-[var(--color-chrome)]">
      <Sidebar {...sidebar} />

      {/* Main */}
      <div className="flex-1 pt-2 pr-2 overflow-hidden flex flex-col">
        <main className="flex-1 overflow-hidden rounded-[var(--radius-lg)] border border-[var(--color-border-subtle)] border-b-0" style={{ background: "linear-gradient(160deg, #1A1B1C 0%, var(--color-surface) 20%)" }}>
          <StatusBarSlotContext.Provider value={statusSlot}>
            {children}
          </StatusBarSlotContext.Provider>
        </main>
        <div className="flex items-center px-4 py-2 shrink-0 gap-3">
          {/* Views fill this through <StatusBarSlot> */}
          <div ref={setStatusSlot} className="flex-1 min-w-0" />
          <div className="flex items-center gap-2 text-xs shrink-0">
            <span
              className={`w-2 h-2 rounded-full ${connected ? "bg-[var(--color-success)]" : "bg-[var(--color-error)]"}`}
            />
            {connected ? (
              <span className="text-[var(--color-text-muted)]">
                Exponential {version && `v${version}`}
              </span>
            ) : (
              <span className="text-[var(--color-error)]">Disconnected</span>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
