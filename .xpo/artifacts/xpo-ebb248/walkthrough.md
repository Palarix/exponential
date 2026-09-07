## Overview

Cleaned up how branch metadata is displayed across the UI — hiding the misleading base-branch SHA when an issue has no commits of its own, and unifying the visual language of inline metadata badges.

## What Changed

### BranchBadge (`web/src/components/ui/BranchBadge.tsx`)

The badge no longer shows a truncated SHA. Instead it shows:

- **Branch icon + commit count** (always, including "0") — the count is the salient information at the board/backlog zoom level, not a hash
- **Blue indicator dot** (absolute-positioned on the top-right corner) when the branch has uncommitted changes but no commits — follows the standard "new/unsaved" dot pattern
- **Rounded-square shape** (`rounded-md`) instead of the previous pill (`rounded-2xl`) — visually distinguishes it from label badges, which stay pill-shaped

The wrapper uses `relative inline-flex` so the blue dot can be positioned outside the badge border without affecting layout.

### PropertySidebar (`web/src/components/IssueDetail/PropertySidebar.tsx`)

The branch info card in the detail sidebar now has three states for the header:

- **commits > 0**: SHA pill (truncated to 7 chars for git convention)
- **commits == 0**: Italic "No commits" text in muted color

Below the branch name, a new uncommitted-changes line appears when `has_uncommitted` is true:

- At 0 commits: "N uncommitted changes" with `+insertions -deletions` pushed right via `justify-between` — since all file stats are uncommitted at this point, the counts are unambiguous
- At >0 commits: Just "Uncommitted changes" text — the file stats mix committed and uncommitted so a count would be misleading

The text uses `--color-text-muted` (no indicator dot) since the sidebar has enough room for descriptive text.

### SubProgress (`web/src/components/ui/SubProgress.tsx`)

Replaced the custom SVG that had a viewBox/size mismatch (16x16 viewBox in a 14x14 element) causing alignment issues with adjacent text. The new implementation:

- Uses the Lucide 24x24 viewBox rendered at 14px — same coordinate space as every Lucide icon, so it aligns natively in flex rows
- **Dashed track** (`strokeDasharray="3 3"`, muted, 40% opacity) as the background ring
- **Solid progress arc** in `--color-accent-primary`, proportional to done/total
- **`CircleCheck`** from Lucide at 100% completion (green)
- Stroke width 2.5 for visual weight

### Artifact Badges (`Backlog.tsx`, `BoardCard.tsx`)

The paperclip + count display was previously unstyled (no border, no background, tiny gap). Now matches the branch badge exactly: `rounded-md`, `gap-1`, `h-6`, bordered, same text color. Applied to both board cards and backlog rows.

## Key Decisions

- **No SHA anywhere on cards**: The SHA is not actionable at the board/backlog level. The commit count is the useful signal for "how much work has happened."
- **Blue dot for uncommitted, not amber**: Blue follows the standard "new/unread" convention. Amber implies a warning.
- **Muted text in sidebar, dot on cards**: Different density contexts call for different indicators — cards need a compact signal, the sidebar has room for words.
- **Unified badge style**: Branch and artifact counts serve the same role (compact metadata) so they should look the same.
