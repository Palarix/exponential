# Spec: timeline view with tool icons for the work steps

## What

A turn shows the agent's work as it happens. Narration ("I'll load the workflow and read the issue first.") is a regular chat message. Under each narration, the tool calls that follow it collapse into one line: the current call with a timer while it runs, then "k actions" once the next narration arrives. Each line expands into a timeline of its calls, with an icon in place of each verb.

(Revised after the first tophats: the first version kept one "Worked for …" list per turn, with narration as dots on the timeline.)

## Why

Observability: the user should follow what the agent does, live. A single collapsed "Working for 40s" hides it; an expanded list of 40 rows buries the narration that explains it. The verbs ("Searched the code for", raw "Bash") take most of a row; icons are shorter and scan faster.

## How

### 1. Tool icons (`chat-thread-utils.ts` + new `ToolIcon.tsx`)

- Add an `icon` key to each `TOOL_VERBS` entry (a string ID, so the utils stay free of React). A small `ToolIcon.tsx` maps the ID to a lucide icon.
- New util `stepIcon(step)` → `{ icon, label, detail }`:
  - `label` is the verb from `stepSummary` (the "ran" verb, or the action verb for a failed or denied call). It becomes the tooltip (`title`) and `aria-label` of the icon.
  - Bash gets the verb "Ran a shell command" / "Run a shell command" as its label, not "Bash".
  - Unknown tools: `icon: "tool"` (Wrench), label = tool name. The row also shows the tool name as text before the detail, so the user still knows what ran.

Icons group tools by what they do, so the same kind of action always looks the same (revised after the first tophat):

| Group | Icon | Tools |
| --- | --- | --- |
| `read`: look at something known | Eye | Read, xpo show, history, git_log, git_show, code_history |
| `edit`: change something | Pen | Edit, Write, xpo update |
| `add` | Plus | xpo add |
| `comment` | MessageCircleText (needs lucide-react ≥ 1.57; upgraded from 1.17 in this issue) | xpo comment |
| `link` / `unlink` | Link / Unlink | xpo link, xpo unlink |
| `git_status` | SquareDot | xpo git_status |
| `git_diff` | FileDiff | xpo git_diff |
| `spec` | FileSliders | xpo spec (reads and writes) |
| `walkthrough` | FileTerminal | xpo walkthrough (reads and writes) |
| `artifact` | Paperclip | xpo artifact (reads and writes) |
| `search`: find things | Search | Grep, Glob, xpo list, xpo rationale |
| `shell` | SquareTerminal | Bash |
| `skill` | BookOpen | Skill |
| `start` | Play | xpo start |
| `merge` | GitPullRequest | xpo merge |
| `tool` | Wrench | unknown tools |

### 1b. Verbs follow the operation (spec, walkthrough, artifact)

Before this issue, the row of an `xpo spec`, `xpo walkthrough` or `xpo artifact` call always read "Read the … of", even for a write, and showed no issue: the detail keys (`mcpToolDetailKeys`) had neither `operation` nor `issue_id`.

- Backend (`internal/exponential/claudestream.go`): add `issue_id` to `mcpToolDetailKeys`, after `id`. When an MCP tool's input has an `operation` string, the detail is `<operation> <detail>` (`write xpo-1`). `ToolDetail`, `MCPToolDetail` (Codex) and the partial-input scan share this, so Claude, Codex and approval cards agree.
- Frontend (`chat-thread-utils.ts`): for these three tools, a leading known operation word picks the verb and is cut from the detail. With no known operation (older chats), the current wording stays.

| Tool | Operation | Ran | Asked |
| --- | --- | --- | --- |
| spec / walkthrough | read | Read the spec of | Read the spec of |
| | write | Wrote the spec of | Write the spec of |
| | delete | Deleted the spec of | Delete the spec of |
| artifact | read | Read an artifact of | Read an artifact of |
| | add | Added an artifact to | Add an artifact to |
| | delete | Deleted an artifact of | Delete an artifact of |
| | list | Listed the artifacts of | List the artifacts of |

### 2. Turn layout (`ChatThread.tsx`)

```
[user bubble]

I'll load the workflow and read the issue first.        ← narration: a message
[≡] 3 actions                                     ›  ← finished group
Now the code. The timeout lives in the handler.          ← narration: a message
⟳ [🔍] checkoutTimeout\s*=                       12s  ›  ← active group
```

