## What changed

The Agent view now uses the `scroll-stable` utility (`web/src/index.css`, from xpo-c26f7b) on every scroller, not just the chat thread.

- **New-chat landing pane** (`AgentView.tsx`): `overflow-y-auto` → `scroll-stable`. The pane now always keeps a scrollbar gutter on both sides, so its centered content keeps its position whether the recent-chats list overflows or not.
- **All chats** (`ChatHistory.tsx`): removed `overflow-y-auto`. It was redundant, because `scroll-stable` already sets `overflow-y: scroll`.

## Why the composer doesn't move

In a thread, the composer sits below the thread's scroller, outside it. On the landing pane, the composer sits inside the scroller. Because the gutter is `both-edges`, it takes the same width on the left and right, so the landing content stays centered on the pane's middle, like the thread's composer. Before this change, the landing pane only drew a scrollbar when its content overflowed. Content shifted sideways when overflow started or stopped. The thread's always-present gutter made the move to a thread look like a jump.

At very narrow widths, the landing composer can be a few pixels narrower than in a thread, because the gutters on both sides take some of the width. Its center stays the same.

## Tests

The scrollers have `data-landing-scroller` and `data-history-scroller` attributes, like `data-thread-scroller`. A new test in `AgentView.test.tsx` renders both pages. It checks that each scroller has `scroll-stable` and no `overflow-y-*` class. A leftover `overflow-y-auto` would be harmless next to `scroll-stable`, but the test keeps the rule strict.

## Acceptance criteria

- [x] The landing pane in `AgentView.tsx` uses `scroll-stable` instead of `overflow-y-auto`. Evidence: `AgentView.tsx:182`, and the new test.
- [x] The composer and centered content keep the same horizontal position when moving from the landing pane to a thread. Evidence: the gutter is reserved on both sides, and the user checked it in the dev server.
- [x] `ChatHistory` uses only `scroll-stable`. Evidence: `ChatHistory.tsx:104`, and the new test.
- [x] A test asserts the scroll classes. Evidence: "reserves a stable scrollbar gutter on the landing and history panes" in `AgentView.test.tsx`. It failed before the fix and passes after it. `make test` passes.
