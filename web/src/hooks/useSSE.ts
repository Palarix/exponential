import { useEffect, useRef } from "react";

interface SSEOptions {
  onEvent: () => void;
  fallbackInterval?: number;
}

export function useSSE({ onEvent, fallbackInterval = 30000 }: SSEOptions) {
  const eventSourceRef = useRef<EventSource | null>(null);
  const fallbackRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    const connect = () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
      }

      const es = new EventSource("/api/events");
      eventSourceRef.current = es;

      es.onopen = () => {
        if (fallbackRef.current) {
          clearInterval(fallbackRef.current);
          fallbackRef.current = null;
        }
      };

      const handleEvent = () => onEvent();
      es.addEventListener("issue_created", handleEvent);
      es.addEventListener("issue_updated", handleEvent);
      es.addEventListener("issue_deleted", handleEvent);
      es.addEventListener("issue_merged", handleEvent);
      es.addEventListener("issue_commented", handleEvent);

      es.onerror = () => {
        es.close();
        eventSourceRef.current = null;
        if (!fallbackRef.current) {
          fallbackRef.current = setInterval(onEvent, fallbackInterval);
        }
        setTimeout(connect, 5000);
      };
    };

    connect();
    fallbackRef.current = setInterval(onEvent, fallbackInterval);

    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
      if (fallbackRef.current) {
        clearInterval(fallbackRef.current);
      }
    };
  }, [onEvent, fallbackInterval]);
}
