import { request } from "../api/client";
import { createLive, type LiveClient } from "./live-client";

/** The tab's one event stream. useLive starts it; anything can follow a topic on it. */
export const live: LiveClient = createLive({
  create: () => new EventSource("/api/events"),
  post: (stream, change) =>
    request(`/api/events/${encodeURIComponent(stream)}/topics`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(change),
    }),
});
