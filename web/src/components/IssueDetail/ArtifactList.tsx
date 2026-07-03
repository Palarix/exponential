import type { ArtifactSummary } from "../../api/types";
import { FileText, BookOpen, Paperclip } from "lucide-react";
import { formatRelativeTime, shortName } from "../../utils/format";

const TYPE_ICON: Record<string, typeof FileText> = {
  spec: FileText,
  walkthrough: BookOpen,
};

const TYPE_LABEL: Record<string, string> = {
  spec: "Spec",
  walkthrough: "Walkthrough",
};

export default function ArtifactList({
  artifacts,
}: {
  artifacts: ArtifactSummary[];
}) {
  if (artifacts.length === 0) return null;

  return (
    <div className="mt-6 pt-5 border-t border-[var(--color-border-subtle)]">
      <h3 className="text-sm font-semibold text-[var(--color-text-primary)] mb-3">
        Artifacts
      </h3>
      <div className="space-y-1">
        {artifacts.map((a) => {
          const Icon = TYPE_ICON[a.artifact_type] ?? Paperclip;
          const label = TYPE_LABEL[a.artifact_type] ?? a.filename;
          return (
            <div
              key={a.filename}
              className="flex items-center gap-3 px-3 py-2 rounded-[var(--radius-sm)] hover:bg-[var(--color-hover-surface)] transition-colors"
            >
              <Icon className="w-4 h-4 text-[var(--color-text-muted)] shrink-0" />
              <span className="text-sm font-medium text-[var(--color-text-primary)] min-w-0 truncate">
                {label}
              </span>
              {a.artifact_type !== "generic" && (
                <span className="text-xs font-mono text-[var(--color-text-muted)]">
                  {a.filename}
                </span>
              )}
              <div className="flex-1" />
              <span className="text-xs text-[var(--color-text-muted)] shrink-0">
                {shortName(a.updated_by)}
              </span>
              <span className="text-xs text-[var(--color-text-muted)] shrink-0">
                {formatRelativeTime(a.updated_at)}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}
