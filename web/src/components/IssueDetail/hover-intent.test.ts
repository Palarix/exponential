import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { createHoverIntent } from "./hover-intent";

describe("createHoverIntent", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it("opens after the delay", () => {
    const onChange = vi.fn();
    const intent = createHoverIntent(250, onChange);
    intent.enter();
    vi.advanceTimersByTime(249);
    expect(onChange).not.toHaveBeenCalled();
    vi.advanceTimersByTime(1);
    expect(onChange).toHaveBeenCalledWith(true);
  });

  it("does not open when left before the delay", () => {
    const onChange = vi.fn();
    const intent = createHoverIntent(250, onChange);
    intent.enter();
    vi.advanceTimersByTime(200);
    intent.leave();
    vi.advanceTimersByTime(500);
    expect(onChange).not.toHaveBeenCalledWith(true);
  });

  it("closes immediately on leave once open", () => {
    const onChange = vi.fn();
    const intent = createHoverIntent(250, onChange);
    intent.enter();
    vi.advanceTimersByTime(250);
    intent.leave();
    expect(onChange).toHaveBeenLastCalledWith(false);
  });

  it("cancel stops a pending open", () => {
    const onChange = vi.fn();
    const intent = createHoverIntent(250, onChange);
    intent.enter();
    intent.cancel();
    vi.advanceTimersByTime(500);
    expect(onChange).not.toHaveBeenCalledWith(true);
  });
});
