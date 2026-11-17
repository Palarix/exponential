package chat

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCapInput_CutsLongStringValues(t *testing.T) {
	long := strings.Repeat("é", maxInputString) // 2 bytes per rune: twice the cap
	in, _ := json.Marshal(map[string]any{"file_path": "a.txt", "content": long, "nested": map[string]any{"list": []any{long, 3}}})

	out := capInput(string(in))
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("capInput returned invalid JSON: %v", err)
	}
	if got["file_path"] != "a.txt" {
		t.Errorf("file_path = %v", got["file_path"])
	}
	content := got["content"].(string)
	marker := "… [cut: 131072 bytes total]"
	if !strings.HasSuffix(content, marker) || len(content) > maxInputString+len(marker) || !utf8.ValidString(content) {
		t.Errorf("content: len %d, suffix %q", len(content), content[len(content)-40:])
	}
	list := got["nested"].(map[string]any)["list"].([]any)
	if s := list[0].(string); !strings.HasSuffix(s, marker) {
		t.Errorf("nested string not cut: len %d", len(s))
	}
	if list[1] != float64(3) {
		t.Errorf("nested number = %v", list[1])
	}
}

func TestCapInput_KeepsSmallInputByteForByte(t *testing.T) {
	in := `{"b": 1,  "a":"x"}`
	if out := capInput(in); out != in {
		t.Errorf("capInput(%q) = %q", in, out)
	}
	if out := capInput(""); out != "" {
		t.Errorf("capInput(\"\") = %q", out)
	}
}

func TestCapInput_CutsLongInvalidJSONAsOneString(t *testing.T) {
	in := "{" + strings.Repeat("x", maxInputString*2)
	out := capInput(in)
	if !strings.HasPrefix(out, "{xxx") || !strings.HasSuffix(out, "… [cut: 131073 bytes total]") || len(out) > maxInputString+40 {
		t.Errorf("capInput(invalid) len %d", len(out))
	}
}
