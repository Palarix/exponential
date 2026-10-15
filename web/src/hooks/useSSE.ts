import { useEffect, useRef } from "react";
import { connectEvents } from "./sse-utils";

interface SSEOptions {
  /** An issue event arrived. */
  onEvent: () => void;
  /** A chat changed, or the stream (re)opened and may have missed changes. */
  onChatEvent: () => void;
}

export function useSSE({ onEvent, onChatEvent }: SSEOptions) {
  const onEventRef = useRef(onEvent);
  const onChatEventRef = useRef(onChatEvent);
  useEffect(() => {
    onEventRef.current = onEvent;
    onChatEventRef.current = onChatEvent;
  }, [onEvent, onChatEvent]);

  useEffect(() => connectEvents({
    create: () => new EventSource("/api/events"),
    onEvent: () => onEventRef.current(),
    onChatEvent: () => onChatEventRef.current(),
  }), []);
}
