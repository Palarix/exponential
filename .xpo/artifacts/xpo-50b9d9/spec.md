# Agent chat: smooth scroll and fade-in when new content streams in

## What
While the chat follows new output, ease the scroller toward the bottom instead of jumping. Fade in each new block of a thread (messages, text chunks, tool groups, cards, status lines, dividers) as it appears.

## Why
Today the view jumps a few lines on each streamed chunk, and blocks appear at once. Gliding and fading make the stream easier to read.

## How

### When the thread is "live"
`useChatThread` resets the thread, then loads the history asynchronously, on open, on chat switch, and on reload. The thread mounts empty before its history arrives, so "after first render" would fade and glide the whole history. Instead the thread state gets a `loaded` flag:
- New reducer action `{ type: "loaded" }` sets `loaded: true`. `emptyThread` and `reset` have `loaded: false`.
- `useChatThread` dispatches `loaded` after the fetched events, background and queue, and right after the reset for a new chat (no chatId).
- `ChatThread` keeps a `live` ref, set to `thread.loaded` in an effect. The commit that brings the history in still sees `live` false, so it renders and pins without fade or glide. Everything after it is live.

### Smooth follow (`useStickToBottom(glide)`)
- The hook takes the `live` ref as `glide`. `pin()` stays the single entry point. While `glide.current` is true and `motionAllowed()`, it starts (or keeps running) one rAF loop instead of setting `scrollTop`. Each frame moves `scrollTop` part of the way to the **current** bottom (`scrollHeight - clientHeight`), so a new chunk mid-glide only moves the target.
- Step: exponential approach with a time constant of `--duration-normal` (150ms), independent of frame rate: `top + (target - top) * (1 - exp(-dt / tau))`. It moves at least 1px, since browsers may round `scrollTop`, and lands within 1px. Pure helper `easeToward(top, target, dtMs, tauMs)` in `utils/scroll.ts`.
- The time constant comes from the CSS token, read once with `getComputedStyle` (`parseDuration("150ms") → 150`; fallback 150).
- **Jump, don't glide** when movement is not allowed (Off, or System default with OS reduced motion), and while the thread is not live (opening, switching or reloading a chat lands at the bottom at once). A resized scroller (taller composer, resized popup) glides too; to revisit at tophat, if it looks wrong it jumps instead.
- **Following:** the loop only moves down, so `nextStick` keeps following (`top >= prevTop`). Each frame first runs `nextStick` on the current position, so a user scroll up between frames stops following and the loop. The scroll handler also stops the loop when following turns off. The loop updates `lastTop` with every write.

### Fade-in (`ChatThread`)
- New utility `animate-stream-in`: the `fade-in` keyframes at `--duration-normal`, ease-out. Fade tier: plays under On and System default (with or without OS reduced motion), none under Off. Added to the Off rule in `index.css` and the tier table.
- `stream-in.ts`: the `StreamIn` context (the `live` ref) and `useStreamIn()`, which reads the ref once at mount (`useState`) and returns the class or `""`.
- `Block` in `ChatThread.tsx`: a `div` that appends that class. Each block decides at its own mount (a turn's footer mounts long after the turn), with no extra DOM.
- Blocks that fade: user message and its attachments, agent-continued line, text segment, answered question, steer message, tool group, thinking line, approval/question cards (one wrapper div each), stopped / interrupted / error lines, the turn footer, dividers (mode and cold cache).
- Text that grows inside an existing text block does not fade per token: the block fades when it appears, later chunks extend it.

## Decisions
- Custom rAF ease rather than `scrollTo({ behavior: "smooth" })`: native smooth scroll restarts on every chunk, and its duration is not ours to set.
- The scroll counts as movement (jumps under reduced motion); the fade counts as a fade (plays under reduced motion). This matches the tiers from xpo-c80c0d.
- Glide time constant is 150ms (`--duration-normal`), to be judged at tophat.
- New content can be below the fold for the length of the glide; that is the glide itself, not the old after-paint flash. Pins still run in a layout effect, so the glide starts before paint.
- `Turn` is exported for the fade test, as `ToolTimeline` already is.

## Acceptance criteria
- [ ] Following, new output glides the view to the bottom over about `--duration-normal`; repeated chunks keep gliding without stutter or stopping short.
- [ ] Following, scrolling up during a glide stops following and the glide; scrolling back to the end resumes.
- [ ] A glide never switches following off.
- [ ] Opening, switching or reloading a chat lands at the bottom without a glide; resizing the popup or growing the composer glides the last line back into view.
- [ ] Motion Off, or System default with OS reduced motion: following jumps as before.
- [ ] New blocks fade in under On and System default (also with OS reduced motion); no fade under Off.
- [ ] Blocks already in the thread when it opens do not fade.
- [ ] Unit tests: `easeToward`, `parseDuration`, the `loaded` reducer action, the `animate-stream-in` tier rule (`motion-css.test.ts`), a freshly rendered thread has no `animate-stream-in`, a live turn's blocks do.
- [ ] `make test` passes.
