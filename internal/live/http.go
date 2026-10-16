package live

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Defaults for a stream's timing.
const (
	DefaultWriteTimeout = 10 * time.Second
	DefaultKeepalive    = 25 * time.Second
)

// Stream is the HTTP side of a hub: GET opens an SSE stream, POST changes
// its topics.
//
// The wire format is named SSE events with JSON data:
//
//	event: hello   data: {"stream":"<id>"}
//	event: signal  data: {"topic":"issues"}
//	event: log     data: {"topic":"chat:<id>","data":<frame data>}
//	event: gone    data: {"topic":"chat:<id>"}
//
// There are no SSE ids: the client holds each log topic's position and
// sends it when it adds the topic, also after every hello.
type Stream struct {
	Hub          *Hub
	WriteTimeout time.Duration // per write; DefaultWriteTimeout when 0
	Keepalive    time.Duration // DefaultKeepalive when 0
}

// ServeEvents answers GET: it opens a subscriber, sends hello, and writes
// its events until the client goes, a write fails, or the hub closes.
func (st *Stream) ServeEvents(w http.ResponseWriter, r *http.Request) {
	sub, err := st.Hub.Subscribe()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "the event stream is shutting down")
		return
	}
	defer sub.Close()

	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")

	rc := http.NewResponseController(w)
	write := func(fn func(io.Writer)) bool {
		if err := rc.SetWriteDeadline(time.Now().Add(st.writeTimeout())); err != nil && !errors.Is(err, http.ErrNotSupported) {
			return false
		}
		fn(w)
		return rc.Flush() == nil
	}

	if !write(func(w io.Writer) { writeEvent(w, EventHello, map[string]string{"stream": sub.ID}) }) {
		return
	}
	keepalive := time.NewTicker(st.keepalive())
	defer keepalive.Stop()
	for {
		select {
		case <-sub.wake:
			outs := sub.Take()
			if len(outs) > 0 && !write(func(w io.Writer) {
				for _, o := range outs {
					writeOut(w, o)
				}
			}) {
				return
			}
		case <-keepalive.C:
			if !write(func(w io.Writer) { io.WriteString(w, ": keepalive\n\n") }) {
				return
			}
		case <-sub.done:
			return
		case <-r.Context().Done():
			return
		}
	}
}

// TopicChange is the body of a topics POST.
type TopicChange struct {
	Add []struct {
		Topic string `json:"topic"`
		After int    `json:"after"`
	} `json:"add"`
	Remove []string `json:"remove"`
}

// ServeTopics answers POST for the stream in the path value "stream". An
// unknown stream gets 404, and an unknown topic 400 with no change made.
func (st *Stream) ServeTopics(w http.ResponseWriter, r *http.Request) {
	sub := st.Hub.Lookup(r.PathValue("stream"))
	if sub == nil {
		writeError(w, http.StatusNotFound, "stream not found")
		return
	}
	var req TopicChange
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	for _, a := range req.Add {
		if _, _, ok := st.Hub.replayFor(a.Topic); !ok {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("unknown topic %q", a.Topic))
			return
		}
	}
	for _, topic := range req.Remove {
		sub.Remove(topic)
	}
	for _, a := range req.Add {
		_ = sub.Add(a.Topic, a.After)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (st *Stream) writeTimeout() time.Duration {
	if st.WriteTimeout > 0 {
		return st.WriteTimeout
	}
	return DefaultWriteTimeout
}

func (st *Stream) keepalive() time.Duration {
	if st.Keepalive > 0 {
		return st.Keepalive
	}
	return DefaultKeepalive
}

func writeOut(w io.Writer, o Out) {
	if o.Event == EventLog {
		writeEvent(w, o.Event, struct {
			Topic string `json:"topic"`
			Data  any    `json:"data"`
		}{o.Topic, o.Data})
		return
	}
	writeEvent(w, o.Event, map[string]string{"topic": o.Topic})
}

func writeEvent(w io.Writer, event string, data any) {
	b, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
