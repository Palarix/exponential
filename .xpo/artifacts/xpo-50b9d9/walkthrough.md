# Agent chat: smooth scroll and fade-in when new content streams in

## What changed

While the chat follows new output, the view now eases to the bottom instead of jumping. New blocks of the thread fade in. Both follow the Motion and Animations preference from xpo-c80c0d. The glide counts as movement: it jumps under reduced motion and under Off. The fade counts as a fade: it stops only under Off.

## How the pieces fit

### "Live": when new output starts

The glide and the fade must only apply to new output, never to a chat's history. That rule turned out to need state the thread did not have.

`useChatThread` dispatches `reset`, then fetches the history and dispatches it later. That happens on open, on chat switch, and when the server drops the topic and the chat reloads. `ChatThread` therefore first renders an empty thread, then the whole history in a later commit. A "live after first render" rule would have faded in and glided through the entire history every time.

So the thread state gained `loaded` (`chat-thread-utils.ts`):
- `emptyThread` and `reset` have `loaded: false`.
- A new `{ type: "loaded" }` action sets it. `useChatThread` dispatches it after the fetched events, background tasks and queue. For a new chat (no chatId), it dispatches it right after the reset.

`ChatThread` keeps a `live` ref and sets it from `thread.loaded` in a `useEffect`. Effects run after the commit, so the commit that brings the history in still sees `live === false`: its blocks mount without a fade and its pin jumps. Every later commit is live.

### Glide (`useStickToBottom(glide)`)

The hook takes the `live` ref. `pin()` is still the only entry point, called from the layout effect after each render and from the `ResizeObserver`.

- While `glide.current && motionAllowed()`, `pin()` starts a `requestAnimationFrame` loop if none is running and otherwise returns. Each frame reads the **current** bottom (`scrollHeight - clientHeight`), so chunks that land mid-glide only move the target. A fast stream is one continuous glide, not a restart per chunk.
- Each frame steps with `easeToward(top, target, dt, tau)` (`utils/scroll.ts`): an exponential approach, `top + (target - top) * (1 - exp(-dt/tau))`, which takes the same time at any frame rate. Two details matter:
  - It moves at least 1px. Browsers may round `scrollTop` to whole pixels, and a sub-pixel step would round back to the same value forever.
  - It lands exactly within 1px, and the loop stops once the browser reports the end.
- `tau` is the `--duration-normal` token (150ms), read once with `getComputedStyle` and parsed by `parseDuration`.
- Otherwise `pin()` cancels any running loop and jumps, as before.

**Following stays correct.** The loop only moves down and writes `lastTop` after each write, so `nextStick` sees `top >= prevTop` and keeps following. A user can scroll up between frames before their scroll event fires. Each frame therefore first runs `nextStick` on the current position, and the scroll handler also cancels the loop when following turns off. The loop never fights the user.

A resized scroller (taller composer, resized popup) glides too. We tried that at tophat and kept it.

### Fade (`stream-in.ts`, `Block`, `animate-stream-in`)

- `animate-stream-in` in `index.css` is the existing `fade-in` keyframes at `--duration-normal` (`animate-fade-in` uses `--duration-fast`). It belongs to the Fade tier: it is listed in the `[data-motion="off"]` rule and the tier table, and is left alone under reduced motion.
- `stream-in.ts` holds the `StreamIn` context (the `live` ref) and `useStreamIn()`. The hook reads the ref once, in a `useState` initializer, so a block's fade is decided at its own mount and never changes. The context lives in its own file because react-refresh lint forbids a component file from exporting a context.
- `Block` in `ChatThread.tsx` is a `div` that adds the class. The thread's blocks use it in place of their root `div`: user message, attachments, agent-turn line, text segment, answered question, steer message, tool group, thinking line, stopped / interrupted / error lines, turn footer, dividers. Approval and question cards get one wrapper `Block` each. Using a component per block matters: a turn's footer mounts when the turn ends, long after the turn itself, and still fades.
- `Block` spreads props before `className`, so attribute order in the markup stays as it was (existing tests match on `<div role="separator" …`).
- Text streaming into a block that is already visible does not fade per chunk. The block fades once, when it appears.

## Decisions

- **A custom rAF loop, not `scrollTo({ behavior: "smooth" })`.** Native smooth scroll restarts on every chunk, and its duration is not ours to set.
- **Gate on `thread.loaded`, not on mount.** Explained above. It is the one addition beyond the original spec.
- **150ms time constant.** Approved at tophat.
- **Resizes glide.** Approved at tophat.
- `Turn` is exported for the fade test, as `ToolTimeline` already was.

## Acceptance criteria

- [x] Following, new output glides the view to the bottom over about `--duration-normal`, and repeated chunks keep gliding: one retargeting loop in `useStickToBottom`; tophatted on the demo board.
- [x] Following, scrolling up during a glide stops following and the glide: a per-frame `nextStick` check plus `stop()` in the scroll handler; tophatted.
- [x] A glide never switches following off: the loop only moves down and updates `lastTop` with every write (`nextStick` tests cover `top >= prevTop`).
- [x] Opening, switching or reloading a chat lands at the bottom without a glide: the `loaded` gate; the reducer test covers `loaded`/`reset`. Resizing glides, approved at tophat.
- [x] Motion Off, or System default with OS reduced motion: following jumps (`motionAllowed()` in `pin`).
- [x] New blocks fade in under On and System default, not under Off: the `animate-stream-in` tier rule; `motion-css.test.ts` checks the Off rule.
- [x] Blocks already in the thread when it opens do not fade: `ChatThread.test.tsx` "shows a thread that is already there without fading".
- [x] Unit tests: `easeToward` (frame-rate independence, 1px minimum, landing, no overshoot), `parseDuration`, the `loaded` action, the CSS tier, and a live turn's blocks carrying `animate-stream-in`.
- [x] `make test` passes.

## Related

- xpo-f90406: the question card collapses line breaks in the question text. Found while tophatting, filed separately.
