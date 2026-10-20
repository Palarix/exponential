# Walkthrough: action rows show the verb next to the icon

## What was built

xpo-855e28 put an icon in place of each action row's verb and moved the verb into the tooltip. The rows were short, but the user had to hover to know what a row did: a pencil next to an issue ID can be a spec, a walkthrough or an update. This issue puts the verb back on the row, after the icon: "Read CHANGELOG.md", "Searched for `^### Fixed`", "Merged xpo-1". A running call reads "Reading CHANGELOG.md".

## How the pieces fit

### Verbs (`web/src/components/Agent/chat-thread-utils.ts`)

- Each `TOOL_VERBS` entry now has three forms. `verb` is for a call that ran ("Read"). `doing` is for a call that runs ("Reading"). `action` is for a call that was only asked for ("Read"). Tools that can have no detail also have `bare`, `bareDoing` and `bareAction` ("Listed issues" / "Listing issues" / "List issues").
- The `ops` entries (spec, walkthrough, artifact) have the same three forms, through the `Verbs` interface.
- `toolSummary(name, arg, running)` picks `doing` when `running` is true.
- `stepSummary(step, live)` decides which form a step gets:
  - A call that did not run (failed, denied, cancelled, expired) reads as the action.
  - A call that waits for approval also reads as the action.
  - A call that runs, or any call shown `live`, reads as `doing`.
  - Every other call reads as `verb`.
- `stepIcon(step, live)` returns `{ icon, verb?, name?, detail }`. `verb` used to be `label`, which fed the tooltip; now it is row text. An unknown tool returns `name` and no verb. A shell call with a description returns no verb.

### Rows (`ChatThread.tsx`, `ToolIcon.tsx`)

- `ToolRow` renders icon → verb → detail. The verb is `shrink-0` and muted (it shimmers while the call runs). The detail is `truncate` in the secondary color, so the detail gives way first in a narrow window.
- `CurrentCall`, the collapsed line of the active group, calls `stepIcon(step, true)`. Its verb stays progressive even after the call finishes.
- `ToolIcon` is decorative now: `aria-hidden`, no `role`, no `title`, no `label` prop. The visible verb is the row's accessible text.

## Key decisions

- **Hand-written `doing` strings, not a generated "-ing".** English has too many exceptions (Write → Writing, Run → Running), and the table is small. A test checks every tool and op, so a new tool without a `doing` form fails the suite.
- **The shell description stays alone.** "Commit the fix on the issue branch" is already a sentence. A failed or denied shell call also shows only its description; the badge or failure reason says what happened.
- **A call waiting for approval reads as the action.** "Merging xpo-1" next to "Waiting for approval" claims the merge runs. "Merge xpo-1" matches the approval card.
- **The active group's line stays progressive (from tophat).** Before, the line flipped from "Reading foo" to "Read foo" between calls. The group is still working, so its line says so. The past tense appears in the expanded list only. The `live` flag carries this from `CurrentCall` into `stepSummary`.
- **Reading an issue reads "Read" (from tophat).** `xpo show` said "Opened", which did not match its eye icon. It now uses Read / Reading / Read.
- **Shorter Grep verb.** "Searched the code for" became "Searched for" (confirmed by the user). WebSearch keeps "Searched the web for".
- **"for", not "of", when writing.** "Wrote the walkthrough for xpo-1" reads better than "of". The approval card headline follows ("Write the spec for xpo-1").

## Acceptance criteria

- [x] A finished call of a known tool shows icon, verb (past tense), detail. Evidence: `stepIcon` test `Read /r/board.go` → `{ verb: "Read", detail: "board.go" }`; `ToolTimeline` test matches `>Read<.*board\.go`.
- [x] A running call shows the progressive verb. Evidence: `stepIcon` "uses the progressive verb while a call runs"; `ToolTimeline` running test matches `>Searching for<.*TODO`.
- [x] A failed or denied call shows the action form. Evidence: `stepIcon(merge, error)` → `verb: "Merge"`; `ToolTimeline` matches `>Merge<.*upe-1` with "failed: no branch".
- [x] A call waiting for approval shows the action form. Evidence: `stepIcon` "reads a call waiting for approval as what is asked" → `"Merge"`.
- [x] A call without a detail shows the bare verb. Evidence: `stepIcon(xpo list)` → "Listed issues"; running → "Listing issues".
- [x] Bash rows show only the description, including failed and denied calls. Evidence: `stepIcon(Bash "Run the tests")` → no verb; denied `rm` → no verb; `ToolTimeline` has no "Ran a shell command".
- [x] Unknown tools keep the fallback icon and the tool name. Evidence: `stepIcon(NotebookEdit)` → `{ icon: "tool", name: "NotebookEdit" }`; `ToolTimeline` contains `>NotebookEdit<`.
- [x] The collapsed group line shows the progressive verb of its latest call, even after it finishes. Evidence: `ChatThread` test "keeps the current call in the progressive form after it finishes" matches `>Reading<.*board\.go`; confirmed in tophat.
- [x] The verb never truncates; the detail truncates first. Evidence: the verb span is `shrink-0`, the detail span is `min-w-0 truncate` in `ToolRow` and `CurrentCall`.
- [x] Every known tool and op has a `doing` form, and a test checks each one. Evidence: `toolSummary while running` table covers all tools and ops.

`make test` passes: lint, 1446 frontend tests, Go tests.
