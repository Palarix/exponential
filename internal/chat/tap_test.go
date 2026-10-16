package chat

// tap returns a channel of the events r publishes for chat id, settled and
// live, from now on: what the board's event stream would get. The channel
// is never closed.
func tap(r *Runner, id string) <-chan Event {
	ch := make(chan Event, 4096)
	r.mu.Lock()
	defer r.mu.Unlock()
	prev := r.cfg.OnEvent
	r.cfg.OnEvent = func(chatID string, e Event) {
		if prev != nil {
			prev(chatID, e)
		}
		if chatID == id {
			select {
			case ch <- e:
			default:
				panic("tap: the test does not read its events")
			}
		}
	}
	return ch
}
