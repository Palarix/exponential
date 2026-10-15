# Agent chat: turn indicators for jumping between turns

## What
A thin rail of turn markers on the left edge of the full-page Agent view's thread. There is one short horizontal bar per user message, stacked vertically and centered on the left edge. Hovering the rail magnifies the bars. Clicking a bar scrolls the thread to that message.

## Why
Long chats are hard to navigate. The only way back to an earlier turn is to scroll and scan. The rail shows how many messages the chat has and which are on screen, and gets you to any of them in one click.

The chat popup gets no rail. It is the lower-fidelity view that travels with you while you work elsewhere in the app; it stays fully functional without one.

## How
- **Where.** Only when `ChatThread` is `wide` (the Agent view).
- **Layout.** `ChatThread` gets a non-scrolling `relative` wrapper (`flex-1 min-h-0 flex flex-col`) around the scroller. The rail sits in the wrapper at `left-6`, vertically centered, so it stays put while the thread scrolls. `left-6` matches the Agent view top bar's `px-6`, so the bars' left edge lines up with the chat title.
- **Which turns.** One bar per turn with a user message: not turns the board continued (`continued`) and not turns the agent started (`byAgent`). A pure helper, `userTurns(thread)`, returns their turn indices.
- **Anchors.** Each `Turn` root gets `data-turn={i}`; the user bubble gets `data-user-message`. A bar click smooth-scrolls the turn's root to the top, minus 16px. Scrolling up already turns off stick-to-bottom (`useStickToBottom`), so following stops without new code.
- **Bars.** `TurnRail` renders a `<nav aria-label="Turns">` with one `<button>` per bar. Each button is a fixed 10px slot, 48px wide, holding a 12 × 2px bar in `--color-border-ai`: an 8px gap between bars at rest.
- **Magnify.** On rail hover, every slot grows to 14px and every bar to 24 × 3px. The bar under the pointer grows to 40px and takes the primary text color. Keyboard focus (`:focus-visible`) magnifies too; mouse focus after a click does not, so the rail shrinks back once the pointer leaves. Bars grow rightward. CSS transitions only; nothing is computed per pointer move.
- **In view.** A bar is drawn in `--color-text-muted` while its user message bubble overlaps the viewport, so several bars can be bright at once and none while only replies are on screen. A passive scroll listener updates this, throttled with `requestAnimationFrame`. A pure helper, `visibleTurns(spans, viewTop, viewBottom)`, picks them and is unit-tested.
- **Label.** Hovering a bar shows a tooltip to the right with the user message, cut to one line, in AI surface tokens. The button's `aria-label` is "Go to message: <text>".
- **Many turns.** The rail is at most the thread height minus 4rem; slots shrink to fit.
- **When shown.** Only with 2 or more user messages.
- **Bars are plain.** No colors for agent-started or failed turns.

## Acceptance criteria
- [ ] In the Agent view, with 2+ user messages, a rail of bars, one per user message, shows centered vertically on the left edge of the thread, aligned with the top bar's title. With 0 or 1 there is no rail.
- [ ] The popup shows no rail.
- [ ] Continued and agent-started turns get no bar.
- [ ] The rail stays in place while the thread scrolls.
- [ ] Bars have an 8px gap at rest. Hovering the rail magnifies the bars; the hovered bar is the largest. A click does not leave the rail magnified.
- [ ] Clicking a bar scrolls that message to the top of the thread; new output doesn't pull the view back to the bottom afterwards.
- [ ] A bar is highlighted exactly while its user message is visible.
- [ ] Hovering a bar shows the message in a tooltip; buttons have accessible labels and work with the keyboard.
- [ ] `make test` passes.
