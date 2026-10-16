package live

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"time"
)

// Mirror keeps one connection to a remote board's event stream and
// notifies its signal topics on a local hub. In proxy mode, issues and
// drafts live on the remote, chats on this board, and the browser's one
// stream is this board's.
type Mirror struct {
	Hub    *Hub
	URL    string      // the remote's GET /api/events
	Header http.Header // sent with each connect, e.g. the bearer token
	Topics []string    // the remote's signal topics to mirror
	Client *http.Client

	MinBackoff time.Duration // 1s when 0
	MaxBackoff time.Duration // 30s when 0
}

// Run connects and reconnects until ctx ends. Each hello from the remote
// notifies every mirrored topic, since changes may have been missed while
// it was away.
func (m *Mirror) Run(ctx context.Context) {
	b := backoff{min: m.MinBackoff, max: m.MaxBackoff}
	for ctx.Err() == nil {
		if m.follow(ctx) {
			b.reset()
		}
		select {
		case <-time.After(b.next()):
		case <-ctx.Done():
		}
	}
}

// follow reads one remote stream until it ends. It reports whether the
// remote said hello.
func (m *Mirror) follow(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.URL, nil)
	if err != nil {
		return false
	}
	for k, v := range m.Header {
		req.Header[k] = v
	}
	req.Header.Set("Accept", "text/event-stream")
	client := m.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	connected := false
	_ = ReadEvents(resp.Body, func(event, data string) error {
		switch event {
		case EventHello:
			connected = true
			for _, topic := range m.Topics {
				m.Hub.Notify(topic)
			}
		case EventSignal:
			var d struct {
				Topic string `json:"topic"`
			}
			if json.Unmarshal([]byte(data), &d) == nil && slices.Contains(m.Topics, d.Topic) {
				m.Hub.Notify(d.Topic)
			}
		}
		return nil
	})
	return connected
}

// backoff is a retry delay that doubles from min up to max.
type backoff struct {
	min, max time.Duration
	cur      time.Duration
}

func (b *backoff) next() time.Duration {
	lo, hi := b.min, b.max
	if lo <= 0 {
		lo = time.Second
	}
	if hi <= 0 {
		hi = 30 * time.Second
	}
	if b.cur == 0 {
		b.cur = lo
	} else {
		b.cur = min(b.cur*2, hi)
	}
	return b.cur
}

func (b *backoff) reset() { b.cur = 0 }
