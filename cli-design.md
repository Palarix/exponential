# The Exponential CLI First Touch

We need to design a modern, best in class user experience for the xpo CLI tool. Our focus today is on the "first run" experience. This is the very first touch point a user has with our app/system and first impressions are lasting.

## In a folder that is not yet an Exponential project

```bash
$ xpo

Exponential · The software engineering system for human-agent teams

This directory isn't an Exponential project yet.
  xpo init    set up Exponential here
  xpo help    all commands
```

## The `init` command

Should be idempotent: re-running it in an initialized project refreshes the managed files and reports "already initialized, integrations refreshed."

Stamp managed files with the binary version. Any command then does a cheap check and prints one dim line: Integrations are from 0.8.2. Run `xpo init` to refresh.

Have doctor diagnose, with `doctor --fix` for repairs. Model it on flutter doctor: grouped checks, ✓ / ! / ✗, an exact fix command under each failure, and a non-zero exit code for CI.

For this to be safe, everything xpo writes into shared files like CLAUDE.md and AGENTS.md needs delimited managed blocks (`<!-- xpo:begin --> … <!-- xpo:end -->`). Refreshes then replace only that block and never touch user content.

Show the file plan, not "Skill + MCP." The sensitive moment in init is xpo writing into someone's repo, and especially editing their existing `CLAUDE.md`. "Skill + MCP" hides exactly what they want to see. A terraform-style plan builds more trust than any tagline. Also make the harness list a multi-select rather than all-or-nothing Y/n, and show supported harnesses that weren't detected. That makes agent symmetry visible, and doubles as the compatibility proof point.

One caution: check which harnesses actually support project-scoped MCP config. Where one doesn't, that part belongs to setup, and the table shouldn't claim parity.

### Case 1: In a git repo

```bash
$ xpo init

Exponential · The software engineering system for human-agent teams

Project:   ~/code/acme/payments

? Issue prefix › pay
  Issue IDs will look like pay-ab3ef2.

? Agent integrations  (space to toggle, enter to confirm)
  ◉ Claude Code   found
  ◉ Codex         found
  ◯ Gemini CLI    not found

Changes
  + .xpo/                                  
  ~ .gitattributes                         
  + .claude/skills/xpo-workflow/SKILL.md
  + .mcp.json                              
  ~ CLAUDE.md                              
  ~ AGENTS.md                              
  …

? Apply changes? (Y/n)

✓ Exponential is ready

Next
Next
  Commit to share with your team:  git add -A && git commit -m "Add Exponential"
  Create your first issue:         xpo new "…"
```

### Case 2: Not a Git repo

```bash
$ xpo init

Exponential · The software engineering system for human-agent teams

Project:   ~/code/acme/payments

? Exponential needs git. Create a repository here? (Y/n)
...
```


### Notes

- Prefix default and validation: derive the default from the directory name and validate as they type.
- Prompt order: ask every question before writing anything, so `Ctrl-C` leaves no partial state. Your flow already mostly does this; make it a rule.
- Non-interactive mode: support `--yes`, `--prefix`, and `--agents claude,codex`, and switch to non-interactive automatically when `stdin` isn't a TTY.
- Terminal hygiene: respect `NO_COLOR`.

## In a folder that is already an Exponential project

### Case 1: Stale integrations (the common case after a binary upgrade)

```bash
$ xpo init

Exponential is already set up in this project.

Project   ~/code/acme/payments
Prefix    pay

Integrations
  Claude Code   Update available (0.8.2 → 0.9.0)
  Codex         Update available (0.8.2 → 0.9.0)
  Gemini CLI    Found, not set up

? Also set up Gemini CLI? (y/N)

Changes
  ~ .claude/skills/xpo-workflow/SKILL.md
  ~ .codex/…
  ~ CLAUDE.md
  ~ AGENTS.md

? Apply changes? (Y/n)

✓ Updated integrations to 0.9.0

Next
  Commit the update:  git commit -am "Update xpo integrations to 0.9.0"
```

### Case 2: Up-to-date

```bash
$ xpo init

Exponential is already set up in this project (prefix pay)
✓ Integrations are up to date (0.9.0)
```

### Case 3: Manage block was edited by hand

