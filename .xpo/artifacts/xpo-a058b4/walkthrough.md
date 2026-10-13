# Walkthrough: the chat agent can load skills

## What was wrong

The board's chat agent skipped the project's xpo workflow in Work Mode. The xpo-c94d85 and xpo-b204f1 chats wrote no spec and no handoff comment, and didn't stop for review. Both wrote a walkthrough at the end.

The cause was the tool list. `claudeArgs` passes `--tools`, which restricts the built-in tools Claude Code offers. The list was `Read,Grep,Glob,Edit,Write,Bash` in Work Mode and `Read,Grep,Glob` in Chat Mode. Neither had `Skill`, so the agent saw no skills at all.

CLAUDE.md did load: the turn runs in the worktree, and the file is checked in. But CLAUDE.md delegates the workflow to the skill (Hard Rule 4: "load the `xpo` skill and follow it"). Without the Skill tool, that rule points at nothing. The agent was left with one vague prompt line, "Follow the project's instructions for working on an issue."

Measured with `claude -p` in this repo: with the old list, CLAUDE.md loads and no skills are seen. With `Skill` added, the `xpo` skill is listed. Codex (`codex exec`, 0.162.1) already sees AGENTS.md and the `xpo` skill in `.codex/skills`, and reads it with shell commands that its sandbox allows. So Codex only needed the prompt change.

## The changes

**`internal/chat/profile.go`**
- `claudeTools` adds `Skill` in both modes.
- `claudeAllowed` puts `Skill` on the allow list in both modes. Anything not on the allow list becomes an approval card through `--permission-prompt-tool stdio`. Loading a skill only adds instructions, so a card for it would be noise.

**`internal/chat/prompt.md`**

The system prompt is shared by Claude and Codex and covers both modes.
- **Chat Mode:** "Before you plan, spec or start work on an issue, load the `xpo` skill." Specs are written in Chat Mode, before `start`.
- **Work Mode:** "Before you change anything, load the `xpo` skill: it is the workflow for working on an issue. Follow it and the project's instructions, including where they say to stop and wait for the user." The last clause targets the observed failure: the agent ran straight through to the walkthrough without stopping for review.

**`web/src/components/Agent/chat-thread-utils.ts`**
- `TOOL_VERBS` gets `Skill`, so a skill load reads "Loaded the skill xpo" (approval form: "Load the skill xpo"), not a bare "Skill xpo".
- The Go stream parser already lists `skill` in `toolDetailKeys`, so the skill name arrives as the call's detail without backend changes.

## Decisions

- **Skills in both modes.** Planning and spec writing happen in Chat Mode, so the workflow must be loadable there too.
- **Every skill is allowed, global ones included.** The chat behaves like the CLI. The project and the user decide which skills exist.
- **The prompt names `xpo`.** The first draft said "the project's workflow skill", but nothing tells the agent which skill that is. `xpo init` installs the `xpo` skill in every project it sets up, so the chat can name it. A project without it falls back to its instructions.

## Tests

- `profile_test.go` pins the tool lists and `Skill` in the allow list for both profiles.
- `args_test.go`:
  - `TestSystemPromptFor` pins both prompt sentences for both providers.
  - The grants test now expects `Skill` between the xpo wildcard and the chat's grants.
- `mode_test.go` updates the old-chat tool list.
- `chat-thread-utils.test.ts` covers the Skill labels.

## Acceptance Criteria

- [x] Claude turns in both modes get the `Skill` tool, and it's on the allow list, so it raises no approval card. Evidence: `profile_test.go`. A live Chat Mode turn called `Skill xpo` with no card.
- [x] The Work Mode prompt names the `xpo` skill, says to load it before changing anything, and says to follow it, including where it says to stop for the user. Evidence: `prompt.md`, `TestSystemPromptFor`.
- [x] The Chat Mode prompt says to load the `xpo` skill before planning, spec or start. Evidence: `prompt.md`, `TestSystemPromptFor`.
- [x] Tests pin `Skill` in `claudeTools` and `claudeAllowed` for both profiles, and pin the prompt sentences.
- [x] Live check: on the real repo (worktree build on :8099), a planning message made the agent call `Skill` with `xpo` first; the thread recorded the call. The user reviewed the change and approved it.
- [x] `make test` passes.