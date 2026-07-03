import type { ArtifactSummary } from "../../api/types";
import type { Issue } from "../../api/client";
import { fetchArtifactContent } from "../../api/client";
import { FileCodeCorner, FileBracesCorner, Paperclip, Download } from "lucide-react";
import { formatRelativeTime, shortName } from "../../utils/format";

const TYPE_ICON: Record<string, typeof FileCodeCorner> = {
  spec: FileCodeCorner,
  walkthrough: FileBracesCorner,
};

const TYPE_LABEL: Record<string, string> = {
  spec: "Spec",
  walkthrough: "Walkthrough",
};

function downloadArtifact(issueId: string, filename: string) {
  fetchArtifactContent(issueId, filename).then((content) => {
    const blob = new Blob([content], { type: "text/plain;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  });
}

export default function ArtifactList({
  artifacts,
  issue,
}: {
  artifacts: ArtifactSummary[];
  issue: Issue;
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
              className="group flex items-center gap-3 px-3 py-2 rounded-[var(--radius-sm)] hover:bg-[var(--color-hover-surface)] transition-colors"
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
              <button
                onClick={() => downloadArtifact(issue.id, a.filename)}
                className="opacity-0 group-hover:opacity-100 p-1 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-2)] transition-all"
                title={`Download ${a.filename}`}
              >
                <Download className="w-3.5 h-3.5" />
              </button>
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
