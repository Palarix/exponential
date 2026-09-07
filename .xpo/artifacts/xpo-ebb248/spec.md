## What

Hide the branch `head_sha` from UI elements when an issue branch has 0 commits, and unify the visual style of inline metadata badges (branch, artifacts).

## Why

When a branch has no commits of its own, the displayed SHA is the base branch's HEAD — not a commit on the issue branch. Showing it adds visual noise and suggests work has been committed when it hasn't. Additionally, the branch and artifact count badges had inconsistent visual styles (pill vs unstyled), making them harder to scan.

## How

### 1. BranchBadge (board cards + backlog rows)

**File:** `web/src/components/ui/BranchBadge.tsx`

- Removed SHA display entirely — the badge now shows **branch icon + commit count** (always, including "0")
- Blue dot indicator on top-right corner when `has_uncommitted && commits == 0`
- Changed from pill (`rounded-2xl`) to rounded-square (`rounded-md`) to visually distinguish from labels
- Tightened icon-to-count gap from `gap-1.5` to `gap-1`

### 2. PropertySidebar branch card (detail view)

**File:** `web/src/components/IssueDetail/PropertySidebar.tsx`

- **commits > 0**: Show SHA pill (truncated to 7 chars)
- **commits == 0**: Show italic "No commits" text instead of SHA
- Added "uncommitted changes" line below branch name when `has_uncommitted`:
  - At 0 commits: shows file count + insertions/deletions (pushed right)
  - At >0 commits: shows "Uncommitted changes" text
  - Muted text color (no indicator dot — the sidebar has more room for text)

### 3. SubProgress component

**File:** `web/src/components/ui/SubProgress.tsx`

Replaced custom SVG with a hybrid approach using the Lucide 24x24 viewBox (rendered at 14px) for proper alignment with adjacent text:
- **0% done**: Dashed muted track ring only
- **Partial**: Dashed track + solid brand-color progress arc
- **100%**: Lucide `CircleCheck` icon in green
- Stroke width: 2.5

### 4. Artifact count badges (board + backlog)

**Files:** `web/src/components/Backlog/Backlog.tsx`, `web/src/components/Board/BoardCard.tsx`

Unified with branch badge style: `rounded-md`, `gap-1`, same height/padding/border/text color, `Paperclip` icon at size 12.

## Acceptance Criteria

- [x] Board/backlog cards show branch icon + commit count (no SHA)
- [x] Blue dot on badge top-right when uncommitted changes with 0 commits
- [x] Detail sidebar shows "No commits" when commits == 0
- [x] Detail sidebar shows uncommitted changes info below branch name
- [x] Detail sidebar shows SHA pill when commits > 0
- [x] Branch and artifact badges share unified rounded-square style
- [x] SubProgress ring aligns properly with row text using Lucide-compatible viewBox
