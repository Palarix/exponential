import { Component, type ErrorInfo, type ReactNode } from "react";

interface Props {
  children: ReactNode;
  onReset?: () => void;
}

interface State {
  error: Error | null;
}

export default class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("ErrorBoundary caught:", error, info.componentStack);
  }

  handleReset = () => {
    this.setState({ error: null });
    this.props.onReset?.();
  };

  render() {
    if (!this.state.error) return this.props.children;

    return (
      <div className="flex flex-col items-center justify-center h-full gap-6 px-8">
        <svg width="160" height="120" viewBox="0 0 160 120" fill="none" className="opacity-30">
          <rect x="30" y="20" width="100" height="70" rx="8" stroke="var(--color-text-muted)" strokeWidth="1.5" strokeDasharray="4 3" />
          <circle cx="80" cy="48" r="12" stroke="var(--color-text-muted)" strokeWidth="1.5" />
          <path d="M75 43l10 10M85 43l-10 10" stroke="var(--color-text-muted)" strokeWidth="1.5" strokeLinecap="round" />
          <circle cx="80" cy="72" r="2" fill="var(--color-text-muted)" />
          <path d="M50 100h60" stroke="var(--color-text-muted)" strokeWidth="1.5" strokeLinecap="round" strokeDasharray="3 4" />
          <path d="M55 106h50" stroke="var(--color-text-muted)" strokeWidth="1.5" strokeLinecap="round" strokeDasharray="3 4" />
        </svg>

        <div className="flex flex-col items-center gap-2 max-w-100 text-center">
          <h2 className="text-lg font-semibold text-[var(--color-text-primary)]">Something went wrong</h2>
          <p className="text-sm text-[var(--color-text-secondary)] leading-relaxed">
            An unexpected error occurred while rendering this view.
          </p>
          <pre className="mt-2 max-w-full overflow-x-auto text-xs font-mono text-[var(--color-text-muted)] bg-[var(--color-surface-1)] rounded-[var(--radius-md)] px-3 py-2 text-left">
            {this.state.error.message}
          </pre>
        </div>

        <button
          onClick={this.handleReset}
          className="px-4 py-2 text-sm bg-[var(--color-surface-1)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)] transition-colors"
        >
          Try Again
        </button>
      </div>
    );
  }
}
