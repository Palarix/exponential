package server

import (
	"bufio"
	"io"
	"net/http"
	"strings"
	"time"
)

// handleSSERelay answers /api/events in proxy mode. Issue events come from
// the remote server, but agent chats live on this board, so the browser
// gets one stream with both: the remote stream's events, verbatim, and this
// board's own hub events. When the remote stream ends, the response ends
// too, and the browser reconnects.
func (s *Server) handleSSERelay(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, strings.TrimSuffix(s.ProxyURL, "/")+"/api/events", nil)
	if err != nil {
		respondError(w, http.StatusBadGateway, "invalid remote server URL")
		return
	}
	req.Header.Set("Accept", "text/event-stream")
	if s.ProxyToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.ProxyToken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		respondError(w, http.StatusBadGateway, "remote server unreachable — check that the server is running and the URL is correct")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respondError(w, http.StatusBadGateway, "remote server refused the event stream: "+resp.Status)
		return
	}

	// Subscribe before the first flush, so no local event is missed once
	// the browser counts the stream as open.
	ch := s.SSEHub.subscribe()
	defer s.SSEHub.unsubscribe(ch)

	setSSEHeaders(w)
	flusher.Flush()

	remote := make(chan string)
	go readSSEBlocks(resp.Body, remote, r.Context().Done())

	keepalive := time.NewTicker(30 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case block, ok := <-remote:
			if !ok {
				return
			}
			io.WriteString(w, block)
			flusher.Flush()

		case evt, ok := <-ch:
			if !ok {
				return
			}
			writeSSEEvent(w, evt)
			flusher.Flush()

		case <-keepalive.C:
			io.WriteString(w, ": keepalive\n\n")
			flusher.Flush()

		case <-r.Context().Done():
			return
		}
	}
}

// readSSEBlocks sends each event block of an SSE stream, blank line
// included, on out, so a block is written whole. It closes out when the
// stream ends; done stops it early.
func readSSEBlocks(body io.Reader, out chan<- string, done <-chan struct{}) {
	defer close(out)
	br := bufio.NewReader(body)
	var block strings.Builder
	for {
		line, err := br.ReadString('\n')
		block.WriteString(line)
		if err != nil {
			return
		}
		if strings.TrimRight(line, "\r\n") != "" {
			continue
		}
		select {
		case out <- block.String():
		case <-done:
			return
		}
		block.Reset()
	}
}
