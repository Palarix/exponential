import { useState, useCallback } from 'react';

export default function CopyableId({ id, className = '' }: { id: string; className?: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback((e: React.MouseEvent) => {
    e.stopPropagation();
    navigator.clipboard.writeText(id);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }, [id]);

  return (
    <button
      onClick={handleCopy}
      title="Copy issue ID"
      className={`font-mono text-[var(--color-text-muted)] hover:text-[var(--color-accent-primary)] transition-colors cursor-pointer ${className}`}
    >
      {copied ? (
        <span className="text-[var(--color-success)]">Copied!</span>
      ) : (
        id
      )}
    </button>
  );
}

