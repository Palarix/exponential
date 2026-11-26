# Release prep for 1.6.0

## What / Why
Cut 1.6.0, a minor release. Its headline feature is Milestones. The Unreleased section has 132 lines that were written one issue at a time. `build.yml` uses the `## [1.6.0]` section verbatim as the GitHub release body. The section must read as a release note for a 1.5.0 user, not as a development log.

The 1.6.0 milestone is 17/17 done. Nothing is in DOING or BLOCKED. `make build` passes on main at 14aa9d8.

## How

### 1. Clean the Unreleased section
Rules (the same rules as xpo-bb6beb):
- **Features new in 1.6.0 get no Changed or Fixed entries.** Fold each fix into the entry it changes, and describe only the final behavior. Keep every issue ID on the merged entry. The Motion preference and the chat glide are new in 1.6.0 (v1.5.0 has no Motion), so:
  - Fold xpo-397365, xpo-ea5d6b and xpo-7ffcf5 (follow fixes) into the glide entry xpo-50b9d9.
  - Fold xpo-8fc8ba (`make frontend` broken by a test fixture during this release) into the `make frontend` entry xpo-a67011.
  - Check xpo-ea2c99 (tiptap crash). If the Markdown composer (xpo-c35d83) caused it, fold its focus and placeholder behavior into xpo-c35d83. If the crash also happens on v1.5.0, keep it in Fixed.
- **Keep real changes to 1.5.0 behavior in Changed or Fixed.** The Filter menu and submenus exist in v1.5.0, so xpo-c5d3ff, xpo-a71a45 and xpo-c78c8d stay in Fixed. Agent chat exists in v1.5.0, so its fixes (xpo-f5df49, xpo-f90406, xpo-804704) stay.
- **Drop superseded intermediate states.** Releases was the working name of Milestones and never shipped (v1.5.0 has no Releases). Replace the Milestones **Migration** sub-bullet with one sentence: "Boards that ran a development build with Releases need `xpo doctor --fix` once."
- **Order:** in Added, put Milestones first, then the factory workflow, then Agent chat as a group, then board UI, then CLI/MCP/API. In Changed, keep the BREAKING workflow entry first, then Agent chat, board UI, tooling.
- **Format:** one blank line between every entry. Several Fixed entries from xpo-ea2c99 onward have no blank line. Every entry ends with its issue ID in parentheses. Use the house writing style. Add no new information.

### 2. Version bump
In the worktree, run `make release-prep VERSION=1.6.0`. It sets `CLIVersion` and turns `## [Unreleased]` into `## [1.6.0] — <date>`, with an empty Unreleased section above it.

### 3. Markers
Run `make cli`, then `./xpo doctor --fix`. It updates the `xpo:begin` / `xpo:skill` markers to 1.6.0. Check that `xpo doctor` passes. Review the diff of every file that `doctor --fix` changes before you keep it, as in 1.5.0.

### 4. README "Upgrading from 1.5.x"
The section exists and covers the workflow split (xpo-f19bfa). Check it against the final changelog. Add an item only for a change that breaks an existing setup.

### 5. Tag and GitHub release (after merge)
1. Commit `.xpo/issues.db` on main, so the tree is clean.
2. On main, run `make release-tag VERSION=1.6.0`.
3. **Ask the user before** `git push origin main v1.6.0`.
4. The tag push runs `build.yml`. It builds 6 archives and `checksums.txt` and creates the GitHub release.
5. Check the release with `gh release view v1.6.0` (assets and body).
6. Remove "plus privacy-respecting telemetry" from the 1.6.0 milestone goal. No telemetry work is in this release.
7. Complete milestone 1.6.0 after the release is published.

## Acceptance criteria
- [ ] The 1.6.0 section has no Changed or Fixed entries for features new in 1.6.0.
- [ ] The 1.6.0 section has no descriptions of designs replaced within the release.
- [ ] Every issue ID from the current Unreleased section appears in 1.6.0.
- [ ] Every entry has one blank line before it.
- [ ] `CLIVersion` is `1.6.0`, and `xpo version` prints it.
- [ ] CHANGELOG has `## [1.6.0] — <date>` with an empty Unreleased section above it.
- [ ] Markers carry 1.6.0, and `xpo doctor` passes.
- [ ] `make test` passes.
- [ ] After merge, tag `v1.6.0` is on main. The tag is pushed only after the user confirms.
- [ ] The GitHub release has 6 archives and `checksums.txt`, and its body is the 1.6.0 section.
- [ ] The 1.6.0 milestone goal has no telemetry phrase, and the milestone is completed.

## Decisions
- **No CHANGELOG entry for this issue.** The README section exists already. Add an entry only if step 4 changes the README.
- **Minor, not major.** The workflow split is breaking for agent instructions. 1.3.0 and 1.5.0 made breaking changes as minor releases too.
- **Rewrite, not reformat.** The per-issue detail stays in each issue's spec and walkthrough.
