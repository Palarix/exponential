# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Agent chat: an "All chats" page (`#/agent/history`), opened from the Agent view header and the status-bar history menu. It lists every chat in a full-width table.
  - **Find:** search by title, Active/Archived tabs, and agent, period (7 days, 30 days, all) and issue filters (`?issue=<id>`).
  - **Usage:** a summary strip totals chats, tokens in/out and cost for the filtered rows. Hovering a chat's agent shows its turns, tokens and cost. Chats without a reported cost (Codex) show "—".
  - **Manage:** rename, archive, unarchive and delete per row, or in bulk. Checkboxes take the place of the state icon on hover.
  - **Archive:** an archived chat leaves the recent lists, the switcher and the pills, and opens read only with an Unarchive banner. Archive is refused while a reply runs or the chat is in Work Mode. An issue's chat list shows "N archived chats".
  - **API:** `POST /api/local/chats/{id}/archive` and `/unarchive`. Chats carry `archived_at` and `usage` (turns, tokens, cost), and older chats get their totals from their logs. (xpo-1446f2)

- Agent chat: chat chips and the recent-chats lists show a red alert circle when a chat's last turn failed. Its tooltip gives the error. It stays until the next turn starts; opening the chat does not clear it. Interrupted chats keep the red dot. Chats include `last_error`. (xpo-c56e70)

- Agent chat: Claude can ask you questions with `AskUserQuestion`, in both modes. The chat shows a question card: each question with its options (one pick or several), and "Other…" for a free-text answer. Send returns the answers to the agent, and Skip tells it you skipped. The thread keeps each question with your answer. `POST /api/local/chats/{id}/approvals/{approval}` takes `{"decision":"answer","answers":{…}}`, and `approval_resolved` events carry `answers`. Claude only; Codex follows in xpo-eafa9b. (xpo-447687)

- Agent chat: Claude gets more built-in tools. Both modes get WebSearch, WebFetch and LSP. Work Mode also gets the task list (TaskCreate, TaskUpdate, TaskList, TaskGet) and subagents (Agent). WebFetch asks before every fetch, and the other new tools run without a card. Codex turns get live web search in both modes. The transcript shows a verb and icon for each new tool, with a new bot icon for subagents. (xpo-80bf05)

- Agent chat: the browser tab's favicon gets a dot when a chat needs you: amber while any chat waits on an approval, blue when a reply finished while the tab was hidden. The blue dot clears when you return to the tab. The chat list now keeps polling in hidden tabs. (xpo-bbcab4)

- Agent chat: chat chips in the status bar and the recent-chats lists show a status dot: yellow while an approval waits, red when interrupted, blue for a finished reply you have not seen. The spinner stays for running chats. A reply counts as seen only while its chat is shown and the window has focus; the blue dot clears 1.5s after you come back. This replaces the yellow shield icon. (xpo-f2d4c8)

- Merged issues get a "View changes" button that opens a read-only Merged view: the walkthrough, commits, diff and conversation, read from the merge commit, with who merged it and when. `GET /api/issues/{id}/merged` returns the merge's commits and diff, and issues in the API carry `merged: true`. (xpo-64be51)

- MCP `add`/`update` take `before` or `after` (an issue ID) to place an issue directly above or below another issue in the manual order, so agents can reorder the board. The CLI gets the same as `xpo add`/`xpo update --before/--after` and in `--json` payloads. An issue already in place is left alone ("X is already after Y"). (xpo-61232d)

- Agent chat: issues show the chats that worked on them.
  - **Chat pill:** backlog rows and board cards show a chat pill with the chat count. Its icon shape gives the most urgent state: a dot for waiting on your approval, three dots for running, a warning for interrupted. A click opens the chat in Work Mode on the issue, or a searchable list of the issue's chats.
  - **Issue details:** a "k chats" row below Relations opens the same list.
  - **Context menu:** right-click an issue for "Open work chat" and a "Chats" submenu.
  - **Enter work mode on a past chat:** when no chat is working on the issue, a chat's row actions (right-click, `⋯` or Shift+F10) put it back into Work Mode with its conversation, prefilled with "Let's continue with xpo-…". Continue Work still opens the current work chat or starts a new one.
  - **Cold cache divider:** a dashed "Context cache expired" divider appears in a chat once its prompt cache has expired, so you know the next message re-sends the whole conversation. Claude's cache lifetime comes from what each turn reports (1h or 5m); Codex assumes 5 minutes.
  - **API:** `POST /api/local/issues/{id}/work-chat` takes an optional `{"chat_id"}` to enter Work Mode with an existing chat. Chats include `cache_ttl` and `cache_expires_at`; `turn_done` events include `cache_ttl`.

  (xpo-219de9)

- Agent chat: Chat Mode and Work Mode in one chat. A chat can now implement an issue in its worktree without starting a second session, and the conversation carries over.
  - **Chat Mode:** the default. The agent reads the repo and changes issues. `merge` is denied, and `start` asks first: the card reads "Enter Work Mode on xpo-…?".
  - **Work Mode:** the agent runs in the issue's worktree. Read, Edit, Write and Bash run there without a card; Edit or Write outside the worktree, and `merge`, ask first.
  - **Start Work:** opens a new chat in Work Mode, titled "<issue ID> <issue title>". The first message is prefilled but not sent. If a chat is already working on the issue, Start Work opens that chat.
  - **Continue Work:** on an in-progress issue, opens its Work Mode chat, or creates one on the existing worktree without changing the issue.
  - **Composer:** the "Read-only" badge is now a mode segment: "Chat", or "Work · xpo-…" with "Go to issue" and "Exit work mode". Exit work mode keeps the worktree, and the issue stays in progress.
  - **Transcript:** a divider marks each switch. After a successful `merge`, the chat goes back to Chat Mode on its own.
  - **Codex:** read-only sandbox in Chat Mode, workspace-write in Work Mode. The approval policy is now `on-request`, so `start` and `merge` ask instead of failing.
  - **Remote mode:** Work Mode is unavailable, and the agent cannot call `start`.
  - **API:** `POST /api/local/issues/{id}/work-chat`, `POST /api/local/chats/{id}/leave-work`; chats include `mode`, `work` and `worked_issues`.

  (xpo-00f0e5)

- Agent chat: approval cards. The agent asks before a tool call runs, and the chat shows a card with "Deny", "Allow for this chat" and "Allow".
  - **First use:** `merge` in a chat now asks before it runs, instead of running silently.
  - **Claude Code:** turns run with `--permission-prompt-tool stdio`; the prompt goes in as a stream-json message and stdin stays open for the answers.
  - **Codex:** command, file-change and MCP-tool approval requests are forwarded to the same cards instead of being declined.
  - **Allow for this chat:** xpo keeps the grant with the chat and passes it to every later turn. It never writes to Claude Code's settings files.
  - **Card:** one row, "Merge upe-1 via `xpo merge`". A shell call shows its command; other tools show only the arguments the headline does not.
  - **States:** cancelling a turn resolves its cards as "Cancelled"; a card whose turn ended shows "Expired". Chat lists mark chats waiting for an approval.
  - **API:** `POST /api/local/chats/{id}/approvals/{approval_id}` with `{decision}`; chat lists include `waiting`.
  - **Work list:** calls that did not run read as what was asked ("Merge upe-1", not "Merged"). Failed rows end with "failed: <reason>", and the full error opens in a card on hover, focus or click. A turn with a failure opens its work list; collapsed, it shows "· k failed". A call you denied is not counted as a failure.
  - **Layout:** in the Agent view, the thread and the reply box share the same edges.

  (xpo-7bc86e)

- Agent chat: the composer footer is one row with the model, effort and permissions: `✳ Claude Opus 5.5 ⌄ | Medium · 1M ⌄ | 🔒 Read-only`.
  - **Model:** one menu lists every installed agent's models, grouped by agent, with the Claude and ChatGPT marks. Models show their full names, learned from the model each finished turn actually ran; the default reads "Name (Default)". Fable is added to the Claude list.
  - **Effort:** Medium is pre-selected and there is no "Default effort" option. The trigger also shows the model's context window once known.
  - **Read-only badge:** explains that code changes happen in work sessions.
  - **Per-chat selections:** opening a chat, in the popup or the full-page view, restores the agent, model and effort its last message used. Your last pick is remembered for new chats.
  - **Polish:** the chat popup is 460px wide. The empty Agent page is spaced evenly. The recent chats popover opens above its button. Scrolling menus have an even inset on every side.

  (xpo-74390c)

- Agent chat supports Codex as well as Claude Code, with an agent picker in the composer.
  - **Detection:** the board finds which of `claude` and `codex` are installed. `chat.agent` sets the default; an unknown or uninstalled agent logs a warning and falls back to the first installed one.
  - **Codex:** runs through `codex app-server` with a read-only sandbox and approval policy `never`. Answers stream live, tool calls show in the work list, and follow-ups continue the same Codex thread. The xpo MCP tools are pre-approved; in remote mode the token is passed in an environment variable.
  - **Models:** Codex models and their effort levels come from `model/list` at board start.
  - **Composer:** the agent picker shows when 2+ agents are installed and locks once a chat has messages. Changing the agent resets model and effort.
  - **API:** `GET /api/local/agent` lists every provider with its models and efforts. `POST /messages` takes `provider`; sending to a chat that belongs to another agent returns 409.
  - **Storage:** chat meta stores a generic `session_id`. Chats saved with `claude_session_id` still resume.

  (xpo-5b3157)

- Remote mode guards: actions that would run git in the server's checkout are hidden or refused until Team Mode (xpo-8757a9).
  - **Board:** in remote mode, Start Work and Review Changes are hidden. A new local route, `GET /api/local/mode`, tells the UI which mode the board runs in.
  - **`xpo serve`:** `POST /api/issues/{id}/start` and `/merge` return 501. Its `/mcp` no longer offers the `start` and `merge` tools.
  - **`xpo mcp`:** with `remote.url` set, it does not offer `start` and `merge`. The CLI commands are unchanged.

  (xpo-a60065)

- Read-only git MCP tools: `git_status`, `git_diff`, `git_log`, `git_show` and `code_history`. The agent chat can now show what an issue changed and which issues touched a file or lines.
  - **Where:** the local `xpo mcp` server only. `xpo serve` does not offer them on `/mcp`. A chat in remote mode gets them once xpo-6dd7a0 lands.
  - **Safety:** xpo builds every git command. The caller gives only an issue ID, a path, a SHA, a line range and a limit. External diff, textconv, fsmonitor and hooks never run, and the tools take no index locks.
  - **Output:** capped at 64 KiB, with `truncated: true` and a marker line.
  - **Remote mode:** the issue is looked up on the server; git always runs against the local checkout.

  (xpo-a7f115)

- Agent chat: denied and failed tool calls show in the work list.
  - **Marker:** a red × replaces the bullet. The error text shows below the step, clamped to 3 lines. A denied call's text starts with "Denied:".
  - **Header:** the collapsed "Worked for N seconds" line adds "· N failed" in red.
  - **Timing:** a step keeps its spinner until the tool result arrives, not only until Claude finishes writing the tool input.
  - **Events:** `tool_call` and `tool_start` carry `tool_id`. `tool_call` has `status` `done`, `error` or `denied`, and a `message`.

  (xpo-8673f4)

