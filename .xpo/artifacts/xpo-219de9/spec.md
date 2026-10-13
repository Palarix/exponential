# Agent chat: show linked chats on issues

## What

Issues show the chats that worked on them.

- **Rows:** backlog rows and board cards get a monochrome chat pill with the chat count. Its icon shape shows the chats' state. Branch, attachment and sub-issue indicators share the same pill, and pills and labels stay dark on a highlighted row.
- **Sidebar:** issue details gets a "Chats" button next to Relations. It opens a searchable popover with the issue's chats.
- **Context menu:** the issue context menu on backlog rows and board cards gets a "Chats" submenu with the same list, and "Open work chat" when a chat is in Work Mode on the issue.
- **Entering Work Mode from history:** when no chat is in Work Mode on the issue, a past chat can be put back into Work Mode from its row's menu in the chat list.
- **Cold cache warning:** a chat whose prompt cache has expired shows a "Context cache expired" divider, so you know the next message re-sends the whole conversation.

## Why

xpo-00f0e5 binds a chat to an issue in Work Mode, but the issue doesn't show its chats. From the board, the user can't see which issues have a chat or jump to one. The only way back is Recent chats.

Continue Work keeps its xpo-00f0e5 behavior. Exiting Work Mode means "I'm done with that context window", so Continue Work opens the chat that is still in Work Mode, or creates a new one. Picking up an older conversation is a deliberate choice, made from the chats list.

The rule from xpo-00f0e5 still holds: at most one chat is in Work Mode on an issue at any moment. Entering Work Mode from the list goes through the same `BindWork` check.

## Dependencies

- **xpo-00f0e5 (done):** chat Meta records `work` and `worked_issues`, and adds the work-chat endpoint and Continue Work.
- **xpo-b45049 (related):** a chat opened with `openChat` gets no status-bar pill. Chats opened from the row icon or the popover inherit this until xpo-b45049 is fixed.

## Measurements

- Claude Code 2.1.296 reports the cache TTL each turn wrote: the `result` event's `usage.cache_creation` has `ephemeral_1h_input_tokens` and `ephemeral_5m_input_tokens`. Measured on 2026-10-13 with a subscription login, the TTL was 1h.
- Codex reports no cache TTL. OpenAI documents that cached prefixes stay for 5 to 10 minutes of inactivity. xpo assumes 5 minutes.

## Acceptance Criteria

**Row icon**
- [ ] A backlog row and a board card show a chat pill when at least one chat worked on the issue (`worked_issues`). The pill shows the chat count, also when it is 1. The tooltip names it ("2 chats on xpo-…").
- [ ] The chat pill is a button: it gets a background on hover.
- [ ] The icon follows the most urgent state among the issue's chats. Icons are monochrome, like the rest of the row; the shape tells the state:
  - waiting for an approval → `message-square-dot`;
  - running → `message-square-more`;
  - interrupted → `message-square-warning`;
  - idle and in Work Mode on the issue → `message-square`;
  - otherwise → `message-square`.
- [ ] With a chat in Work Mode on the issue, clicking the icon opens that chat. Otherwise it opens the chats popover. The click never opens the issue.

**Row indicators (folded in from xpo-f8df84)**
- [ ] On backlog rows, board cards and My Issues, every row indicator uses one pill style, after Linear: fully rounded, 24px tall and at least 32px wide, a faint border (subtler than label chips), a muted 12px icon.
- [ ] The branch pill shows its commit count and the chat pill its chat count, always, also 0 or 1. The attachment pill shows only its icon. Details are in the tooltip: the branch's commits, files and line changes (or "no commits yet"), the chat count, the number of attachments.
- [ ] The branch pill keeps its dot for uncommitted changes.
- [ ] Sub-issue progress uses the same pill and keeps its "1/6" count; that number is the information.
- [ ] Only clickable pills (the chat icon) get a background on hover.
- [ ] Pills have the surface background (`--color-surface`), so they stay dark when the row is hovered or highlighted.
- [ ] Bordered label chips get the same surface background, everywhere labels appear.

**Issue details**
- [ ] Issue details shows a "[k] Chats" button below Relations when the issue has chats.
- [ ] The button opens the chats popover.

**Chats list (popover and submenu)**
- [ ] The list shows the issue's chats, newest first. Each row shows the state icon, title and last activity. The chat in Work Mode carries a "Work" badge.
- [ ] A search field filters the list by title.
- [ ] Clicking a row opens the chat and closes the popover or menu. It doesn't change the chat's mode.
- [ ] Rows carry no buttons. A row's actions open with a right-click, or with a `⋯` button that shows only on hover or keyboard focus.
- [ ] With no chat in Work Mode on the issue, a row's actions include "Enter work mode".
  - It binds that chat to the issue and opens it with the draft "Let's continue with xpo-….".
  - A BACKLOG or PLANNED issue is started first, as with Start Work.
- [ ] "Enter work mode" is disabled on a running chat, and on a chat in Work Mode on another issue. Its hint says why.

**Issue context menu**
- [ ] On an issue with chats, the context menu on backlog rows and board cards has a "Chats" submenu with the chats list.
- [ ] With a chat in Work Mode on the issue, it also has "Open work chat".

**Continue Work**
- [ ] Continue Work keeps its xpo-00f0e5 behavior: it opens the chat in Work Mode on the issue, or creates a new chat.

**Cold cache warning**
- [ ] Each finished turn records when its prompt cache expires (`cache_expires_at` in chat Meta). Claude's TTL comes from the turn's usage; Codex uses 5 minutes.
- [ ] A chat whose cache has expired shows a divider below its last turn: "Context cache expired". Its lines are dashed, unlike the solid lines of mode dividers, because it does not separate modes. Its tooltip says "Your next message re-sends the whole conversation." The divider appears while the chat is open as soon as the time passes, and the thread scrolls to it.
- [ ] The marker goes away when a new turn starts.

