# Agent chat: Work Mode can't load skills

## What

The board's chat agent can load the project's skills in both modes. In Work Mode, its system prompt tells it to load the project's workflow skill before it changes anything.

## Why

In Work Mode the agent skipped the project's xpo workflow. The xpo-c94d85 and xpo-b204f1 chats wrote no spec and no handoff comment, and didn't wait for approval. Both wrote a walkthrough at the end, and xpo-c94d85 merged.

The Claude turn runs with `--tools "Read,Grep,Glob,Edit,Write,Bash"` (Chat Mode: `Read,Grep,Glob`). That list leaves out the `Skill` tool, so the agent sees no skills. CLAUDE.md loads, but its Hard Rule 4 says "load the `xpo` skill and follow it", which the agent can't do. All it has left is the system prompt's "Follow the project's instructions for working on an issue."

## Measurements

Measured on 2026-10-13 in this repo:

| Agent | Run | Project instructions | Skills seen |
|---|---|---|---|
| Claude Code | `claude -p --tools Read,Grep,Glob,Edit,Write,Bash` | CLAUDE.md, yes | none |
| Claude Code | same, plus `Skill` | CLAUDE.md, yes | `xpo` and the user's global skills |
| Codex 0.162.1 | `codex exec --sandbox read-only` | AGENTS.md, yes | `xpo` (from `.codex/skills`) and global skills |

Codex already finds the skill and reads it with shell commands, which its sandbox allows in both modes. So Codex only needs the prompt change.

## Acceptance Criteria

- [ ] Claude turns in both modes get the `Skill` tool. It is on the allow list, so loading a skill raises no approval card.
- [ ] The Work Mode prompt names the `xpo` skill: load it before changing anything, then follow it and the project's instructions, including where they say to stop and wait for the user.
- [ ] Chat Mode prompt: when the user asks to plan, spec or start work on an issue, load the `xpo` skill first.
- [ ] Tests pin `Skill` in `claudeTools` and `claudeAllowed` for both profiles, and pin the prompt sentence.
- [ ] Live check: a Claude Work Mode chat on the board loads the `xpo` skill (a Skill call shows in the thread) and follows the workflow.
- [ ] `make test` passes.

## Flow

1. `profile.go`:
   - `claudeTools` returns `Read,Grep,Glob,Skill` in Chat Mode and `Read,Grep,Glob,Edit,Write,Bash,Skill` in Work Mode.
   - `claudeAllowed` adds `Skill` in both modes.
2. `prompt.md`:
   - Work Mode bullet: "Before you change anything, load the `xpo` skill: it is the workflow for working on an issue. Follow it and the project's instructions, including where they say to stop and wait for the user."
   - Chat Mode bullet: "Before you plan, spec or start work on an issue, load the `xpo` skill."

3. The thread shows a Skill call as "Loaded the skill xpo" (approval form: "Load the skill xpo"). The Go stream parser already passes the `skill` input as the call's detail.

## Decisions

- **Skill in Chat Mode too.** Loading a skill only adds instructions, and planning and writing specs happen in Chat Mode.
- **Allow every skill, not only `xpo`.** The project and the user decide which skills exist. Global skills come with the user's setup, as they do in the CLI.
- **Name the `xpo` skill in the prompt.** "The project's workflow skill" doesn't tell the agent which skill that is. `xpo init` installs the `xpo` skill in every xpo project, so the chat can name it. If a project lacks the skill, the agent falls back to the project's instructions.

## Out of scope

- Sandboxing Bash (xpo-3d7608).
- Merging without approval: `merge` already asks the user. The prompt change makes the agent stop earlier, at review.