- Issue ID chips: issue IDs show as accent-tinted chips that open the issue on click and show a preview on hover.
  - **Where:** agent chat answers, your own messages, tool rows, issue descriptions, specs, walkthroughs and comments.
  - **Code:** IDs in inline code are chips too. IDs in fenced code blocks stay plain text.
  - **Links:** an ID inside an existing link is not wrapped again. A Markdown link to an issue with its own text keeps that text.
  - **Unknown IDs:** a gray chip with a "not found" tooltip.

  (xpo-289f40)

- Agent view:
  - **Opening it:** **Agent** is the first entry in the personal section of the sidebar (`g a`). It opens a full-page chat at `#/agent`, and each chat has its own URL (`#/agent/<id>`).
  - **Empty state:** a wide composer, suggestion cards, and the 5 most recent chats.
  - **Header:** a "Title ⌄" switcher lists recent chats. A "⋯" menu (also in the popup) renames chats inline or deletes them after a confirmation. Deleting a running chat stops it first.
  - **Popup:** a new expand button opens the popup's chat in the Agent view.
  - **Status bar:** a history button lists recent chats.
  - **Pruning:** on start, `xpo board` deletes idle chats older than 30 days and keeps at most 200.
  - **Routes:** `PATCH` (rename) and `DELETE` on `/api/local/chats/{id}`.

  (xpo-c047ce)

- Agent chat in `xpo board`:
  - **Opening it:** an **Agent** button in the status bar (or ⌘J / Ctrl+J) opens a chat popup from any view. The popup sends the open issue as context.
  - **Agent:** replies come from your local Claude Code (`claude -p`, one process per turn, resumed with `--resume`), running in the project root with your settings. It can read code (Read, Grep, Glob) and use the xpo MCP tools. It cannot run commands or edit files.
  - **Thread:** replies stream in, and tool calls collapse into a "Worked for N seconds" row.
  - **Composer:** model (Haiku, Sonnet, Opus) and effort pickers.
  - **Pills:** minimized chats stay as pills in the status bar.
  - **Storage:** chats are stored per user in `.xpo/chats/` (git-ignored).
  - **Routes:** `/api/local/agent` and `/api/local/chats/*`, with an SSE stream per chat. The board answers them in remote mode too, where the agent uses the remote server's `/mcp`.
  - **Config:** `chat.agent` and `chat.model`.

  The stream-json parser is now shared by drive and chat (xpo-56ae3a)

- Issue drafts: the new issue dialog has a "Save draft" button, and the discard prompt offers "Save draft" too. Drafts are stored per user in `.xpo/drafts/` (git-ignored, outside the event log). A "Drafts" sidebar item with a count badge appears while drafts exist (`g r`). Reopen a draft to keep editing, create the issue from it, or discard it. New `/api/issue-drafts` endpoints (`GET`/`POST`/`PUT`/`DELETE`, `issue.create` capability) and a `draft_changed` SSE event (xpo-758398)

### Changed

- Agent chat: the board no longer deletes chats older than 30 days at startup. It archives them instead, skipping running and Work Mode chats, and only you delete chats. Chat titles keep up to 200 characters instead of 60. The "Delete branch" checkbox in the merge view uses the accent color. (xpo-1446f2)

- Agent chat: approval cards are reworked. (xpo-aaf8b1)
  - **Layout:** the headline, the call's arguments, then the buttons. A call without arguments fits on one row.
  - **Arguments:** every argument shows as key/value rows in a monospace panel, so values like `force true` no longer read as part of the headline.
  - **Tool name:** "via" matches the headline's size, and the tool name sits in a code chip.
  - **Buttons:** `[Deny] [Allow once ▾]`. "Allow for this chat" moved into the chevron's menu.
  - **Keyboard:** the waiting card takes focus, with the composer's focus border. Enter allows once and Esc denies. Focus moves to the next card, then back to the composer. The card never takes focus from a text field you are typing in.

- Buttons look the same on any surface. (xpo-aaf8b1)
  - `secondary` is a wash of its own text color for rest, hover and pressed, with an optional `bordered` outline and a `tone` (`danger`, `warning`) for colored text.
  - `ghost` uses the same wash on hover.
  - `primary` darkens when pressed.
  - `danger` rests a little darker and hovers a little lighter than its base color, and darkens when pressed.
  - A new `warning` variant works the same way, with dark text.

