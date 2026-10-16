package live

import (
	"bufio"
	"io"
	"strings"
)

// ReadEvents parses an SSE stream and calls fn with each event's name
// ("message" when it has none) and data. Comments and ids are skipped. It
// returns when the stream ends, or with fn's first error.
func ReadEvents(r io.Reader, fn func(event, data string) error) error {
	br := bufio.NewReader(r)
	var event string
	var data []string
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		line = strings.TrimRight(line, "\r\n")
		switch {
		case line == "":
			if len(data) > 0 {
				if event == "" {
					event = "message"
				}
				if err := fn(event, strings.Join(data, "\n")); err != nil {
					return err
				}
			}
			event, data = "", nil
		case strings.HasPrefix(line, ":"):
		default:
			field, value, _ := strings.Cut(line, ":")
			value = strings.TrimPrefix(value, " ")
			switch field {
			case "event":
				event = value
			case "data":
				data = append(data, value)
			}
		}
	}
}