**Thread and composer polish**
- [ ] Mode dividers read "Entered Work Mode on xpo-…" and "Entered Chat Mode".
- [ ] In Work Mode, the composer's mode segment shows the hammer and the issue ID only ("🔨 xpo-…"), without "Work ·". Its accessible name stays "Work Mode on xpo-…".
- [ ] The composer's send and stop buttons are 24×24px circles. They no longer shrink in the footer row.

**General**
- [ ] Nothing shows when the agent is unavailable, including in remote mode.
- [ ] Icons and the popover update when a chat enters or leaves Work Mode, starts or ends a turn, or is deleted.
- [ ] `make test` passes.

## Flow

### 1. Cache expiry (`internal/chat`)

1. Parse `usage.cache_creation` in the Claude `result` event. The TTL is 1h when `ephemeral_1h_input_tokens > 0`, and 5m when `ephemeral_5m_input_tokens > 0`.
2. A turn that only read the cache, with no creation, refreshes the TTL it read. Keep the chat's last known TTL. Default to 5m.
3. Add `CacheTTL` (seconds) to the `turn_done` event. Add `CacheExpiresAt` to `Meta`: the turn's end time plus the TTL.
4. Codex turns record 5 minutes.

### 2. Enter Work Mode with an existing chat (`internal/server/chat.go`)

1. `POST /api/local/issues/{id}/work-chat` takes an optional body `{"chat_id": "…"}`.
2. With `chat_id`:
   - If another chat holds the issue, respond 409 (`ErrIssueHeld`).
   - Otherwise, start the issue or find its worktree as today, then `BindWork` the given chat.
   - Respond `200 {chat, created: false, resumed: true}`.
   - If the chat is running, `BindWork` returns `ErrBusy` → 409.
3. Without `chat_id`, nothing changes. This is what Continue Work uses.

### 3. Frontend: issue-to-chats lookup

1. Add `chatsForIssue(chats, issueId)`. It returns `{holder?: ChatMeta, chats: ChatMeta[], state}`:
   - chats are filtered by `worked_issues`, newest first;
   - `state` is the most urgent of waiting, running, interrupted, work or idle.
2. Read the chats from `useOptionalAgentChat()`, which already loads the chat list. Outside an agent provider, or with the agent unavailable, the list is empty.
3. Poll the chat list while any linked chat runs, not only chats with a pill, so row icons stay current. Invalidate it after work-chat and leave-work calls.

### 4. Components (`web/src/components/Agent`)

1. `IssueChatsList`: the searchable list, with each row's action menu ("Open", "Enter work mode"). `IssueChatsPopover` wraps it for the row icon and the sidebar; the context menu's "Chats" submenu renders it directly.
2. `IssueChatIcon`: the chat pill with its count. A click opens the holder or the popover.
3. Place `IssueChatIcon` after the branch badge in `BacklogIssueRow` and `BoardCardContent`.
4. Add the "Chats" submenu and "Open work chat" to `ui/ContextMenu.tsx`, shown only when the issue has chats.
5. Add a `PropertyRow` below Relationships in `PropertySidebar.tsx`: a `message-square` icon and "1 Chat" or "k Chats".

### 5. Cold cache marker

1. `ChatThread` renders the marker as a divider after the last turn when the chat isn't running and `cache_expires_at` has passed.
2. A timer set to `cache_expires_at` re-renders the thread, so the marker appears while the chat is open.

## Decisions

- **Continue Work doesn't reopen a chat that left Work Mode.** Leaving Work Mode closes that context window. Going back to it is a choice made in the chats list.
- **Row actions hide behind a menu.** A button on every chat row is visual noise. "Enter work mode" is a rare, deliberate action, so a right-click or `⋯` menu fits it.
- **One state icon per issue, not per chat.** The row has room for one icon. The most urgent chat decides its icon. The popover shows each chat's own state.
- **Lucide `message-square` variants instead of new symbols.** They separate waiting (dot), running (three dots) and interrupted (warning) by shape.
- **One pill style for row indicators, after Linear.** Square bordered chips with numbers next to bare icons made the rows busy and incoherent; bare icons alone looked unanchored. Every indicator sits in the same faint, fully rounded pill. Counts stay where they read at a glance (commits, chats, sub-issues), always shown so pills don't change width by meaning. Pills and label chips sit on the surface color, so they stay dark and grounded on a highlighted row. Chat state shows by shape only, never by color or brightness.
- **The cache TTL comes from the agent when it reports it.** Claude reports the TTL it wrote. A fixed guess would be wrong by 55 minutes for subscription logins.
- **No new list API.** The chat list already carries `work` and `worked_issues`, and the provider already loads it.
- **Only Work Mode links a chat to an issue.** A chat that only mentioned an issue isn't tracked.

## Edge Cases

**MEDIUM (agent proposes, developer confirms)**
- A chat worked on several issues. It's listed on each. Entering Work Mode on one issue is refused when the chat is in Work Mode on another, so it never drops another issue's writer.
- The user enters Work Mode with a chat whose cache is cold. It works; the marker already shows the cost.
- A model change also empties the cache (measured in xpo-00f0e5). The marker doesn't detect that; it tracks time only.

**LOW (agent handles)**
- The chat list loads after the board. Icons appear once it arrives.
- A chat is deleted. Its entry and its count go away on the next fetch.
- A chat from before this change has no `cache_expires_at`. It shows no marker until its next turn.

## Out of scope

- Status-bar pills for opened chats: xpo-b45049.
- Tracking chats that only mentioned an issue.
- Team and remote views: chats are user-local.