- Agent chat: the question card asks one question at a time. (xpo-46afc0)
  - **Tabs and review:** with several questions, tabs at the top list the questions (✓ when answered) and a Review tab at the far right. The review step shows each question with your answer, or "Not answered", and a "Change" button on hover. "Submit answers" sends from there. A single question has no tabs and no review step.
  - **Footer:** "Chat about this" (left) replaces Skip, and Next (right) moves forward; there is no Previous. "Chat about this" now tells the agent to end its turn and wait for your message, and the thread reads "Chat instead".
  - **Faster answers:** a single-choice pick moves on by itself after 250 ms. With focus in the card, ←/→ switch steps and 1–9 pick an option; each option shows its number. The "Other…" field sits inside its option.
  - **Look and motion:** recessed tab and footer bands, raised option rows with an accent tint when picked, and short animations (the pick lands, the tab's ✓ pops, the next step slides in from its side). The animations are off under reduced motion.

- Agent chat: on a DOING issue, Continue Work renders as a secondary button while a chat is in Work Mode on the issue, and stays primary otherwise. Clicking it works as before. (xpo-374792)

- A `blocked_by` blocker now counts as resolved only when it is DONE. CANCELED and DUPLICATE blockers still block, until you remove the link or point it at another issue.
  - **Board:** Start Work is disabled while any blocker is unresolved. Hovering it shows a tooltip that lists each blocker and its status, with a one-click Remove link for canceled and duplicate blockers.
  - **`start` and drive:** `start` refuses such issues, and the error says to remove the link. Drive skips them.

  (xpo-35943d)

- Dependencies upgraded. Frontend: vite 8 (Rolldown; vendor chunks move to `rolldownOptions.output.codeSplitting.groups`), @vitejs/plugin-react 6, eslint 10, fractional-indexing 4, React 19.3, recharts 3.10, tiptap 3.32 and the rest to latest. TypeScript stays on 5.9 (typescript-eslint supports <6.1) and eslint-plugin-react-hooks on 7.0 (xpo-09fa6f). `bun.lock` is regenerated and the unused `web/package-lock.json` is removed. Go: glamour 1.0, MCP go-sdk 1.8, golang.org/x/* and the rest to latest. **Building from source now requires Go 1.26** (required by golang.org/x/*). The README and Dockerfile are updated to match. (xpo-b2ca41)
- Agent chat: a turn shows the agent's work live. Narration is a regular message. The actions between messages collapse into one line: while the agent works, the current action's icon and target with a shimmer and the timer on the right; afterwards "6 actions · 2 failed". Click a line to see its actions on a timeline. Each action shows an icon in place of its verb (tooltip and screen-reader label keep the verb). A Lightbulb line shows while the agent is thinking. Failures no longer expand the list, and a failed action has only a red icon. The turn's total time moves under its last message, replacing the "Worked for …" header. lucide-react is upgraded to 1.57. (xpo-855e28)
- Agent chat: the chat popup, the composer, approval cards, message bubbles and the menus they open draw on their own monochrome surface scale (`--color-*-ai-*` tokens), a step above the app's, so the chat stands out from the board. The Agent view keeps the app canvas. A `.surface-ai` class puts shared menus and popovers on that scale. (xpo-03fa37)
- Agent chat popup: it opens centered above its status-bar pill, or above the Agent button for a new chat, instead of the bottom-right corner. It is resizable from its top edge, sides and top corners, from 360×400 up to 600×900, and remembers its size. (xpo-b45049)
- Issue rows: one pill style for row indicators, after Linear. Sub-issue progress, branch, chats and attachments sit in the same faint, fully rounded pill, with a muted 12px icon. The branch pill shows its commit count and the chat pill its chat count; details are in the tooltips. Pills and label chips have the surface background, so they stay dark on a hovered or highlighted row. Applies to backlog rows, board cards and My Issues. (xpo-219de9)
- Agent chat polish: mode dividers read "Entered Work Mode on xpo-…" and "Entered Chat Mode". The Work Mode segment shows the hammer and the issue ID, without "Work ·". The send and stop buttons are 24px. (xpo-219de9)

### Fixed

- Agent chat: the transcript lines up with the reply box again once the thread scrolls. The scrollbar no longer narrows only the right side and pulls the transcript off-center. (xpo-260a28)

- Agent chat: a background process the agent started (a dev server, a monitor) no longer keeps the turn running and blocks replies until Stop. Each chat now keeps one agent process (Claude Code or Codex) across turns, and a turn ends when the agent replies.
  - **Background tasks:** the composer shows "Monitoring N tasks", or "N agents working" for subagents. A card on the title lists the tasks; its Stop button ends the agent with its tasks. The next message resumes the session. `POST /api/local/chats/{id}/background/stop`; `GET /chats/{id}` includes `background`; live `background_tasks` events.
  - **Agent turns:** when a background task ends and Claude Code replies on its own, the reply shows as its own turn (`agent_turn` event), marked with what woke the agent.
  - **Stop** interrupts the turn only; background tasks keep running. Model and effort changes apply in place, without restarting the agent.
  - **Lifecycle:** a mode change restarts the agent ("Entered Chat Mode. Background tasks stopped." when tasks ran). Agents idle for 30 minutes without tasks, deleted chats and board shutdown end their agents. A board that starts stops agents a crashed board left running.

  (xpo-fb4f9b)
- Markdown: rendered checklist items now wrap like the editor. Wrapped lines line up with the first line of text instead of the checkbox, and the checkbox sits at the editor's height. (xpo-d4e6e4)
- Issue activity: status, estimate and priority chips now line up with the row's text. The row and its chips use flex centering instead of baseline-aligned inline text. (xpo-d3578a)
- Dashboard: Active Epics rows no longer nest the copyable issue ID button inside the row button, which was invalid HTML and logged a React error on every Dashboard load. Clicking the row still opens the epic; clicking the ID copies it. (xpo-66c6a4)
- Agent chat: `xpo spec`, `xpo walkthrough` and `xpo artifact` calls always read "Read the … of" and showed no issue, also for writes. Tool details for MCP tools now include the call's `operation` and `issue_id` ("write xpo-1"), for Claude, Codex and approval cards, and the chat words them "Wrote the spec of xpo-1". (xpo-855e28)
- Agent chat: Work Mode no longer asks for approval every time the agent edits its issue's spec or walkthrough. Claude may now edit files in the bound issue's `.xpo/artifacts/<issue>/` folder without a card. Other issues' artifacts and everything else outside the worktree still ask. (xpo-2c724e)

- Issue activity timeline: every field change is described, cleared values read as words, and rows read as muted sentences. (xpo-ca763f)
  - **New descriptions:** priority ("set priority to High" / "cleared the priority"), parent ("set parent to [issue]" / "removed the parent issue") and cycle ("added this issue to Cycle 12" / "removed this issue from its cycle").
  - **Cleared values:** a cleared estimate reads "cleared the estimate" instead of "set estimate to 0 Points". Removing the assignee reads "removed the assignee"; before, such an event had no text. Removing all labels reads "removed all labels" on the web and in the CLI.
  - **Same wording in the CLI:** `xpo history` uses the same phrases, replacing "unassigned", "removed parent", "removed from cycle" and "added to cycle X".
  - **Muted rows:** actor names and values stay muted and turn white with a pointer cursor on hover. Comment headers keep the white name.
  - **Spacing and alignment:** each row is one run of text with normal word spacing instead of 8px gaps, and long rows wrap like a sentence. Chip labels sit on the text baseline, and the estimate triangle no longer hangs below it.

- MCP `update` and `xpo update --json` clear `assignee`, `cycle_id`, `priority` and `story_points` when the field is `null`, like `parent`. They used to fail with "no fields set", or leave the field unchanged when another field was set. `null` on `description`, `labels` and `links` still means "unchanged"; clear those with `""` or `[]`. Each nullable field's schema description says how to clear it, and the CLI timeline describes cleared values in words instead of "assigned to " or "set priority to 0". (xpo-bbfcf9)

- MCP `update` and `xpo update --json` can clear an issue's parent. `parent: null` now clears it instead of failing with "no fields set". A literal quoted empty string (`"\"\""`) is treated as `""` instead of failing with `issue "" not found`. The `parent` schema description explains how to clear it. (xpo-328824)

- Agent chat: after you allow the agent's `start`, the chat continues in Work Mode on its own. You no longer have to send "go" to get it working. The continued turn shows no user message, just the "Entered Work Mode on …" divider; a new `turn_continued` chat event opens it. (xpo-ad6069)

- Merging an issue while its merge view is open (for example from the agent chat) no longer shows a raw `{"error":"no branch found for this issue"}` screen. The view switches to the Merged view in place. An issue without a branch shows a clear "No branch found" screen, and API errors across the UI show their message instead of raw JSON. (xpo-64be51)

- The attachment pill on backlog rows and board cards shows its count again, like the branch, chat and sub-issue pills. (xpo-199a69)
- Agent chat: an open chat keeps following new output while you're at the bottom, also when a card or code block grows after it renders, the composer grows, or the popup is resized. Only scrolling up stops following; scrolling back to the end resumes it. (xpo-63b755)
- Agent chat: the agent can now load skills, in Chat Mode and Work Mode, without an approval card. Its prompt tells it to load the `xpo` skill before planning or changing anything, so Work Mode follows the project's workflow: spec, handoff, and a stop for review. A skill load shows as "Loaded the skill xpo". (xpo-a058b4)
- Agent chat: maximizing a chat popup into the Agent view now removes its pill from the status bar. A dock button at the right of the Agent view's header puts the chat back in the status bar as a popup and returns to the view you were on before. (xpo-b204f1)
- Agent chat: the composer's issue chip is now the context the next message carries. Opening an issue adds it as a chip. Sending a message carries it and clears the chip, so later turns carry no issue context. A removed chip stays removed after closing and reopening the popup, until you open an issue again. A Work Mode chat never gets its bound issue as a chip. (xpo-c94d85)
- Review Changes now shows when a worktree's only changes are new, untracked files. Branch stats count them as uncommitted (files and line insertions; gitignored files and `.xpo/` excluded). The dialog opens on "Uncommitted" when the branch has no commits yet, instead of an empty "All changes". When Review Changes shows, it is the primary, top button, and Start Work or Continue Work turns secondary. (xpo-926f0b)
- Agent chat: opening an existing chat (Recent chats, Start Work or Continue Work, an issue's chat pill or chats list) now adds its status-bar pill, so minimizing it no longer loses the way back. (xpo-b45049)
- `XPO_*` environment variables now set every config key, as the README promised. Before, a key without a default, such as `XPO_REMOTE_URL`, was ignored unless a config file already set it. List keys take a comma-separated value. Map keys (`labels`, `permissions`) can still only be set in a config file. (xpo-e81612)
- Plain issue IDs in descriptions, comments and chat answers were never linked, because the API sends the prefix without the hyphen. (xpo-b83f1c)
- Agent chat tool calls:
  - **Arguments:** Grep patterns and other tool arguments with escaped quotes are no longer cut off (`\\{\\` instead of the pattern). They are decoded properly, which also fixes drive's spinner detail.
  - **Readable summaries:** the work list shows tool calls in words, such as "Read board.go", "Searched the code for …", "Listed issues matching …", and "Created issue …".
  - **Titles:** xpo tools show the issue ID or title instead of the description.
  - **Alignment:** bullets line up in the narrow popup.

  (xpo-6afc58)
- Menu items with a suffix (relative times, filter counts, "⌘⌫") no longer overflow their row (xpo-c047ce)

## [1.4.0] — 2026-10-12

### Added

- README section "Upgrading from 1.3.x": the one-link-per-pair rule for `link`, and refreshing agent instructions with `xpo init` / `xpo doctor --fix` (xpo-4659c8)
- Board cards: the assignee avatar is now an assignee picker button. Unassigned cards show a muted dashed placeholder avatar. Press `a` on a focused card to open the picker (xpo-5afd82)
- Backlog rows: the assignee avatar is now an assignee picker button, like the status, priority and estimate cells. Unassigned rows show a muted dashed placeholder avatar instead of nothing, so the column stays aligned. Press `a` on a focused row to open the picker (xpo-dcd4f6)
- The issue Activity section reads as a timeline: a vertical rail runs through the avatar column and joins consecutive events and comment cards. When an event row wraps, its continuation lines now indent under the name instead of running under the avatar. Comment bodies (new `.prose-activity` class) and the comment box are now 14px instead of 15px (xpo-a1470f)
- `overwrite` for the MCP `link` tool (`--overwrite` / `"overwrite": true` in the CLI): replaces an existing link of a different kind between two issues, whichever side stores it; the output lists what was replaced (`replaced`, and e.g. `Linked xpo-a relates_to xpo-b (replaced blocks)`) (xpo-c93b11)

### Changed

- Two issues can have at most one link between them. `link` on an already-linked pair fails unless `overwrite` is set, and `add`/`update` reject a `links` set with two entries for the same issue (create previously had no duplicate check). Pairs recorded before this rule keep working until a write changes them. `unlink` now takes its own `UnlinkToolInput` (xpo-c93b11)

- Issue sidebar relations overhaul:
  - **Relationships button:** the Parent property becomes a flag-icon Relationships button ("No relations" / "N Relationships", where the parent counts as one). It opens a popover menu that replaces the Add relation modal:
    - Remove parent comes first when the issue has a parent.
    - Then Parent of…, Sub-issue of…, Related to…, Blocked by…, Blocking… and Duplicate of….
    - Then More ▸, holding Depends on…, Dependency of… and Duplicated by….
    - Picking a kind leads to an issue search with one-line titles.
  - **Relations card:** relations are grouped into per-kind sections worded from the viewed issue ("Blocked by" / "Blocking"). Each linked issue is a chip that shows `×` on hover and a 250 ms preview card (ID, assignee, title, status, priority). Right-click offers Change to ▸ (change a link's kind in place), Remove relation and Copy link. Clicking a chip opens the issue, and a `Waypoints` button on the card opens the dependency graph (xpo-903ca3)

- One attribution rule across the web UI: when an agent acted on behalf of a person, their name renders with a dashed underline and reveals the agent on hover/focus (and to screen readers); direct actions render plain. Applies to the issue Activity timeline (events and comments), Merge view comments, Dashboard activity feed, Timeline, Inbox change summary and the sidebar assignee via a shared `ActorName` component. Replaces the spelled-out "Claude Code on behalf of Name" comment headers, and the Inbox now shows the principal instead of the raw agent (xpo-a9560c, xpo-6e0026)

### Fixed

- Board popovers opened with `s`, `e` and `l` (and the new assignee picker) now anchor to the focused card instead of the board's top-left corner (xpo-bfca2d)
- Board keyboard navigation no longer focuses cards in hidden or collapsed columns. With Backlog hidden, `j` now focuses a visible card and shows the focus ring (xpo-93efdd)
- Issue detail property popovers are now accessible:
  - **Shared pickers:** status, estimate, priority and assignee use the shared `Menu`-based pickers instead of hand-rolled lists. `Menu` owns their keyboard navigation, so number shortcuts also work for estimate and priority.
  - **ARIA:** every sidebar property trigger exposes `aria-haspopup="menu"` and `aria-expanded`, and each picker menu has an `aria-label`. Single-choice pickers (status, priority, estimate, cycle, assignee) announce their options as `menuitemradio` via a new `MenuItem` `radio` prop. Multi-select menus keep `menuitemcheckbox`.
  - **Assignee:** a new shared `AssigneePicker` is used by the sidebar and the context menu. "Remove assignee" becomes a "No assignee" option with a dashed placeholder avatar, checked when nobody is assigned.
  - **Look:** picker menus have an even 4px inset, and checked options show in primary text (xpo-c8eaf4)

- The issue detail view shows a single scrollbar on devices with classic scrollbars, instead of three. The page no longer scrolls: `Layout` and `ViewContainer`'s scroller are `relative`, so `ActorName`'s `sr-only` text can't extend the document. The property sidebar is capped to the pane's height (`100cqh`) instead of the viewport. It hides its scrollbar and fades its bottom edge while more content is below. The global scrollbar styles moved into `@layer base` so utilities can override them (xpo-fe51ab)
- Status icons in the issue sidebar's Relations card no longer shrink next to long titles; `StatusIcon` now always applies `shrink-0`, which fixes every flex-row call site (xpo-334821)
- MCP `merge` records the MERGE event and DONE transition as the agent acting on behalf of the user (`created_by` agent, `on_behalf_of` principal) instead of as the configured user; `MergeIssue` and `ResolveReviewIssue` now sync the client identity to the transport (xpo-40ab84)
- The issue-detail artifact list shows the principal with the dashed-underline agent attribution instead of the raw agent identity; artifacts now carry `on_behalf_of` from their latest write in the HTTP API and CLI/MCP JSON output (xpo-61ff1e)
- The Dependencies view shows each `relates_to` link once (table row and graph edge) instead of twice; its derived inverse is now deduplicated as an undirected pair. A `relates_to` between two issues that also share a blocking/duplicate link is hidden as redundant, and cycle detection only considers `blocked_by`/`depends_on` edges, so relates_to no longer produces false cycle warnings (xpo-2fc583)
- The Dependencies table search trims its query: leading and trailing whitespace no longer cause missed matches, and a whitespace-only query shows all rows like an empty search (xpo-3d9d5a)
- `xpo demo` no longer assigns issues to agents with no person behind them: agent work is assigned to a human (Claude for Sarah, Devin for Jake, Whiskers for Luna) with the agent as `assignee_via`, agent events carry `on_behalf_of`, and agents are left out of the demo contributors (xpo-53daa5)
- The Dependencies graph dims nodes and edges outside the focus/hover set less (nodes 0.55, edges 0.4, DONE 0.7), so dimmed labels and ids stay readable; opacity now comes from `dep-dimmed`/`dep-done` CSS classes backed by `--dep-*-opacity` variables in `index.css` instead of inline styles (xpo-aab636)
- `xpo link`, `xpo link --json` and the MCP `link` tool share one `Client.Link` helper and return identical "already exists" and self-link errors (xpo-f94753, fixed by xpo-c93b11)

## [1.3.0] — 2026-10-09

### Added

- README "Upgrading from 1.2.x" section: branch-mode migration, re-running `xpo init`, `xpo board` network binding and access token, JSON output changes, principal-first assignees. The Board section now documents the `--host` access token (xpo-e60c26)
- `xpo board --host <non-loopback>` requires a per-run access token. It is printed on startup as `…/?token=…` and exchanged for an `HttpOnly`, `SameSite=Lax` session cookie named `exponential-<sha256(token)[:8]>`. Without the cookie, `/api/*` returns 401 and pages redirect to a new `/auth` form; the web UI redirects there on any 401. `/healthz` stays public, and loopback binds are unchanged (xpo-897f35)
- Web UI bundles Inter Variable and JetBrains Mono Variable (latin + latin-ext woff2, embedded in the binary) instead of relying on locally installed fonts; `@font-face` pins `ascent-override` / `descent-override` / `line-gap-override` so Linux and macOS lay out text identically. SIL OFL 1.1 license texts ship at `/licenses/` (xpo-bade50)
- `SearchInput` in the My Issues TopBar filters the active tab by title, ID or label; Backlog and My Issues share a new `matchesSearch` predicate, and Backlog search now ignores surrounding whitespace (xpo-5ed512)
- `SearchInput` in the Labels TopBar filters labels by name (case-insensitive substring), with a "No labels match" empty state (xpo-c6043f)
- `unlink` MCP tool and `xpo unlink <source> <target> -t <type>` CLI command (with `--json`): remove one relationship between two issues, whichever side stores it, including links to deleted issues (xpo-9d5d00)
- "Worked by" filter (Agent / Human) in the Backlog, My Issues and Inbox filter menus, backed by a new `assignee_via` field on issues in the CLI JSON, MCP and HTTP API (xpo-35fc16)
- "Open" button in the issue detail top bar that opens the issue's local worktree in VS Code via a `vscode://file/…` link; the path comes from a new local-only `GET /api/local/issues/{id}/worktree` endpoint, which `xpo board` answers itself even in remote/proxy mode (xpo-c4fbed)
- Unified JSON I/O layer: `internal/jsonio/` package as single source of truth for all JSON-serializable types shared between CLI and MCP (xpo-1a5504)
- `xpo list --json` outputs `jsonio.ListOutput` envelope (xpo-2e3060)
- `xpo show --json` outputs `jsonio.ShowOutput` envelope with children and archived status (xpo-b5d8f0)
- `xpo comments --json` outputs `jsonio.CommentsOutput` envelope (xpo-e43fff)
- `xpo blocked` without arguments lists all blocked issues; `--json` flag for JSON output (xpo-c3f7a0)
- `xpo inbox --json` outputs `jsonio.InboxOutput` envelope (xpo-d31d61)
- `SearchInput` component added to Timeline TopBar (xpo-23529e)
- `MenuFilter` component for filterable menus; `bare` prop on `Menu` for chrome-free mode inside existing containers (xpo-9a7907)
- Digit keys (0-9) registered in Menu keyboard handler for number-key shortcuts (xpo-9a7907)
- Composable `Menu`, `MenuItem`, `SubMenu`, `MenuDivider`, `MenuLabel` component system with ARIA roles, keyboard navigation via the shortcut registry, safe-triangle diagonal movement, submenu close delay, and sub-menu flyout positioning; ContextMenu and FilterMenu refactored to compose from these primitives (xpo-de8d06)
- `PopoverPanel` component for visual chrome in raw-content Popovers (xpo-de8d06)
- `pointInTriangle` utility for safe-triangle hover calculations (xpo-de8d06)
- `popover-utils` extended with `right-start` placement for sub-menu flyouts with left-flip and viewport clamping (xpo-de8d06)
- `menu-utils` pure functions for index navigation with wrapping/skip and shortcut matching (xpo-de8d06)
- `Text` component for consistent typography primitives (xpo-e0334f)
- Shared `Tabs<T>` component with `[`/`]` keyboard cycling via the shortcut registry; Backlog and My Issues migrated, removing duplicated tab markup (xpo-d47781)
- Control-priority shortcuts fold into the active view's group in the keyboard help overlay instead of creating separate sections (xpo-d47781)
- Keyboard system architecture design document (`design-docs/keyboard-system.md`) (xpo-d47781)
- Shared `SearchInput` component with `/` focus and `Esc` clear via the shortcut registry; Backlog and Dependencies migrated (xpo-ef6602)
- Shared `IconButton` component for toolbar actions with built-in tooltip and active-indicator dot; 6 instances across Backlog, Board, Inbox, MyIssues, and Timeline migrated (xpo-4235c6)
- `Popover` refactored to anchor-ref positioning with placement options, 4-edge viewport clamping, and built-in click-outside/Escape dismissal; 17 consumer instances migrated, `InlineDropdown` rewritten on Popover (xpo-5d1c38)
- `Button` component supports `forwardRef` for ref-based popover anchoring (xpo-5d1c38)
- Shared `FilterButton` component bundling IconButton, FilterMenu, `f` shortcut, and active dot; Backlog, MyIssues, and Inbox migrated (xpo-e16962)
- `xpo spec` CLI command for viewing and managing issue specs (xpo-d9d6f2)
- `xpo walkthrough` CLI command for viewing and managing issue walkthroughs (xpo-09a820)
- `--with-spec` and `--with-walkthrough` flags on `xpo show` command and MCP `show` tool to inline artifact content (xpo-68eb93)
- `xpo artifact --json` input/output with `jsonio` envelope (xpo-43d8c5)
- `xpo start --json` input/output with `jsonio` envelope (xpo-cdd060)
- `xpo merge --json` input/output with `jsonio` envelope (xpo-e19cfd)
- `xpo link --json` input/output with `jsonio` envelope (xpo-885f42)
- Standardized JSON error output across all CLI `--json` commands — consistent `{"error": ...}` envelope on validation and runtime failures (xpo-d1a06b)
- `Heading` component for consistent title and heading styling (xpo-250ddf)
- `CountBadge` component for issue count badges (xpo-f27a3e)
- `xpo doctor --fix`: auto-fixes local state silently, shows change plan for committed files, offers replace/keep/diff for edited managed blocks (xpo-76e2a1)
- `xpo doctor --strict`: warnings exit non-zero for CI (xpo-76e2a1)
- `xpo init` flags: `--yes`, `--prefix`, `--agents`, `--force` for non-interactive/CI use (xpo-76e2a1)
- Version stamping in managed block markers — integration version derived from blocks, not stored in config (xpo-76e2a1)
- Re-init intelligence: up-to-date one-liner, stale version diffs, downgrade protection, edited-block detection (xpo-76e2a1)
- `charmbracelet/huh` dependency for multi-select and select prompts (xpo-76e2a1)
- 53 new tests: managed blocks, health checks, doctor scenarios, version comparison (xpo-76e2a1)
- `xpo doctor` reports `issues.db` lines whose `created_at` does not strictly increase in file order (report-only) (xpo-e9a2eb)

### Changed

- `make release` is replaced by `make release-prep VERSION=x.y.z`, which bumps `CLIVersion` and dates the CHANGELOG section in the release issue's worktree without committing, and `make release-tag VERSION=x.y.z`, which tags on `main` after the merge after checking the branch, a clean tree and the version, and does not push. It uses `perl -pi` instead of GNU-only `sed -i`, so it works on macOS (xpo-e60c26)
- `xpo start` (CLI and MCP) is idempotent when run from inside the issue's existing worktree. It returns success with the same branch and worktree path, moves the issue to DOING if needed, and doesn't recreate the worktree or re-run `worktree_setup`. This lets an orchestrator start an issue and then let the agent inside the worktree call `start` again. `--force` still tears down and recreates the worktree (xpo-7325f7)
- Web UI writes now go through TanStack Query mutations (`api/mutations.ts`) instead of components calling the API and then `onRefresh()`. Status, estimate, priority, label, assignee and cycle edits, plus Board and Backlog drags, show up immediately and roll back with a "Couldn't save: <reason>" toast if the server rejects them. A status cascade to sub-issues refetches issues once instead of once per child. Creating an issue, starting work, deleting, commenting and label edits now toast on failure instead of failing silently. The `onRefresh`, `patchIssue` and `onConfigLabelsChange` props are gone (xpo-6e757d)
- Web UI per-view data (cycles, cycle progress, dashboard metrics and activity, timeline, instances, and issue history, artifacts, worktree, commits, diffs and mergeability) now loads through shared TanStack Query hooks: requests are deduplicated across views, and cycles, dashboard and timeline refresh on server events instead of going stale until remounted. MergeView spins its Refresh icon while refreshing (at least one full turn on a click), then briefly shows a check, or an alert plus a "Couldn't refresh: <reason>" toast; a failed uncommitted-diff poll shows "Couldn't refresh, showing changes as of HH:MM" beside it instead of silently showing outdated changes (xpo-ec0480)
- Web UI restructured from a single `App` component into `Layout` → `Sidebar` + `ViewRouter` → views. Views no longer receive props: they read issues, config, inbox and user from TanStack Query hooks, whose cache is refreshed by server events, and own their own tab, filter and sort state. Saved filters, sort and tabs keep their existing localStorage keys. Filter chips are rendered into the status bar by each view, the sidebar, routes, titles and `g` shortcuts all come from one `VIEWS` registry, and the Inbox list now uses `ViewContainer` (xpo-de3808)
- Issue detail prev/next (`j`/`k`) now walks the order of the view the issue was opened from (Issues, Board, My Issues or a cycle), instead of always the Issues list; deep links and other views use the default sorted order (xpo-de3808)
- Backlog rows and status headers no longer draw separator lines and have rounded backgrounds (xpo-c26f7b)
- All TopBar views except Inbox render through a shared `ViewContainer` shell (TopBar, optional header bars, and a full-width content area with the scrollbar at the view's right edge; width-limited views center a `max-w-7xl` or issue-detail column inside it). Scroll areas that change while visible (views, sidebars, MergeView panes and diffs, pickers, the command palette, scrolling menus) use a new `scroll-stable` utility: a forced, transparent track with a symmetric `both-edges` gutter, so content no longer shifts when a scrollbar appears in Chrome or Safari. The global `.overflow-y-auto { overflow-y: scroll }` override is gone, so `overflow-y-auto` behaves normally again (xpo-c26f7b)
- Links are bidirectional: `A blocks B` also means `B blocked_by A`, whichever issue stores it. `show` (MCP, CLI, API, web) lists derived inverse links with `derived: true`. `start` and `xpo drive` honour blockers stored on the other issue. `link` reports a relationship already stored from the other side as existing. `update { links }` treats the list as the issue's full relationship set, so a link left out is removed from whichever side stores it (xpo-9d6609)
- Assignees are principal-first: an issue is always assigned to a person, never an agent. Old assignments to agent identities (`Claude Code <agent@host.local>`) resolve to the person the agent worked for, at read time, with no event-log rewrite. Avatars and assignee names always show the person; agent-written comments read "Claude Code on behalf of <person>" (xpo-35fc16)
- `start` assigns the issue to the person the starter works for (the MCP/drive principal, or the CLI user); an issue held by someone else keeps its assignee unless `--force` (xpo-35fc16)
- `xpo drive` records its events on behalf of the git-config user (xpo-35fc16)
- CLI timeline and inbox lead with the person: "Nicolas (via Claude Code)" (xpo-35fc16)
- `xpo add --json` now uses `jsonio.AddInput` directly (dropping `inputs` shim), calls `ValidateCreatePayload` for full validation parity with MCP, and emits `jsonio.AddOutput` JSON instead of plain text (xpo-3847a5)
- `xpo update --json` now uses `jsonio.UpdateInput` directly, calls `ValidateUpdatePayload` for full validation parity with MCP, and emits `jsonio.UpdateOutput` JSON instead of plain text; `done`, `planned`, and `blocked <id>` gain `--json` output (xpo-199282)
- `xpo rationale --json` now outputs `jsonio.RationaleOutput` envelope instead of raw search result (xpo-11e559)
- **Breaking**: `xpo history --json` now outputs a single JSON envelope with rich timeline entries instead of JSONL; MCP `history` tool output upgraded to include payload, on_behalf_of, and source fields (xpo-f56ce4)
- `internal/inputs/` reduced to a backward-compat re-export shim over `internal/jsonio/`; will be removed once CLI commands are migrated (xpo-1a5504)
- `StatusPicker`, `PriorityPicker`, `EstimatePicker`, `LabelPicker` rewritten as Menu + MenuItem compositions, eliminating duplicated keyboard handling, focus management, and CheckIcon definitions (xpo-9a7907)
- `Popover` is now positioning-only; visual chrome moved to `PopoverPanel` or `Menu` (xpo-de8d06)
- `ContextMenu` decomposed from monolithic 470-line component into Menu/MenuItem/SubMenu composition (xpo-de8d06)
- `FilterMenu` rebuilt with Menu/SubMenu; positioning fixed with `useLayoutEffect` + `queueMicrotask` (xpo-de8d06, xpo-b5a70c)
- Backlog and Board view option menus refactored from inline JSX to Menu compositions (xpo-de8d06)
- Menu items use compact sizing (`py-1.5`) with three-tier visual hierarchy: secondary text, hover highlight, active highlight (xpo-de8d06)
- `xpo init` is now a unified wizard: prefix prompt → agent multi-select → terraform-style change plan → confirm → apply. Replaces the separate `init mcp` and `init skill` subcommands (xpo-76e2a1)
- Agent instruction files use managed blocks (`<!-- xpo:begin VERSION sha256:HASH -->`) for safe refresh — only the xpo-owned section is replaced, user content is never touched (xpo-76e2a1)
- Prefix stored without trailing dash (`pay` not `pay-`); separator added at point of use. Existing configs normalized on load (xpo-76e2a1)
- `xpo doctor` reorganized into four sections: xpo, Project, Integrations, This machine (xpo-76e2a1)
- Branded welcome message when running `xpo` or any command outside a project directory (xpo-76e2a1)
- Tagline defined once (`ui.Tagline`) and used consistently across all CLI output (xpo-76e2a1)
- `comments` command refactored from raw `storage.ReadEvents()` to `Client.GetIssue()` (xpo-e43fff)
- `xpo comment --json` refactored to use shared MCP `CommentToolInput` schema via `jsonio`, with full validation parity (xpo-aa4529)
- All CLI `--json` commands now resolve input IDs (short IDs, bare hashes) to canonical issue IDs before processing (xpo-db6d52)
- Pulse types moved from `internal/pulse` to `jsonio` package; velocity calculation no longer double-counts issues (xpo-b18d99)
- Cosmetic consistency pass for `--json` flags across all CLI commands — uniform flag descriptions and help text (xpo-66b8af)

### Security

- `xpo board` no longer listens on all network interfaces. It binds `127.0.0.1` by default (`--host` overrides it and logs a warning for non-loopback addresses) and rejects requests whose `Host` header isn't a loopback name or the `--host` value, blocking DNS rebinding. Previously anyone who could reach the port had unauthenticated read/write access, and in remote mode could use the user's bearer token through the proxy. The README documents `ssh -L` as the supported way to view a board on another machine. The Vite dev proxy targets `127.0.0.1` (xpo-809ad0)

### Fixed

- Event timestamps in `issues.db` strictly increase in file order again. `merge` stamped the DONE update about 100µs before its MERGE event, and cascading or link updates gave all their events one shared time, so `xpo doctor` reported out-of-order and tied timestamps after every merge. Storage now stamps each appended event, and each event in a rewritten web-UI tail, at least 1ns after the line before it, so this holds for every writer. `merge` and `update` also build their events in time order. This repo's 28 affected lines were re-timed once (xpo-e669d4)
- Agent-written comments in the merge view's Conversation tab show the principal's avatar and "Claude Code on behalf of <Principal>" instead of the raw agent identity; `xpo comments` and the TUI details pane show "<Principal> (via <Agent>)". Comments now carry `on_behalf_of` in the HTTP API, remote transport, CLI `--json` and MCP `show` output (xpo-a103c3)
- Dropping an issue onto a Backlog row with no status group no longer sends `status: null` (xpo-6e757d)
- MergeView no longer re-renders the whole diff on every 3s uncommitted-diff poll or Refresh. Unchanged polls and refreshes no longer block the page (before: ~190–200ms per poll on a 1,000-line diff), and a change to one file re-renders only that file's card (xpo-5839aa)
- MergeView's All changes / Uncommitted toggle no longer shifts sideways by a couple of pixels when switching, because each label reserves its bold width (xpo-e64b2b)
- Switching Backlog tabs now loads that tab's saved filters instead of carrying over the previous tab's filters and overwriting the new tab's saved ones (xpo-0c0556)
- Clicking a label on the Labels page no longer briefly sets an invalid `#/backlog` URL, and the applied label filter is saved like a manually chosen one (xpo-787e7b)
- Issue activity timeline no longer logs a React missing-key warning for updates that change several fields (xpo-1657f1)
- MergeView diff panes no longer jump sideways when expanding or collapsing diffs; scrolling menus (Assignee, Cycle, filter submenus) now actually get the stable gutter their undefined `scrollbar-gutter-both` class intended (xpo-c26f7b)
- Backlog right-click context menu no longer stops working after the list was empty (at load, or after a search with no matches was cleared) (xpo-164038)
- `merge` with `strategy: ff` no longer adds a separate `xpo: merge <id>` commit: the issue database and artifacts are amended into the fast-forwarded tip, and an explicit `commit_message` rewords it. The `merge` tool, the mcp-tools reference and skill step 9 now tell agents to omit `strategy` unless the user asks for one. Re-run `xpo init` in other projects to refresh installed skill copies (xpo-c16e28)
- Issue detail sidebar Cycle popover now uses the shared `CyclePicker` (also used by the context menu), gaining a filter input, `Cycle N (status)` labels and keyboard navigation (xpo-f16b80)
- Context Menu Cycle submenu now has a filter input and uses the shared `Menu`/`MenuItem` styling and keyboard navigation; cycles are labelled `Cycle N (status)` so the checkmark only marks the selected cycle (xpo-70e66a)
- Context Menu Assignee submenu now autofocuses its filter input and uses the shared `Menu`/`MenuItem` styling and keyboard navigation, matching Status, Priority and Labels (xpo-690930)
- Every `xpo` command stalled ~5 s (and printed a stray `ESC ]11;?`) under a pseudo-terminal that doesn't answer terminal queries: bubbletea v1's package init queried the background colour. Migrated the interactive prompts to `charm.land/huh/v2`, which removes bubbletea v1 (xpo-d053b5)
- `xpo init` and `xpo doctor` now detect CLAUDE.md/AGENTS.md instruction blocks that differ from the current template even when the version is unchanged; `init` refreshes them and `doctor` reports "Instructions out of date" (fixed by `--fix`) (xpo-757732)
- Backlog: ghost parents (faded parent rows shown above sub-issues in another status group) can be collapsed and expanded by click, ←/→ and Expand/Collapse all; the collapsed state is shared with the real parent row. While a search or filter is active, saved collapsed state is ignored so matches are never hidden, and collapses made during the search are temporary (xpo-0cbecc)
- Issue detail Activity avatars show the actor's gravatar or initials instead of the dashed placeholder; avatar and name share an "on behalf of …" tooltip when an agent acted for someone (xpo-53ed0e)
- Backlog: parent issues whose sub-issues are all hidden from the current view no longer show a non-clickable collapse chevron; the slot is left blank, like issues without sub-issues (xpo-e7c400)
- Agents no longer call `merge` on an uncommitted worktree. Step 9 of the generated xpo skill now requires a commit (it is a gate and an explicit step, with an optional changelog entry first), and the self-contradicting "do not commit" sentence is gone. The `merge` tool description says a committed worktree is required. Generated skill files end with an `xpo:skill` version/hash marker: `xpo doctor` reports "Skill out of date" when a skill differs from the current template, even at the same CLI version, and reports local edits separately. `xpo init` / `doctor --fix` refresh out-of-date skills and offer replace/keep/diff for edited ones instead of silently overwriting them (xpo-e88f1b)
- The instance registry no longer treats every process as dead on Windows, which emptied the web UI instance list and pruned live peers on `Register`. Liveness is now checked per platform: `Signal(0)` on Unix (with `EPERM` counted as alive) and `OpenProcess` + `GetExitCodeProcess` on Windows. `make lint` also runs `GOOS=windows go vet ./...` (xpo-60dc3c)
- `update { links }` now stores dependencies with `source_id` set to the issue's ID and rejects self-links and duplicate (target, kind) pairs; `links: []` clears all links instead of failing with "no fields set". Existing rows with an empty `source_id` are attributed to their owning issue on read, so `link` behaves correctly for them. The `link` tool, the `add`/`update` `links` schemas and the CLI `link` help now state that only `blocked_by` gates `start` (xpo-080fc4)
- Concurrent writers (several MCP servers, the CLI, the web server) no longer lose or corrupt events in `issues.db`: append, collapse and archive all hold an exclusive `.xpo/events.lock`, and each event line is written in a single call. `xpo archive` re-reads the log under the lock, keeps events written since the preview, and skips issues reopened since then, reporting them. Readers ignore a torn final line from an in-progress append (xpo-d6288f)
- `xpo merge` no longer fails with a phantom "resolve the conflict" error when xpo runs from inside the issue's worktree: every merge git call now targets the hub checkout. MERGE/DONE events are recorded only after the code commit succeeds, so a failed merge (hook, conflict, nothing to commit) leaves the issue in its old status with nothing staged and never rewrites `issues.db`. Errors carry git's real output, and the conflict advice appears only for real conflicts (xpo-9fef67)
- Web UI edit collapsing no longer merges edits across actors or past a newer event on the same issue, moves the merged event to the end so timestamps follow file order, keeps edited comments at their original time, and prunes no-op edits against running state; this repo's `issues.db` timestamps were repaired once so time-ordered and file-ordered replay agree (xpo-e9a2eb)
- Popover visibility in production builds: anchor ref not attached when `useLayoutEffect` fires; deferred via `queueMicrotask` (xpo-b5a70c)
- FilterMenu arrow-left key not registered for submenu entry (xpo-e16962)
- FilterMenu submenu options don't scroll into view on keyboard navigation (xpo-e16962)
- TopBar center slot now stays truly centered regardless of left/right content width changes (xpo-c078b7)
- MergeView "Files changed" badge now reflects the active scope — shows uncommitted file count when viewing uncommitted changes instead of always showing 0 (xpo-54a423)
- "All Changes" view falls back to uncommitted diffs when there are no commits yet, instead of showing an empty pane (xpo-54a423)
- Scope selector ("All changes" / "Uncommitted") promoted from file-tree sidebar to the tab bar for better visibility (xpo-54a423)
- Issues.db validation in `xpo doctor` with line-number reporting for invalid entries (xpo-76e2a1)
- `comment --json` now accepts MCP `CommentToolInput` payload format (xpo-eb0d61)
- `worktree_setup` hook output captured instead of leaking to stdout (xpo-7abb38)

### Removed

- **BREAKING:** branch mode is removed; worktrees are the only way to work on an issue (xpo-863802). `xpo start` always creates (or reuses) a worktree under `.xpo/worktrees/`, and `xpo merge` always merges from the hub checkout and never runs `git checkout`. Removed:
  - `xpo start --mode` and `xpo merge --no-wt`, which now fail with "unknown flag".
  - The `mode` field of the MCP `start` tool and `xpo start --json`. Passing it is now rejected as an unknown field.
  - The `worktrees` config key. `worktrees: false` prints a warning and is ignored; `worktrees: true` is a silent no-op, and `xpo init` no longer writes it.
  - The squash/ff `issues.db` union-merge that only branch mode needed.

  **Migrating:** before upgrading, merge or park any issue you started in branch mode, and check out the default branch in the primary checkout. `xpo merge` refuses to run while the hub is on another branch. `xpo start` refuses an issue whose branch is checked out in the hub, and tells you to switch the hub back first. Branches without a worktree still merge normally from the hub.
- Unused `PendingEventsPanel` component (xpo-c26f7b)
- "Agent Identity" instructions telling agents to set themselves as assignee, from CLAUDE.md/AGENTS.md/GEMINI.md, the xpo skill and the generated templates (xpo-35fc16)
- `xpo init mcp` and `xpo init skill` subcommands — absorbed into unified `xpo init` (xpo-76e2a1)
- `internal/inputs` backward-compat re-export shim — all CLI commands now use `jsonio` directly (xpo-aebfc5)

## [1.2.1] — 2026-09-10

### Fixed

- Tests no longer depend on the host machine's `init.defaultBranch` git config — test packages isolate from global/system git config via `TestMain` (xpo-3b2ec1)
- Mergeability check no longer falsely blocks on uncommitted working directory changes — now uses `git merge-tree` to compare committed refs only (xpo-fad57f)
- Merge view disables the merge button and shows "No commits to merge" when there are zero commits ahead

### Added

- MergeView Files tab scope toggle: switch between "All changes" (branch vs main) and "Uncommitted" (working tree vs HEAD) with 3-second auto-refresh polling; includes synthetic diffs for untracked files (xpo-6211f5)
- Amber file icon highlighting in MergeView file tree for uncommitted/untracked worktree files; mergeability API now returns `dirty_files` array (xpo-110b09)
- Multi-harness init: `xpo init` is now config-only; new `xpo init mcp` writes per-harness MCP server config, `xpo init skill` installs workflow skills and agent instruction files — each concern is independently runnable (xpo-ceb434)
- Agent harness support for Claude Code, GitHub Copilot, Cursor, OpenAI Codex, and OpenCode — per-harness MCP config formats (JSON with `mcpServers`/`mcp` keys, TOML, and OpenCode's `local-array` style) and skill directories (xpo-ceb434)
- Global skill installation (`xpo init skill --global`): canonical location at `~/.config/xpo/skills/` with symlinks into each harness's skill directory; macOS/Linux only (xpo-ceb434)
- `xpo init skill` interactive flow: detects installed agents, prompts for global/local scope, respects existing agent instruction files (append without `--force`, replace xpo section with `--force`) (xpo-ceb434)
- `xpo doctor` reorganized into three sections — Project Configuration, MCP Configuration, Agent Integration — each with its fix command (xpo-ceb434)
- `xpo init` prints actionable next-step hints when MCP or skills are not yet configured (xpo-ceb434)
- 20 new scenario tests covering init/mcp/skill clean, idempotent, force, and cross-format workflows (xpo-ceb434)

### Changed

- Skill upgraded to v3.1: restructured workflow steps (rationale tool, explicit gates, handoff self-review, issue labels with primary/secondary system, backlog review section, CANCELED/DUPLICATE statuses) (xpo-dd97a5)
- Skill directory renamed from `xpo-workflow` to `xpo` to match reference skill (xpo-dd97a5)
- Colored CLI output: green ✓ success, yellow ! warnings, red ✗ errors via lipgloss (xpo-dd97a5)
- `xpo init` prompts for issue ID prefix interactively, shows quick health check with MCP/skill status, single call-to-action for `xpo doctor` (xpo-dd97a5)
- `xpo doctor` summary line ("7 passed, 4 warnings"), one fix hint per section, tighter formatting (xpo-dd97a5)
- Root command description: "The git-native engineering system for human-AI teams" (xpo-dd97a5)
- Agent registry trimmed from 10 to 6 harnesses: Generic Agent, Claude Code, GitHub Copilot, Cursor, Codex, OpenCode — removed Windsurf, Gemini, Cline, Roo Code, Aider, Continue (xpo-ceb434)
- `PersistentPreRunE` allowlist uses parent-chain check so `init` subcommands work without config (xpo-ceb434)

## [1.2.0] — 2026-09-09

### Added

- Expanded vitest coverage to 255 tests across 12 test files — covers labels, sort, constants, filters, dependency graph, timeline, inbox, command palette, and md5 utilities; extracted and deduplicated business logic from 10+ component files into shared utility modules (xpo-653324)
- My Issues view: right-click context menu with full issue actions (status, priority, assignee, labels, estimate) (xpo-653324)
- Labels view: click a label to navigate to filtered Backlog (xpo-653324)
- Command palette: fzf-style scored fuzzy search — substring and exact matches rank highest, scattered character matches rank lowest (xpo-653324)

### Changed

- Dependency graph: replaced dagre with elkjs — proper orthogonal edge routing, native edge label placement, edges connect at node borders; canonical direction is now upstream (arrows point toward prerequisites); dim non-focus nodes/edges; completed toggle properly hides terminal issues; default zoom 125% (xpo-653324)
- Board view: custom pointer-aware collision detector for reliable cross-column drag-and-drop; Backlog/Canceled/Duplicate columns hidden by default (xpo-653324)
- Assignee pickers: filter out identities without valid email addresses (xpo-653324)

### Fixed

- Backend: moving a child issue to BLOCKED no longer auto-starts the parent (first-start trigger only fires on DOING) (xpo-653324)
- Backend: removing all dependencies/labels from an issue now persists correctly (omitempty fix on UpdatePayload slices) (xpo-653324)
- `xpo init`: `worktrees: true` now written to config.yaml by default (xpo-653324)
- Dependency add-relation modal: stable height when switching between kind selection and issue search (xpo-653324)
- Board: empty columns now have a tall enough drop target for drag-and-drop (xpo-653324)

- Frontend test infrastructure with vitest — 72 tests covering diff utilities (`parseDiffByFile`, `addLineNumbers`, `buildFileTree`, `flattenSingleChildDirs`, `buildSplitLines`) and format utilities; `make test` now runs vitest alongside Go tests (xpo-46c418)
- Keyboard help overlay: added Board section documenting all Board shortcuts; disambiguated key separators (`/` = or, `→` = then, `+` = together); expanded status number shortcuts to show actual status names; widened to 4-column layout (xpo-87503f)
- Optional `default_branch` field in `.xpo/config.yaml` — explicitly sets the base branch for branching, merging, and diff operations, bypassing git-based detection; `xpo init` auto-detects and writes it (xpo-961910)
- Global config accessor (`config.Get()` / `config.Set()`) — package-level functions can now read config without parameter threading (xpo-961910)
- Shared `TopBar` component (`components/ui/TopBar.tsx`) with left/center/right slots — replaces 13 duplicated top-bar implementations with a single composable primitive; includes `cn()` utility (`clsx` + `tailwind-merge`) for safe Tailwind class overrides (xpo-09ebcc)
- Backlog and Dependencies search inputs show `/` keyboard hint when empty and `Esc` when populated (xpo-09ebcc)
- Cycles view now has a proper fixed top bar matching all other views (xpo-09ebcc)
- `xpo start --mode worktree|branch` CLI flag and MCP `mode` parameter — override the global worktree config per-invocation; replaces the old `--no-wt` flag (xpo-00198b)
- Dependencies view: interactive dependency graph — filterable table as default with kind/search/completed filters, click-to-drill-down graph powered by dagre for exploring dependency chains; relationship-aware resolved logic; entry from issue detail property sidebar (xpo-a49ec2)
- `xpo rationale` CLI command and MCP tool — BM25-ranked full-text search across specs and walkthroughs for design rationale; supports `--top N` / `-n N` and `--json`; rich terminal output with colored scores, document types, labels, and clickable artifact paths (xpo-d6b4de)
- `xpo history` now renders a visual activity timeline with typed dot markers (`●` events, `○` commits, `◆` merges, `☰` artifacts) connected by a vertical rail; interleaves git commits with issue events; supports `--json`, `--reverse`, `--since <duration>`, `--limit N`, and auto-paging through `$PAGER` (xpo-d82dc7)
- `xpo history` without arguments shows a project-wide timeline across all issues and recent commits (xpo-d82dc7)
- Git hooks (`.githooks/`) — commit-msg strips AI-agent trailers and enforces prefix convention; pre-commit runs `go vet` and `eslint` on staged files; pre-push runs the test suite. Activated automatically via `make setup`
- `xpo pulse` CLI command — compact terminal dashboard of project health metrics (velocity, cycle time, lead time, WIP, blockers, throughput) with color-coded values; supports `--json` for machine-readable output (xpo-9ba68b)
- `xpo init` now writes a `name:` field to `config.yaml` derived from the folder name, used as the project heading in `xpo pulse` and `xpo board` (xpo-9ba68b)
- Empty state messaging for Backlog, Active, and Done filtered views — shows tab-specific or filter-aware hints instead of a blank screen (xpo-5a17a7)
- Timeline event-type filter menu — replaces single-select tabs with a multi-select checklist popover (Issues, Closed, Comments, Merges, Artifacts, Commits); client-side filtering for instant toggling; renders ARTIFACT events and CANCELED/DUPLICATE status transitions that were previously invisible (xpo-eb76cd)
- Walkthrough tab in the merge view — renders the issue's walkthrough as Markdown and becomes the default tab when present (xpo-3bfbc8)
- Backlog view: `[` / `]` keyboard shortcuts to cycle through sub-tabs (All Issues, Backlog, Active, Done) with wrapping (xpo-e5d2e4)

### Changed

- Parents with no visible children in nested backlog mode show a static right-pointing chevron instead of a clickable expand toggle; ghost parents show a static down-pointing chevron — clicking neither triggers cross-group side effects (xpo-2dd9a8)
- Ghost row opacity increased from 50% to 65% for better readability (xpo-2dd9a8)
- All top bars standardized to `h-12` height with `pl-5 pr-3` asymmetric padding (xpo-09ebcc)
- Backlog search is now always visible in the center of the top bar instead of appearing on `/` keypress (xpo-09ebcc)
- Labels view top bar normalized from `h-13 px-6` to standard TopBar defaults (xpo-09ebcc)
- Timeline kind filter converted from segmented control to underlined tabs on the left (xpo-09ebcc)
- Filter buttons in MyIssues and Notifications changed from text+icon to icon-only with tooltip, matching Backlog/Board style (xpo-09ebcc)
- Backlog filter and view-options icon buttons normalized to `w-7 h-7` bordered style, matching Board (xpo-09ebcc)
- Dependencies search Escape key now clears the search and blurs the input (xpo-09ebcc)
- CLI accent color (issue IDs, labels) lightened from `#6f42c1` to `#b392f0` for dark terminal readability (xpo-d82dc7)
- Branch badges on board/backlog cards now show commit count instead of SHA, with a blue indicator dot for uncommitted changes (xpo-ebb248)
- Detail sidebar shows "No commits" and uncommitted change details instead of the base branch SHA when a branch has no commits (xpo-ebb248)
- SubProgress ring uses Lucide-compatible viewBox for proper text alignment, with a dashed track and brand-color progress arc (xpo-ebb248)
- Artifact count badges unified with branch badge style (rounded-square, bordered) across board and backlog views (xpo-ebb248)

### Fixed

- Markdown checklist checkboxes now visible and flush-left in both rendered markdown and tiptap editor — border uses `--color-border-control` for proper contrast, padding zeroed to align with surrounding list content (xpo-958474)
- Spec and walkthrough content in issue detail panel now live-updates when the file changes on disk — artifact content cache is invalidated on `updated_at` change instead of only on issue navigation (xpo-0b193b)
- `xpo merge` auto-checkout failed in branch mode when `Worktrees` config defaulted to true — `FindWorktreeForBranch` matched the hub checkout itself; now compares against `HubRoot()` (xpo-1fccf2)
- Added 11 merge test cases covering all three strategies, error paths, branch deletion, worktree cleanup, hub-wrong-branch, and issues.db preservation for FF (xpo-7a5bba)
- Backlog drag-and-drop performance — eliminated per-pointermove re-renders by moving DnD visual state (drop indicator, group highlight, nest target) from React state to refs with direct DOM manipulation; extracted `BacklogIssueRow` and `BacklogGroupHeader` as `React.memo` components (xpo-d049dc)
- Cross-group drag over ghost parent rows no longer shows a spurious indicator in the originating group — ghost rows now participate in collision detection, and `findIndex` lookups skip ghost instances (xpo-d049dc)
- Cancelling a drag with Escape no longer triggers a click-through to the issue detail view on mouseup (xpo-d049dc)
- Merge view auto-refreshes when new commits are pushed, with a manual refresh button in the tab bar; `.xpo/` files hidden from files/diff tabs (xpo-af0b09)
- Merge view walkthrough and conversation tabs now constrain to 832px and center, matching the issue detail content width (xpo-2c6297)
- ContextMenu sub-menus now flip left and clamp vertically when near viewport edges instead of clipping (xpo-911a91)
- MergeView no longer crashes when `branch_stats` becomes `undefined` after a merge completes — replaced non-null assertions with optional chaining and an early-return "branch merged" state (xpo-db1b0a)
- Global scrollbar thumb color changed from `--color-border-default` to `--color-border-overlay` for better visibility against dark menu/popover backgrounds (xpo-e1acfd)
- FilterMenu and ViewOptionsMenu popovers now center on their trigger button with viewport clamping, instead of right-aligning (xpo-374512)
- Scrollbar appearance no longer shifts layout left — scroll containers use `overflow-y: scroll` with a thin 6px transparent-track scrollbar (xpo-0b2c6c)
- Inline create row in backlog status groups now aligns with normal issue rows — icons wrapped in matching containers, blank spacer replaced with dimmed `xpo-······` placeholder ID (xpo-5d3fb0)
- Fixed all 63 frontend eslint errors across 21 files with no suppression comments — ref writes moved out of render, setState positioning replaced with direct DOM, non-component exports extracted to separate modules, missing hook deps added, unused vars removed, `any` types replaced (xpo-c08d52)
- `make test` now depends on `make lint` (`go vet` + `bun run lint`), ensuring lint errors are caught before declaring implementation done (xpo-c08d52)
- Completed ghost children (DONE/CANCELED/DUPLICATE) no longer appear in non-terminal views by default; "Show done ghosts" toggle in view options to opt back in (xpo-b109e5)
- Merge view diff/files endpoints now target the correct worktree directory instead of the hub checkout, and normalize diff output prefixes for compatibility with `diff.mnemonicPrefix` git config (xpo-2d6b57)
- `computeBranchStats` now detects uncommitted changes in worktree directories, making the merge view button appear for worktree branches with pending changes (xpo-67a1c4)
- `xpo merge` no longer blocks on untracked files (e.g. `idea.md`) on the hub checkout — only modified tracked files that actually conflict with the incoming branch are rejected (xpo-cb6b49)
- Merge error messages now explicitly warn against stashing to prevent agent stash-thrash loops that corrupt `.xpo/issues.db` (xpo-cb6b49)
- Nested hierarchy mode no longer duplicates children across status groups — ghost children under a real parent are skipped when the child's status group is visible on the same tab (xpo-93e234)
- `xpo merge` now auto-checkouts the default branch in branch mode instead of erroring — detects branch vs worktree mode at runtime by checking whether a worktree exists for the issue's branch (xpo-d5c047)
- Web merge view mergeability check updated to use the same relaxed logic (xpo-cb6b49)

## [1.1.0] - 2026-08-31

### Added

- **Terminal statuses: CANCELED and DUPLICATE** (xpo-a18d46) — two new terminal statuses alongside DONE, with `IsTerminal()` / `IsCompleted()` helpers replacing hardcoded DONE checks across the entire codebase. CANCELED uses Lucide `CircleMinus`, DUPLICATE uses `CirclePercent`, both in medium gray. All terminal statuses unblock dependents, trigger parent auto-close, and are rejected by `xpo start`. Only DONE counts toward velocity and burndown. Includes "Show empty groups" toggle in backlog View Options and a column visibility dropdown in the Board view.
- **"Done" view tab** (xpo-65e8e3) — dedicated backlog tab showing only completed issues, sorted by most recently completed first. Tab order: All Issues → Backlog → Active → Done. Supports flat/nested toggle with independent localStorage persistence.
- **Hierarchy toggle for backlog views** (xpo-9091b7) — per-view toggle (nested/flat) in a new View Options dropdown, persisted in localStorage per tab. Nested mode renders full parent-child trees with ghost parents (interleaved by sort order) and ghost children for cross-status context. Flat mode renders all issues at depth 0 grouped by parent with breadcrumbs. Ghost rows use prefixed DnD IDs to avoid duplicate registration. Dragging a parent across status groups shows the cascade "Update sub-issues?" prompt. View options dropdown consolidates layout, expand/collapse, and sort controls behind a single Settings icon.
- Expand All / Collapse All icon buttons in the backlog toolbar for toggling all parent issue nodes at once (xpo-b44c4c)
- **Worktree-based concurrent workflows** (xpo-f7ea12) — `xpo start` and `xpo merge` now use git worktrees by default, making multi-agent and multi-session work safe by design. The primary checkout stays parked on `main` as the integration hub; each `xpo start` creates an isolated worktree at `.xpo/worktrees/<branch>/`; `xpo merge` runs from the hub and cleans up the worktree automatically. All issue events serialize through the hub so the web board reflects real-time state from every agent. Opt out per-command with `--no-wt` or globally with `worktrees: false` in config.
  - Hub-rooted storage layer — all `issues.db` and artifact I/O routes through the primary checkout via `git rev-parse --git-common-dir`, so MCP servers in worktrees read/write the hub's event log (xpo-580061)
  - `xpo start` creates worktrees with `--force` takeover (removes existing worktree), `worktree_setup` config hook for post-creation build steps (e.g. `make deps`), and MCP `start` tool returns `worktree_path` (xpo-1765ca)
  - `xpo merge` verifies hub is on the default branch, skips checkout, removes worktree after merge regardless of `--keep-branch`, and relaxes clean check to allow uncommitted `.xpo/` events (xpo-f96832)
- Review view for branches with uncommitted changes — shows working-tree diff in the merge view even before committing, with merge disabled and guidance to commit first (xpo-4295bf)
- `xpo init` and `xpo doctor` now create/ensure `.gitattributes` with `merge=union` for `issues.db`, preventing merge conflicts on the append-only event log (xpo-c762f7)

- Generated `xpo-workflow` skill now includes worktree isolation guidance (`start`/`merge` tools, worktree path usage, parallel issue isolation) and an explicit review gate after the completion comment — agents must wait for user tophat before writing walkthrough, committing, or merging (xpo-cf75df)
- `xpo init` is now idempotent — re-running on an existing project refreshes agent skill files without overwriting `config.yaml`, `issues.db`, or other project state (xpo-cf75df)

### Improved

- `xpo init` now replaces stale agent instructions instead of skipping files that already have them; heading changed from "Exponential Agent Instructions" to "Agent Instructions"; both old and new headings are detected for backward compatibility (xpo-8f7b7a)
- `xpo merge` via MCP and REST API now deletes the issue branch by default after merge; pass `keep_branch: true` to opt out (xpo-408db9)
- Render cap for Board (50 per column) and Backlog (100 per group) views — reduces DOM nodes for large projects while keeping all counts and stats accurate; "+N more" button reveals the rest (xpo-f16a94)
- Label ordering: metadata labels now sort alphabetically to the left, primary labels (matching `default_labels` config) sort to the right in backlog, board, and detail views (xpo-e928c1)
- Markdown rendering: tables now have full grid borders, cell padding, and distinct header row; headings use graduated top margins for visual hierarchy; lists are nearly flush with body text; overall vertical rhythm between paragraphs, code blocks, and blockquotes increased for better readability (xpo-dbd580)
- Issue detail scrollbar moved to the right edge of the viewport; property sidebar is now sticky-positioned and scrolls independently on short screens (xpo-a8be39)
- Activity timeline system entries (status changes, merges, artifact updates) now render with tighter vertical spacing; comment cards retain generous margins (xpo-13153f)

### Fixed

- `xpo merge` with squash or ff-only strategy no longer overwrites `issues.db` events on `main` when worktrees are disabled — performs a line-based union of both versions before committing, matching the `merge=union` behavior that only fires during 3-way merges (xpo-c58af8)
- `xpo merge` now stages `.xpo/artifacts/<issue-id>/` (spec, walkthrough, generic artifacts) alongside `issues.db` in the merge commit — previously left as untracked files on `main` (xpo-dda39d)
- Backlog rows now dynamically fit labels into available space (35% of row width budget) with `+N` overflow, adapting as the window resizes; board cards cap at 2 (xpo-5f9ee7)
- Sub-issues table rows no longer wrap when issues have many labels; shows first 2 labels with "+N" for the rest (xpo-ff3cb7)
- Notification inbox labels no longer overflow when issues have many labels (xpo-7eff79)
- MCP `list` tool now returns issues in user-defined sort order instead of creation order (xpo-2792c3)
- Epic "Update sub-issues?" dialog no longer resets completed children; detail view now shows the same cascade prompt as the backlog (xpo-886099)
- Sub-issues preview in cascade dialog caps labels at 2 (primary first), truncates long titles, and scrolls when the list is long (xpo-ee99a5)
- Drag-and-drop no longer accidentally changes parent relationships; Alt is now required for nesting and unparenting (xpo-cd0f62)
- Alt parent/unparent now works in Active and Backlog views: nest highlight responds to mid-drag Alt press, ghost parent indicator placement fixed, and "Remove from parent" added to the right-click context menu (xpo-9859c9)
- Ghost parent row no longer duplicates popovers when the real parent is also visible in a different status group (xpo-377101)
- Status group header story points no longer include completed children or double-count parents; DONE parents show delivered points (xpo-dbdfd3)
- Scroll position now resets to top when navigating between issues via arrow keys or prev/next buttons (xpo-a6a17e)

## [1.0.3] - 2026-07-20

### Added

- Discard confirmation dialog when closing the new-issue modal with unsaved content — guards Escape, backdrop click, X button, and Cancel button (xpo-762a91)
- Floating formatting bar on text selection in markdown editors — text type, bold/italic/strikethrough/underline, inline link editor, quote, code, code block, and list options (xpo-df91c4)
- "Issue not found" page when navigating to an invalid issue ID, with a link back to the Backlog (xpo-147414)
- Clickable priority icon in Backlog rows — opens a priority picker popover; `p` keyboard shortcut (xpo-a274c0)

### Fixed

- Checkmark and shortcut number alignment in status, priority, and estimate picker popovers
- Generated agent instructions now include the project's issue ID prefix and an example ID (xpo-2ba758)
- Board view now sorts issues within each status column the same way as the Backlog — by `sort_order` for active statuses, by recency for DONE (xpo-337854)
- Relationship links in issue detail view now resolve correctly — dependency target IDs and parent IDs are resolved to canonical `xpo-` prefixed IDs via `GetIssue()` in MCP add/update tools (xpo-147414)
- Existing dependencies with prefixless IDs matched via suffix fallback in the frontend (xpo-147414)
- CMD+K search now uses substring matching for issue titles instead of fuzzy subsequence — eliminates false positives that buried actual results past the 12-item cap (xpo-d41e32)
- Web UI `handleDraft` endpoint now validates all input — title, status, estimate, parent ID, dependency target IDs, and JSON payload errors (xpo-49cbfe)
- `handleDraft` capability mismatch — each operation (create, update, comment, delete) now checks its specific capability instead of a blanket `issue.create` (xpo-ddb470)
- Estimate validation added to update path and negative estimates rejected in both create and update (xpo-7868ea)
- MCP link tool rejects self-links and duplicate links; deleted dependency targets shown as "(deleted)" instead of broken links (xpo-a90985)
- CycleID validated as YYYY-MM-DD format during create and update; rejected when cycles aren't enabled (xpo-e75d09)
- Length limits on string input fields — title (500), description (100KB), comment (100KB), assignee (200), labels (100), artifact content (1MB) (xpo-75f1cf)

## [1.0.2] - 2026-07-13

### Added

- **Improved agent instructions out of the box**: `xpo init` now generates a two-layer agent setup — a thin always-on stub with hard invariants in the agent instruction file (CLAUDE.md, AGENTS.md, etc.) and a portable `xpo-workflow` skill with the full development lifecycle procedure. Specs are framed as thinking tools that collapse the design space. Includes interactive/non-interactive mode guidance, spec drift prevention, walkthrough enforcement, and a harness-neutral MCP tool reference.
- **Agent auto-detection**: `xpo init` detects installed agents (Claude Code, Gemini, Cursor, Aider, etc.) via PATH lookup and writes to the correct instruction file and skill directory for each. Falls back to generic `AGENTS.md` if no agents are found.
- **MCP and agent setup during init**: `xpo init` now creates `.mcp.json` and agent instruction files automatically — no longer deferred to an interactive `xpo doctor` session.
- **Streaming progress in `xpo drive`**: real-time display of agent activity (tool use, file paths, thinking verbs) replaces static spinner. Phases are collapsible — active phase expands with steps, completed phases collapse to a single `✔ Phase (Xs)` line.
- **Supervisor model override**: `drive.supervisor.model` config key (defaults to `sonnet`) lets supervisor calls use a faster/cheaper model for spec evaluation, planning, review, and walkthrough.
- **Session reuse**: supervisor and coder executors reuse Claude CLI sessions via `--resume` for prompt cache hits, cutting token usage roughly in half.
- **Test command validation**: broken test commands (exit code ≠ 1) are detected during preparation and skipped instead of sending the coder into a retry loop. `xpo init` no longer writes a default test command.
- **Release archives**: GitHub Actions now packages release binaries as `.tar.gz` (Linux/macOS) and `.zip` (Windows) instead of plain binaries.

### Changed

- Drive config restructured: `supervisor` and `coder` are now nested objects with `agent` and `model` subkeys

### Fixed

- Normalize labels case-insensitively to prevent duplicates
- Show current week running total in dashboard velocity card
- Parse pasted text as markdown in MarkdownEditor
- Makefile `frontend` and `clean` targets now preserve `internal/server/static/.gitkeep` so CI test runs don't fail from a missing embed directory

## [1.0.1] - 2026-07-09

### Changed

- Updated agent instructions to MCP-first workflow — MCP tools are now the primary interface, CLI command reference is supplemental
- Updated README with screenshot

### Fixed

- Added placeholder file to `internal/server/static/` so Go embed works without a prior frontend build

## [1.0.0] - 2026-07-08

### Changed

- **Rebrand**: Beats → Exponential (`xpo` binary). Module path `github.com/palarix/exponential`, data dir `.xpo/`, env prefix `XPO_*`, MCP tools `xpo_*`. Clean break — no backward compatibility.
- **Public release**: MIT license under Nicolas Bettenburg, cleaned repository history.

## [0.3.0] - 2026-06-16

### Added

- **Distributed mode**: `xpo serve` runs a headless server; `xpo board` proxies to a remote instance
- **Authentication**: SSH public-key challenge/response auth with JWT tokens (`xpo login`/`logout`/`whoami`)
- **Role-based permissions**: configurable per-user capabilities via `config.yaml`
- **Inbox**: personal notification feed — CLI (`xpo inbox`) and WebUI with grouped notifications and tabbed read state
- **Cycles**: configurable sprint/cycle planning with progress tracking and burndown charts
- **Branch workflows**: `xpo start` creates a feature branch and locks the issue; `xpo merge` squash-merges and closes
- **Code review**: `xpo review` shows a unified diff from the terminal; WebUI diff viewer with per-commit navigation
- **Dashboard**: pulse metrics — velocity charts (weekly + daily), flow stats (cycle time, lead time), WIP tracking, epic progress, attention items, workload distribution, and bug age
- **SSE**: real-time board updates via Server-Sent Events
- **Filtering**: structured backlog filtering with nested menus and status bar chips
- **Drag-and-drop**: reorder issues across status groups with globally consistent sort order (data model v3)
- **Docker**: `Dockerfile` for packaging `xpo serve`
- **MCP server**: `merge` tool; `.mcp.json` auto-detection in `xpo init`/`xpo doctor`
- **Contributors**: configurable contributor list for assignment
- **Empty state**: full-page onboarding for new projects

### Changed

- Data model upgraded to v3 — globally consistent `sort_order` keys with automatic migration from v2
- Error messages audited for user-facing clarity — no raw Go errors or stack traces leak to clients
- Issue cache: projected issues are cached instead of reprojected on every API call
- Request logging: structured JSON logs for `xpo serve`
- Story point display toggle persisted in localStorage

### Fixed

- Sub-issue ordering and dependency view toggle
- Bulk status updates for parents with sub-issues
- Description flicker on save in issue detail view
- Title flicker on edit and link clicks entering edit mode
- Keyboard focus and navigation in filter popover
- Auto-progressed parent status correctly marked as inferred
- Drag-and-drop sorting to top of backlog group
- Unestimated issues count as 1 point; epics excluded from totals
- Epics show remaining story points instead of full count
- Status icons and borderless label badges in new issue dialog
- New issues append at end of backlog
- Login credential and config storage consistency
- Collapse/prune handling of status field in CreatePayload
- Inbox participation filter counts update authors
- Nil-slice JSON responses return `[]` instead of `null`

## [0.2.0] - 2026-04-01

### Added

- **Web UI**: `xpo board` serves a React-based board view with embedded static assets
- **Comments**: add and view comments on issues (`xpo comment`, `xpo comments`)
- **Dependencies**: `xpo link` command to express issue relationships (blocks, depends_on, relates_to, etc.)
- **Archive**: `xpo archive` moves old DONE/deleted issues to a separate store; regular commands search the archive as fallback
- **Delete**: `xpo delete` with cascade option for parent issues
- **Interactive mode**: Bubble Tea TUI for `xpo add` and `xpo update`
- **Autocomplete**: shell completions for issue IDs across all commands
- **Version command**: `xpo version`
- **Filtering**: `--since`, `--before`, `-m` (match) flags on `xpo ls`; hide DONE by default (`-a` to show all)
- **Duplicate detection**: warn when creating issues with similar titles

### Changed

- **Breaking**: renamed `issues.jsonl` to `issues.db`
- **Breaking**: upgraded to v2 data model — single Issue type with labels, assignee, estimation, automations
- Dynamic column widths based on terminal size
- Markdown-rendered descriptions in terminal
- Separated UI and business logic from CLI commands

### Fixed

- First item in list had purple background highlight
- First item in list had extra indentation
- Sorting issues correctly
- Timezone handling (all dates now UTC)
- Interactive add truncating title after first whitespace
- `xpo config` keys missing underscores
- ANSI escape sequences in bash completions

## [0.1.0] - 2026-02-01

### Added

- Initial release
- Event-sourced issue tracker stored in `.xpo/issues.db`
- CLI commands: `xpo init`, `xpo add`, `xpo update`, `xpo ls`, `xpo show`, `xpo history`
- `xpo doctor` for health checks
- `xpo snapshot` for point-in-time state export
- Git auto-commit support (configurable)
- Configurable issue ID prefix
- Parent/child issue hierarchy with auto-sync
- Status workflow: BACKLOG → PLANNED → DOING → BLOCKED → DONE
