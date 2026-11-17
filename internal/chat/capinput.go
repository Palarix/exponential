package chat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// maxInputString is the most of one string value that an approval_requested
// event stores of the tool input. The approval card shows every argument in
// full, so more only bloats the log and the page.
const maxInputString = 64 << 10

// capInput returns the tool input JSON with each string value over
// maxInputString cut and marked. The JSON stays valid and keeps its key
// order, so the card still shows every argument. Input with nothing to cut
// comes back unchanged. Input that is not JSON is cut as one string.
func capInput(input string) string {
	if len(input) <= maxInputString {
		return input
	}
	if !json.Valid([]byte(input)) {
		return cutString(input)
	}
	dec := json.NewDecoder(strings.NewReader(input))
	dec.UseNumber()
	var buf bytes.Buffer
	cut, err := capValue(dec, &buf)
	if err != nil || !cut {
		return input
	}
	return buf.String()
}

// cutString cuts s to maxInputString bytes on a rune boundary and appends a
// marker with the full length.
func cutString(s string) string {
	n := maxInputString
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n] + fmt.Sprintf("… [cut: %d bytes total]", len(s))
}

// capValue copies the next JSON value from dec to buf, cutting long strings.
// It reports whether it cut any.
func capValue(dec *json.Decoder, buf *bytes.Buffer) (bool, error) {
	tok, err := dec.Token()
	if err != nil {
		return false, err
	}
	switch t := tok.(type) {
	case json.Delim:
		end := byte('}')
		if t == '[' {
			end = ']'
		}
		buf.WriteByte(byte(t))
		cut := false
		for i := 0; dec.More(); i++ {
			if i > 0 {
				buf.WriteByte(',')
			}
			if t == '{' {
				key, err := dec.Token()
				if err != nil {
					return false, err
				}
				writeJSONString(buf, key.(string))
				buf.WriteByte(':')
			}
			c, err := capValue(dec, buf)
			if err != nil {
				return false, err
			}
			cut = cut || c
		}
		if _, err := dec.Token(); err != nil {
			return false, err
		}
		buf.WriteByte(end)
		return cut, nil
	case string:
		if len(t) > maxInputString {
			writeJSONString(buf, cutString(t))
			return true, nil
		}
		writeJSONString(buf, t)
	case json.Number:
		buf.WriteString(t.String())
	case bool:
		fmt.Fprint(buf, t)
	case nil:
		buf.WriteString("null")
	}
	return false, nil
}

// writeJSONString writes s as a JSON string, without escaping HTML.
func writeJSONString(buf *bytes.Buffer, s string) {
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	buf.Truncate(buf.Len() - 1) // Encode appends a newline
}
