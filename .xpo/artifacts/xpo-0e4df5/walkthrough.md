# Walkthrough: Agent chat turn rail

## What was built
The full-page Agent view gets a rail of short bars on the thread's left edge, one per user message. Hovering magnifies the bars and shows the message in a tooltip. Clicking a bar scrolls that turn to the top. A bar is bright while its message bubble is on screen. The chat popup has no rail: it is the lighter view you carry around the app.

## How the pieces fit
- **`ChatThread` wrapper.** The scroller used to be the component's root. It now sits in a non-scrolling `relative flex-1 min-h-0 flex flex-col` wrapper, and the rail is absolutely positioned in that wrapper, so it doesn't scroll with the transcript. The wrapper needs `min-h-0`, because its overflow is visible; the scroller itself doesn't, because its overflow is not. The scroller carries `data-thread-scroller`, which the scrollbar-gutter test now uses to find it.
- **Anchors.** `Turn` takes its `index` and renders `data-turn={index}`. The user bubble gets `data-user-message`. The rail queries both inside the scroller rather than keeping refs per turn, which keeps `Turn` unchanged apart from the two attributes.
- **`TurnRail`.** It renders a `<nav aria-label="Turns">` with one `<button>` per user turn. Positions come from `contentSpan`, which converts `getBoundingClientRect` into scroller content coordinates (`rect.top - scroller.top + scrollTop`). That doesn't depend on which element is the `offsetParent`.
- **Helpers in `chat-thread-utils.ts`.**
  - `userTurns(thread)`: indices of turns that are neither `continued` nor `byAgent`.
  - `visibleTurns(spans, viewTop, viewBottom)`: indices of the spans that overlap the view.

  Both are pure and unit-tested; the DOM code only measures and maps.
- **Highlight updates.** A passive scroll listener, throttled with `requestAnimationFrame`. The effect depends on a string key of the user-turn indices, so streaming output doesn't re-subscribe it on every event.

## Decisions from review
- **Agent view only.** The popup is the lower-fidelity view and stays as is.
- **Only user messages get bars**, and bars are plain: no colors for agent or failed turns.
- **Fixed slots, not flex-shared height.** The first version used `flex-1` with a max height. The nav has no height of its own, so the slots collapsed to the 2px bar and the bars nearly touched. Each button is now a fixed 10px slot (8px gap), 14px while hovering. The slots can still shrink under the nav's `max-h` when a chat has many messages.
- **Lighter at rest.** 12 × 2px bars; on rail hover 24 × 3px, and the bar under the pointer 40px.
- **Keyboard-only focus magnify.** `group-focus-within` kept the rail large after a click, until the user clicked elsewhere. It now uses `group-has-[:focus-visible]`, so only keyboard focus magnifies.
- **Highlight = message visible.** First it was the topmost turn, then every turn whose span (up to the next turn) was in view. The final rule: bright exactly while the user bubble overlaps the viewport. So none are bright while you read a long reply.
- **Position.** We tried a spot 1em left of the reading column and went back to the view's edge. The rail is at `left-6`, matching the Agent view top bar's `px-6`, so the bars line up with the chat title.
- **Tooltip** uses AI surface tokens (`surface-ai-2`, `border-ai`), as `ai-surface.test.ts` requires for `ChatThread.tsx`.

## Acceptance criteria
- [x] Agent view with 2+ user messages shows a rail, one bar per user message, centered on the left edge and aligned with the title; none with 0 or 1. Evidence: "shows one bar per user message in the Agent view", "has no rail … with fewer than two user messages"; position checked by the user.
- [x] The popup shows no rail. Evidence: "has no rail in the popup" (no `wide`).
- [x] Continued and agent-started turns get no bar. Evidence: `userTurns` test; the rail test's agent turn gets no button.
- [x] The rail stays in place while scrolling: it lives outside the scroller. Checked by the user.
- [x] 8px gap at rest, magnify on hover with the hovered bar largest, no stuck magnify after a click. Evidence: slot class test (`h-2.5`); checked by the user.
- [x] A click scrolls the message to the top, and new output doesn't pull the view back (`nextStick` stops following on an upward scroll). Checked by the user.
- [x] A bar is highlighted exactly while its message is visible. Evidence: `visibleTurns` tests, including "lists none while only the agent's reply is in view"; checked by the user.
- [x] Tooltip and accessible labels. Evidence: `aria-label="Go to message: …"` asserted in the rail test; buttons are native and keyboard-focusable.
- [x] `make test` passes.
