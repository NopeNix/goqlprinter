import { useEffect, useRef, useCallback } from "react";

type SSEEventHandler = (data: unknown) => void;

interface UseSSEOptions {
  /** Map of event name → handler */
  events: Record<string, SSEEventHandler>;
  /** Whether to connect (default true) */
  enabled?: boolean;
}

/**
 * Hook that maintains an EventSource (SSE) connection to /api/events.
 * Automatically reconnects on errors with exponential backoff.
 */
export function useSSE({ events, enabled = true }: UseSSEOptions) {
  const eventsRef = useRef(events);
  eventsRef.current = events;

  const forceReconnect = useCallback(() => {
    // Trigger reconnect by dispatching a custom event
    window.dispatchEvent(new CustomEvent("sse-reconnect"));
  }, []);

  useEffect(() => {
    if (!enabled) return;

    let es: EventSource | null = null;
    let retryDelay = 1000;
    let retryTimeout: ReturnType<typeof setTimeout> | null = null;
    let disposed = false;

    function connect() {
      if (disposed) return;

      es = new EventSource("/api/events");

      es.onopen = () => {
        retryDelay = 1000; // reset backoff on success
      };

      // Listen for each registered event type
      const handler = (e: MessageEvent) => {
        try {
          const data = JSON.parse(e.data);
          const fn = eventsRef.current[e.type];
          if (fn) fn(data);
        } catch {
          // ignore parse errors
        }
      };

      for (const eventName of Object.keys(eventsRef.current)) {
        es.addEventListener(eventName, handler);
      }

      es.onerror = () => {
        es?.close();
        es = null;
        if (!disposed) {
          retryTimeout = setTimeout(connect, retryDelay);
          retryDelay = Math.min(retryDelay * 2, 30000);
        }
      };
    }

    connect();

    // Listen for forced reconnects
    const onReconnect = () => {
      es?.close();
      es = null;
      if (retryTimeout) clearTimeout(retryTimeout);
      retryDelay = 1000;
      connect();
    };
    window.addEventListener("sse-reconnect", onReconnect);

    return () => {
      disposed = true;
      window.removeEventListener("sse-reconnect", onReconnect);
      if (retryTimeout) clearTimeout(retryTimeout);
      es?.close();
    };
  }, [enabled]);

  return { forceReconnect };
}
