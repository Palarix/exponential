# Command palette (⌘K): find and switch to agent chats

## What

Add a **Chats** group to the ⌘K command palette. It finds active agent chats by title or by linked issue, and opens the chosen chat.

## Why

The palette searches actions, navigation and issues, but not chats. There is no keyboard way to jump to an earlier chat.

## How

### 1. Matching: `searchChats(chats, issues, query)` (pure, tested)

New file `web/src/components/CommandPalette/chat-search.ts`.

- Input: the active chats (`agent.chats`, newest first, archived chats excluded), the issues, the query.
- A chat's **linked issue** is `work.issue_id` while in Work Mode, else the last entry of `worked_issues`. None for chats that never worked on an issue.
- Empty query: the 5 newest chats.
- With a query, score each chat; drop chats that score 0; sort by score, ties keep newest first; at most 8.
  - Query is a substring of any ID in `worked_issues` or of `work.issue_id`: 6000 (same weight as an issue ID match). Earlier Work Mode issues match too.
  - Else the best of `fuzzyScore(chat.title, query)` and `fuzzyScore(linkedIssue.title, query)`.
- Returns `{ chat, issueId?: string }[]`.

### 2. Palette rows

`chatCommandItems(matches, query, onOpen)` in `web/src/components/CommandPalette/chat-items.tsx` turns the matches into palette items (`CommandItem` moves to a shared `types.ts`), so the rows and their action are testable without a DOM (the palette renders in a portal).

- Group name: **Chats** with a query, **Recent Chats** without.
- Group order: Actions, Navigation, Issues, Chats.
- Icon: `IssueChatStateIcon` with the chat's state (waiting, running, interrupted, work, idle).
- Label: chat title, or "Untitled chat".
- Meta: the linked issue ID (mono, muted, like issue rows), then the relative last-activity time.
- Action: `onOpen(chat.id)`; the palette passes a callback that closes the palette and calls `agent.openChat(id)` (opens the popup, which adds its status-bar pill).
- Outside `AgentChatProvider` (`useOptionalAgentChat()` returns null), no Chats group.

### 3. Placeholder

`Search issues, chats, actions, navigation...`

## Edge cases

- No chats: no group, no empty-state row.
- Archived chats are not listed (they live on the All chats page).
- A chat whose linked issue isn't in `issues` (e.g. deleted): ID still shown and matched; title match skipped.

## Acceptance criteria

- [ ] Typing part of a chat title in ⌘K lists the matching chat under "Chats"
- [ ] Typing a Work Mode chat's issue ID lists that chat
- [ ] Empty query shows recent chats
- [ ] Enter / click opens the chosen chat

## Tests

- `chat-search.test.ts`: title match, issue ID match (current work and earlier `worked_issues`), linked issue title match, empty query returns 5 newest, no match returns empty, limit 8, linked issue choice, deleted issue ID.
- `chat-items.test.tsx`: group names, label fallback, meta shows the issue ID, action calls `onOpen` with the chat ID.

## Decisions (confirmed by user)

1. Empty query shows the 5 newest chats.
2. Issues come before Chats.
3. ID matching includes every `worked_issues` entry.