- New util `turnSegments(turn)` splits the steps into, in order, text segments (consecutive text steps) and tool groups (consecutive tool steps).
- **Text segments** render as messages, in the answer's markdown style, streaming live. There is no separate "answer": the text after the last group is just the last message. The copy button copies that last text when the turn is done.
- **A tool group** renders as one line with a chevron; clicking it expands the group's calls as a timeline:
  - **Active** (the turn runs and the group is the last segment): the latest call's icon and detail, the turn's timer on the far right. No spinner: the text shimmers (`.text-shimmer` in `index.css`, a brighter band sweeping across muted text, like the Claude Code CLI; it also runs under `prefers-reduced-motion`, as the only sign of progress). "Thinking" shimmers the same way. With an approval waiting: amber dot and "Waiting for your approval".
  - **Finished**: "k action(s)", plus a red "· n failed" when calls failed. No auto-expand on failure: agents make and recover from mistakes often, so a failure is not a reason to open the list.
- **Thinking line**: while the turn runs and no group is active, and the agent is thinking or has produced nothing yet, a line with a Lightbulb icon, "Thinking", and the timer on the right. While narration streams, no line: the text is the progress.
- **Turn duration**: when the turn ends, its total time ("1m 24s") sits on the far right of the row under the last message, beside the copy button. The "Worked for …" header is gone.
- New util `formatDuration(seconds)`: "12s", "1m 24s", "1m".
- `collapsedFailureNote` becomes per group; `currentStep(turn)` is replaced by the group's last call.

### 3. Group timeline

```
├─ [icon] file.go
├─ [icon] Run the tests
└─ [icon] pattern
```

- The active group's timeline leaves out its latest call (shown on the line, or by the approval card), so a running call never appears twice. An active group with only that call has no chevron.
- One continuous 1px rail drawn by each row's own segment; the last row's segment stops at its middle (`└─`). Each call branches off the rail with its icon, then the detail.
- States: running → its icon, with shimmering text; failed / denied by rules → only the icon in the error color; the text and the reason ("failed: …") stay in the normal text colors; approval outcome badge and "Waiting for approval" as before.

### 4. Tophat page (removed)

During review, a dev-only page (`web/tophat.html`) rendered sample turns without the backend. It was removed before merge at the user's request.

### 5. Approval cards keep their text

The card headline is a sentence the user is asked to approve ("Merge upe-1"). An icon alone would make the decision less clear. No change to `ApprovalCard`.

## Acceptance criteria

- [ ] Narration renders as regular chat messages, streaming live.
- [ ] Tool calls after a narration collapse into one line: while active, the current call (icon + shimmering detail) + timer on the far right, no spinner; once the next narration arrives, "k actions" (+ "· n failed").
- [ ] Each group line expands into a timeline of its calls: a continuous rail with branch entries.
- [ ] "Thinking" (Lightbulb) with the timer shows while the turn runs and no group is active, unless narration streams.
- [ ] Waiting for an approval shows on the active line; the card shows the action.
- [ ] Failures do not auto-expand; the group line shows the count.
- [ ] A finished turn shows its total time on the far right under the last message.
- [ ] Tool calls show an icon in place of the verb for all known tools (Read, Edit, Write, Bash, Grep, Glob, Skill, xpo tools); each icon has a tooltip and accessible label with the verb; unknown tools show a fallback icon plus the tool name.
- [ ] Running, failed, denied, and approval states stay visible in the timeline.
- [ ] spec, walkthrough and artifact rows name the operation ("Wrote the spec of xpo-1"), with the issue as detail.
- [ ] Tests updated (`chat-thread-utils.test.ts`, ChatThread tests).

## Decisions (confirmed)

1. Approval cards keep their text headline (no icons).
2. Failed calls keep their tool icon, tinted red, instead of the `X`; the red "failed: …" reason stays.
3. (Superseded by the turn layout in section 2.)
4. Icons as in the table above: one icon per kind of action, not per tool (user feedback on the first tophat). Merge uses GitPullRequest. Second round: link, unlink, artifact, add and comment keep their own icons. Third round: git_status (SquareDot) and git_diff (FileDiff) too. Fourth round: spec (FileSliders) and walkthrough (FileTerminal).
5. Turn layout (section 2), after the first tophats: narration as messages; one group line per run of tool calls; Thinking uses Lightbulb; no auto-expand on failure; the turn's total time moves under the last message.
6. One icon on the active line: the call's icon; progress is a text shimmer, not a spinner (also for running rows in the timeline and for Thinking).
7. The expanded active group lists only the calls before the one on its line (user feedback: the running call showed twice).
8. A finished group reads "k actions", not "k tool calls": the user-facing word for what the agent did; "tool call" is jargon.
9. A failed call: only its icon is red; its text and reason read as normal text (failures are common and often harmless).
