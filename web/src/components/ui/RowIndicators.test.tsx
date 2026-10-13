import { describe, expect, it } from "vitest";
import { renderMarkup } from "../../render-markup";
import { AttachmentIndicator, BranchBadge, LabelBadge } from ".";

const stats = { branch: "xpo-1-fix", head_sha: "abc", commits: 3, files_changed: 2, insertions: 10, deletions: 4, has_uncommitted: false };

// Row indicators share one faint pill on the surface color, after Linear; tooltips have the details.
describe("BranchBadge", () => {
  it("is a pill with the commit count, on the surface color", () => {
    const html = renderMarkup(<BranchBadge stats={stats} />);
    expect(html).toContain("lucide-git-branch");
    expect(html).toContain("rounded-full");
    expect(html).toContain("border-[var(--color-border-subtle)]");
    // At least 32px wide, with a 12px icon.
    expect(html).toContain("min-w-8");
    expect(html).toContain('width="12"');
    expect(html).toContain("color-text-muted");
    expect(html).toContain("bg-[var(--color-surface)]");
    expect(html).toMatch(/<\/svg>3<\/span>/);
    // Shown also with no commits.
    expect(renderMarkup(<BranchBadge stats={{ ...stats, commits: 0 }} />)).toMatch(/<\/svg>0/);
  });

  it("keeps the details in the tooltip", () => {
    expect(renderMarkup(<BranchBadge stats={stats} />)).toContain('title="xpo-1-fix — 3 commits, 2 files, +10 -4"');
    expect(renderMarkup(<BranchBadge stats={{ ...stats, commits: 0 }} />)).toContain("no commits yet");
  });

  it("keeps the dot for uncommitted changes", () => {
    expect(renderMarkup(<BranchBadge stats={{ ...stats, commits: 0, has_uncommitted: true }} />)).toContain('title="uncommitted changes"');
  });
});

describe("AttachmentIndicator", () => {
  it("is a paperclip pill with the count in its tooltip", () => {
    const html = renderMarkup(<AttachmentIndicator count={2} />);
    expect(html).toContain("lucide-paperclip");
    expect(html).toContain('title="2 attachments"');
    expect(html).toContain("rounded-full");
    expect(html).not.toMatch(/<\/svg>\d/);
    expect(renderMarkup(<AttachmentIndicator count={1} />)).toContain('title="1 attachment"');
  });
});

describe("LabelBadge", () => {
  it("sits on the surface color so it stays dark on a highlighted row", () => {
    expect(renderMarkup(<LabelBadge label="bug" />)).toContain("bg-[var(--color-surface)]");
    expect(renderMarkup(<LabelBadge label="bug" borderless />)).not.toContain("bg-[var(--color-surface)]");
  });
});
