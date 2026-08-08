## What changed

`xpo init` now updates stale agent instruction sections instead of skipping files that already have them, and the heading was changed from `# Exponential Agent Instructions` to `# Agent Instructions`.

## The problem

`AppendAgentInstructions` was append-only. Once a file had the `# Exponential Agent Instructions` heading, `init.go` skipped it entirely — even if the template had evolved (e.g., the xpo-workflow skill rule was added in a later release). Users who ran `xpo init` early were stuck with outdated instructions forever.

## How it works now

### Dual-heading detection (`agents.go`)

Two new helpers centralise heading detection:

- `findAgentInstructionsOffset(content)` — returns the byte offset of whichever heading appears first: the legacy `# Exponential Agent Instructions` or the current `# Agent Instructions`. It checks the legacy heading first because it contains the new heading as a substring — checking the shorter one first would match inside the longer one at the wrong position.
- `HasAgentInstructions(content)` — boolean wrapper, used by `DetectAgentFiles()` and available to any caller that just needs a yes/no.

### Replace-in-place (`AppendAgentInstructions`)

The function now handles three cases:

1. **File doesn't exist** — creates it with the instructions (unchanged behavior).
2. **File exists, no agent instructions section** — appends the section (unchanged behavior).
3. **File exists, has agent instructions section** — finds the heading offset, keeps everything before it (trimming trailing whitespace), and replaces from the heading to EOF with the current template.

The "heading to EOF" replacement is safe because agent instructions are always appended at the end of the file — that's the convention established by the append-only design, and nothing in the codebase writes content after the section.

### Simplified init loop (`init.go`)

The old code read each agent file, checked for the heading, and branched into skip/append. Now it just calls `AppendAgentInstructions` unconditionally — the function is idempotent. The `strings` import was removed since it's no longer used.

### doctor.go

No code changes needed — it calls `DetectAgentFiles()` which now uses `HasAgentInstructions()` internally, so it recognizes both headings automatically.

## Tests added

Eight new tests in `agents_test.go`:

- `TestHasAgentInstructions_NewHeading` / `_LegacyHeading` / `_NoHeading` — heading detection
- `TestAppendAgentInstructions_CreatesNew` — fresh file creation
- `TestAppendAgentInstructions_ReplacesLegacy` — replaces old heading, preserves user content above, verifies old content is gone and new content (including xpo-workflow rule) is present
- `TestAppendAgentInstructions_ReplacesCurrentHeading` — replaces current heading with updated prefix
- `TestAppendAgentInstructions_AppendsToFileWithoutSection` — appends to a file that has no agent section
- `TestGenerateAgentStub_UsesNewHeading` — verifies the template uses the new heading
