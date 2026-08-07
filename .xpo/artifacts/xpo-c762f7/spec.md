# Spec: .gitattributes merge=union for issues.db

## Changes

### 1. `InitProject` in `setup.go`

After the `.gitignore` step, add a `.gitattributes` step that ensures the line `.xpo/issues.db merge=union` is present. Same pattern as `.gitignore`: read existing file, check for the entry, append if missing.

### 2. `xpo doctor` in `doctor.go`

After the `.gitignore` check, add a `.gitattributes` check. If the `merge=union` entry is missing, auto-add it (same auto-fix pattern as `.gitignore`).

## Acceptance Criteria

- [ ] `xpo init` creates/updates `.gitattributes` with `.xpo/issues.db merge=union`
- [ ] `xpo init --force` on existing projects adds the rule without breaking other entries
- [ ] `xpo doctor` detects and auto-fixes missing entry
- [ ] `make test` and `make build` pass
