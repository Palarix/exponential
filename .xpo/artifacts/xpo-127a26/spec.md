## What

Use `scroll-stable` everywhere in the Agent view: on the new-chat landing pane, and as the only scroll class on the All chats (ChatHistory) scroller.

## Why

The landing pane uses plain `overflow-y-auto`, so it has no gutter. Content shifts sideways when the recent-chats list starts to overflow, and when you go from the landing pane to a thread, whose scroller already uses `scroll-stable` (rule from xpo-c26f7b).

## How

- `AgentView.tsx`: on the landing pane, replace `overflow-y-auto` with `scroll-stable`. Add `data-landing-scroller` so the test can find it, like `data-thread-scroller`.
- `ChatHistory.tsx`: remove the redundant `overflow-y-auto` (the `scroll-stable` utility already sets `overflow-y: scroll`). Add `data-history-scroller`.
- The composer keeps its horizontal center. The gutter is `both-edges`, so it is symmetric and the landing content stays centered on the same axis as the thread's composer.

## Acceptance criteria

- [ ] The landing pane in `AgentView.tsx` uses `scroll-stable` instead of `overflow-y-auto`.
- [ ] The composer and centered content keep the same horizontal position when moving from the landing pane to a thread.
- [ ] `ChatHistory` uses only `scroll-stable`.
- [ ] Tests in `AgentView.test.tsx` assert that the landing and history scrollers have `scroll-stable` and no `overflow-y-*`, like the ChatThread scroller test.
