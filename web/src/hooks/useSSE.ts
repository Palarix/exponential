import { useEffect, useRef } from "react";

interface SSEOptions {
  onEvent: () => void;
}

export function useSSE({ onEvent }: SSEOptions) {
  const onEventRef = useRef(onEvent);
  useEffect(() => {
    onEventRef.current = onEvent;
  }, [onEvent]);

  useEffect(() => {
    let es: EventSource | null = null;
    let reconnectTimer: ReturnType<typeof setTimeout>;

    const connect = () => {
      es = new EventSource("/api/events");

      const handleEvent = () => onEventRef.current();
      es.addEventListener("issue_created", handleEvent);
      es.addEventListener("issue_updated", handleEvent);
      es.addEventListener("issue_deleted", handleEvent);
      es.addEventListener("issue_merged", handleEvent);
      es.addEventListener("issue_commented", handleEvent);
      es.addEventListener("refresh", handleEvent);

      es.onerror = () => {
        es?.close();
        es = null;
        reconnectTimer = setTimeout(connect, 5000);
      };
    };

    connect();

    return () => {
      clearTimeout(reconnectTimer);
      es?.close();
    };
  }, []);
}
