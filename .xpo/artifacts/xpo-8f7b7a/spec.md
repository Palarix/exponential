## What

`xpo init` should replace stale agent instruction sections in agent files (CLAUDE.md, AGENTS.md, etc.) instead of skipping them. The heading should change from `# Exponential Agent Instructions` to `# Agent Instructions`.

## Why

When the instruction template evolves (e.g., the xpo-workflow skill rule was added), projects that already ran `xpo init` get stuck with the old version. The idempotency check sees the heading and skips, leaving stale instructions that may lack critical rules.

## Acceptance Criteria

- Running `xpo init` on a project with outdated instructions replaces the section with the current template
- User-authored content above the agent instructions section is preserved
- Both `# Exponential Agent Instructions` and `# Agent Instructions` headings are recognized
- New projects get `# Agent Instructions` as the heading
- The detection in `DetectAgentFiles` recognizes both headings
- `doctor.go` detection continues to work with both headings
- Existing tests pass; new tests cover the replace-on-re-init behavior

## Flow

1. **`GenerateAgentStub(prefix)`** — change heading from `# Exponential Agent Instructions` to `# Agent Instructions`
2. **`DetectAgentFiles()`** — check for both headings (`# Agent Instructions` OR `# Exponential Agent Instructions`)
3. **`init.go` loop** — when the heading is found, call a new replacement path instead of skipping. The replacement should:
   - Find the start of the section (the heading line)
   - Replace everything from that heading to EOF with the new stub
   - This works because `AppendAgentInstructions` always appends to the end, so the section is always the last content in the file
4. **`AppendAgentInstructions(agent, prefix)`** — add a replacement mode: if an existing section is found, replace from the heading to EOF; otherwise append as before
5. **`doctor.go`** — update the detection check to use the same dual-heading logic

## Decisions

- **Replace from heading to EOF** rather than trying to find an end marker — the agent instructions are always appended at the end of the file, so everything from the heading onward is the managed section. This is simpler and more robust than maintaining end markers.
- **No version tracking** — we don't embed a version in the generated instructions. The replacement is always idempotent (same template = same output), so re-running is safe.

## Edge Cases

- **User added content after the agent instructions** — LOW risk. The convention is that agent instructions are at the end. Content after them would be replaced. This is acceptable since the section is clearly marked as managed.
- **File has both old and new headings** — LOW risk. The replacement finds the FIRST occurrence and replaces from there to EOF, which handles this correctly.
