import { useCallback, useEffect, useRef, useState } from "react";
import { SPIN_PERIOD_MS, spinHoldMs } from "../utils/spinner";

/**
 * Whether a spinner for `busy` work should be visible: it appears once the
 * work has run for `delay` ms (0 = always), and once visible it keeps turning
 * until a whole rotation completes, so fast work still reads as a spin.
 */
export function useSpinner(busy: boolean, delay: number, period = SPIN_PERIOD_MS): boolean {
  const [visible, setVisible] = useState(false);
  const busySince = useRef<number | null>(null);
  const shownAt = useRef<number | null>(null);
  useEffect(() => {
    const show = () => {
      if (shownAt.current === null) shownAt.current = performance.now();
      setVisible(true);
    };

    if (busy) {
      busySince.current = performance.now();
      const timer = setTimeout(show, delay);
      return () => clearTimeout(timer);
    }

    if (busySince.current === null) return;
    const now = performance.now();
    const hold = spinHoldMs({
      busyMs: now - busySince.current,
      visibleMs: shownAt.current === null ? null : now - shownAt.current,
      delay,
      period,
    });
    busySince.current = null;
    if (hold === null) return;
    const timers = [
      setTimeout(show, 0),
      setTimeout(() => {
        shownAt.current = null;
        setVisible(false);
      }, hold),
    ];
    return () => timers.forEach(clearTimeout);
  }, [busy, delay, period]);

  return visible;
}

/** Resolves after the next frame has painted, with that frame's timestamp. */
function afterNextPaint(): Promise<number> {
  return new Promise((resolve) => requestAnimationFrame((frame) => setTimeout(() => resolve(frame), 0)));
}

export type SpinResult<T> = { ok: true; value: T } | { ok: false; error: unknown };

/**
 * A spinner for user-triggered work. `track(run)` starts spinning in the
 * click's own render and only starts `run` once that first spinning frame has
 * painted, so a heavy re-render caused by the work can't hide the feedback.
 * After the work settles it keeps turning until a whole rotation completes,
 * then reports the last run's result to `onSettled`. Overlapping calls extend
 * one continuous spin.
 */
export function useSpinOnce<T>(onSettled?: (result: SpinResult<T>) => void, period = SPIN_PERIOD_MS) {
  const [spinning, setSpinning] = useState(false);
  const startedAt = useRef<number | null>(null);
  const pending = useRef(0);
  const lastResult = useRef<SpinResult<T> | null>(null);
  const onSettledRef = useRef(onSettled);
  useEffect(() => {
    onSettledRef.current = onSettled;
  }, [onSettled]);

  const track = useCallback(
    async (run: () => Promise<T>): Promise<void> => {
      pending.current++;
      const first = startedAt.current === null;
      if (first) setSpinning(true);
      try {
        const frame = await afterNextPaint();
        // The CSS animation starts with the first painted frame, so measure rotations from it.
        if (first && startedAt.current === null) startedAt.current = frame;
        lastResult.current = { ok: true, value: await run() };
      } catch (error) {
        lastResult.current = { ok: false, error };
      } finally {
        pending.current--;
        if (pending.current === 0 && startedAt.current !== null) {
          const shownFor = performance.now() - startedAt.current;
          const hold = spinHoldMs({ busyMs: shownFor, visibleMs: shownFor, delay: 0, period }) ?? 0;
          setTimeout(() => {
            if (pending.current > 0) return;
            startedAt.current = null;
            setSpinning(false);
            if (lastResult.current) onSettledRef.current?.(lastResult.current);
          }, hold);
        }
      }
    },
    [period],
  );

  return { spinning, track };
}