You detect this with a content hash in the marker, e.g. `<!-- xpo:begin 0.8.2 sha256:ab12… -->`

```bash
$ xpo init

Exponential is already set up in this project (prefix pay)

! CLAUDE.md: the xpo section has local edits
? How should xpo handle it?
  › Replace with the 0.9.0 version
    Keep your version
    Show diff
```

### Case 4: Teammate on older binary
Never downgrade! Otherwise two developers on different versions will ping-pong the managed files in every commit.

```bash
$ xpo init

✗ This project's integrations are from xpo 0.9.0. You have 0.8.2.
  No files were changed. 

  Upgrade xpo, then run xpo init again: brew upgrade xpo
```
## The Doctor

Where doctor can link findings like that, it should say so.

```bash
$ xpo doctor

xpo
  ✓ Version 0.9.0 (latest)

Project  ~/code/acme/payments
  ✓ Settings are valid (prefix pay)
  ✓ .gitattributes includes the merge driver
  ✗ Issue log has an invalid entry at line 1,041
    The merge driver isn't configured in this clone, which is the likely cause.

This clone
  ✗ Merge driver isn't configured
    Merges of the issue log may conflict.
  ! Local index is out of date (38 events behind)

Integrations
  ✓ Claude Code   Up to date, MCP server responding
  ! Codex         Update available (0.8.2 → 0.9.0)
  · Gemini CLI    Found, not set up. Add it with xpo init

This machine
  ✓ Shell completions installed (zsh)
  ✓ Claude Code CLI found (used by xpo drive)

2 errors, 2 warnings
Run xpo doctor --fix to resolve 3 of them.
```

## The Fixer

One code path. `doctor --fix` calls the same refresh as init, so there's one writer per scope and no two implementations drifting apart. init is "set up or refresh this project." `doctor --fix` is "make everything doctor found healthy."

```bash
$ xpo doctor --fix

Found 2 errors and 2 warnings

Fixed in this clone
  ✓ Configured the merge driver
  ✓ Rebuilt the local index (1,284 events)

Changes
  ~ .codex/…    Update to 0.9.0
  ~ AGENTS.md   Update xpo section

? Apply changes? (Y/n) y
✓ Updated Codex integration to 0.9.0

Needs your attention
  ✗ Issue log has an invalid entry at line 1,041
    xpo never rewrites the issue log. To inspect the entry:
    xpo log check --line 1041

Fixed 3 of 4 problems
```

Consent follows blast radius.
- Clone-local and gitignored state is fixed without asking. This is the same repair every normal command does silently.
- Committed files always get a plan and a confirm. --yes skips the confirm.
- Anything that expands scope, such as adding a new harness, stays in init.

Doctor is the only command that never self-heals. Otherwise it would report a clean clone that was broken a moment before it ran.

The log is never auto-repaired. An append-only source of truth that the tool rewrites isn't one. Doctor points at the problem and a way to inspect it, and stops there.

Exit codes and CI.
- Exit 1 on errors. Warnings exit 0 unless `--strict`.
- `--json` gives CI a stable check list.
- Checking for a newer xpo release needs the network, so either cache the result or skip it with `--offline`. A diagnose command that hangs on a flaky connection is a bad first impression of its own.

xpo log check is a placeholder until the real inspection command exists.

## Conventions

- Reserve "issue" for xpo issues. Diagnostics are "problems," "errors," and "warnings." This was the worst collision in the earlier drafts.
- Use one term per concept. The terms are integration, merge driver, local index, issue log, xpo section, project, clone, and machine. Filenames appear only where the user needs to find the file.
- Status lines are sentence-case fragments with no period. Explanations beneath them are full sentences.
- Tense depends on what the line describes. Results are past tense ("Updated," "Rebuilt"). Planned changes are imperative ("Update to 0.9.0"). Prompts are questions that name the action ("Apply changes?").
- Section headers are the same in every command. They are Changes, Next, and Needs your attention, so init and doctor --fix read as one system.
- "Found" and "not found" describe detection. "Installed" would overclaim, since detection can miss.
- Commands go on their own line. The one exception is short inline hints ("Add it with xpo init").
- Use → for versions and · as a separator. No em dashes.
