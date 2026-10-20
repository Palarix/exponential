import { createContext, useContext, useState, type RefObject } from "react";

/**
 * Whether the chat thread is live: its history is in, so a block that mounts
 * now is new output. ChatThread provides it.
 */
export const StreamIn = createContext<RefObject<boolean>>({ current: false });

/** The fade for a block that streams in: decided once, when the block mounts. */
export function useStreamIn(): string {
  const live = useContext(StreamIn);
  const [fade] = useState(() => live.current);
  return fade ? "animate-stream-in" : "";
}
