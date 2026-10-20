# Agent chat: action rows show the verb next to the icon

## What

Each row of the actions timeline shows the icon, then a short verb, then the detail: "Read CHANGELOG.md", "Searched for `^### Fixed`", "Merged xpo-f90406". A running call uses the progressive form ("Reading CHANGELOG.md"). Shell rows and unknown tools stay as they are.

## Why

xpo-855e28 replaced the verb with an icon to keep rows short. The icon alone does not tell the user what happened: a pencil next to an issue ID can be a spec, a walkthrough or an update. The verb sits in the tooltip, so the user must hover each row to read the timeline.

## Acceptance Criteria

- [ ] A finished call of a known tool shows icon, verb (past tense), detail: "Read CHANGELOG.md".
- [ ] A running call shows the progressive verb: "Reading CHANGELOG.md".
- [ ] A failed or denied call shows the action form: "Merge xpo-1" with its outcome badge or failure reason.
- [ ] A call waiting for approval shows the action form: "Merge xpo-1" next to "Waiting for approval".
- [ ] A call without a detail shows the bare verb: "Listed issues" / "Listing issues".
- [ ] Bash rows show only the command description, with no verb. This includes failed and denied shell calls.
- [ ] Unknown tools show the fallback icon and the tool name, as today.
- [ ] The collapsed group line (`CurrentCall`) shows the progressive verb of its latest call, even after that call finishes. The past tense appears only in the expanded list.
- [ ] The verb never truncates. The detail truncates first.
- [ ] Every known tool and op has a `doing` form; a test checks each one.

## Flow

1. In `chat-thread-utils.ts`, add `doing` and `bareDoing` to `ToolVerbs` and to the `ops` entries.
2. Fill `doing` / `bareDoing` for every entry in `TOOL_VERBS`, `AGENT_VERBS` and `fileOps` (table below).
3. Change the verbs listed under "Changed verbs".
4. Make `toolSummary` take the step state and pick `doing` for a running call.
5. Make `stepIcon` return `verb` (the text for the row) separately from `detail`. Return no `verb` for Bash when a detail exists.
6. In `ChatThread.tsx` `ToolRow`, render the verb after the icon (`shrink-0`, muted color), then the detail (`truncate`).
7. In `CurrentCall`, render the verb before the detail. Pass `live` to `stepIcon` so a call that ran keeps the progressive verb.
8. Make the icon decorative (`aria-hidden`, no `title`), because the verb is now visible text.
9. Update the tests from xpo-855e28 that assert "no verb as text".

### Verb table (done / running)

| Tool | Done | Running |
|---|---|---|
| Read | Read | Reading |
| Edit | Edited | Editing |
| Write | Wrote | Writing |
| Grep | Searched for · Searched the code | Searching for · Searching the code |
| Glob | Found files matching · Listed files | Finding files matching · Listing files |
| WebSearch | Searched the web for | Searching the web for |
| WebFetch | Fetched | Fetching |
| LSP | Looked up symbols in | Looking up symbols in |
| Skill | Loaded the skill | Loading the skill |
| Agent / Task | Delegated | Delegating |
| AskUserQuestion | Asked | Asking |
| Task* | Added / Updated / Read / Listed … | Adding / Updating / Reading / Listing … |
| xpo show | Read | Reading |
| xpo list | Listed issues matching | Listing issues matching |
| xpo add | Created issue | Creating issue |
| xpo update | Updated | Updating |
| xpo comment | Commented on | Commenting on |
| xpo link / unlink | Linked / Unlinked | Linking / Unlinking |
| xpo start | Started | Starting |
| xpo merge | Merged | Merging |
| xpo spec / walkthrough | Read the spec of · Wrote the spec for | Reading the spec of · Writing the spec for |
| xpo artifact | Read / Added / Deleted / Listed … | Reading / Adding / Deleting / Listing … |
| xpo rationale | Searched design notes for | Searching design notes for |
| xpo git_* / code_history | Checked / Read / Listed / Traced … | Checking / Reading / Listing / Tracing … |

### Changed verbs

- Grep: "Searched the code for" → "Searched for". The code is the default search target; WebSearch keeps "the web".
- xpo show: "Opened" → "Read" (action "Open" → "Read"). Found in tophat: the eye icon and "Opened" did not match; the agent reads the issue.
- `fileOps` write: "Wrote the walkthrough of" → "Wrote the walkthrough for". The same change applies to the spec and to the action form.

## Decisions

- **Explicit `doing` strings, not a generated "-ing" form.** English has too many exceptions (Write → Writing, Run → Running). The table is small.
- **Grep says "Searched for", not "Searched the code for".** Confirmed by the user. Shorter, and the code is the default target.
- **The group line gets the verb too.** Confirmed by the user. The collapsed line and the rows read the same way.
- **The group line stays in the progressive form.** Found in tophat: the line flipped from "Reading foo" to "Read foo" before the next call came in. The active group is still working, so its line says so. A call becomes past tense when it moves into the expanded list.
- **Verb in muted color, detail in secondary color.** The detail is what the user scans for. The verb gives context.
- **Icon becomes decorative.** The visible verb is the accessible name now. A tooltip that repeats the row is noise.
- **Bash keeps the description only.** The description is already a sentence ("Commit the fix on the issue branch"). A failed or denied shell call also keeps the description only; its badge or reason says what happened.
- **A call waiting for approval reads as the action.** "Merging xpo-1" next to "Waiting for approval" claims the merge runs. "Merge xpo-1" matches the approval card.

## Edge Cases

- MEDIUM: Narrow chat window. The verb stays whole; the detail truncates to its readable minimum. Long verbs ("Looked up symbols in") can crowd the detail. Proposed: accept this.
- LOW: Chats saved before detail carried the op (spec/walkthrough) keep the default verb, as today.
- LOW: Bash without a description shows "Ran a shell command" / "Running a shell command".

## Assumptions

- The approval card headline keeps its current action wording, except the spec/walkthrough "write" change above. This issue changes the timeline rows and the group line only.
