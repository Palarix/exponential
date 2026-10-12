package exponential

import (
	"strings"
	"testing"
	"time"
)

// streamFixture is a trimmed `claude -p --output-format stream-json --verbose
// --include-partial-messages` transcript: init, a text block, a tool block,
// then the final result.
const streamFixture = `{"type":"system","subtype":"init","session_id":"sess-1","tools":["Read"]}
{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}}
{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Let me "}}}
{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"look."}}}
{"type":"stream_event","event":{"type":"content_block_stop","index":0}}
{"type":"stream_event","event":{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","name":"Read"}}}
{"type":"stream_event","event":{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"file_path\": \"/x/"}}}
{"type":"stream_event","event":{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"main.go\"}"}}}
{"type":"stream_event","event":{"type":"content_block_stop","index":1}}
not json at all
{"type":"system","subtype":"status","status":"requesting"}
{"type":"result","subtype":"success","result":"Done.","session_id":"sess-1","duration_ms":1500,"total_cost_usd":0.25,"usage":{"input_tokens":10,"cache_read_input_tokens":90,"output_tokens":5}}
`

func parseFixture(t *testing.T, fixture string) ([]StreamEvent, *ClaudeTurnResult) {
	t.Helper()
	var p ClaudeStreamParser
	var events []StreamEvent
	var result *ClaudeTurnResult
	for _, line := range strings.Split(fixture, "\n") {
		if r := p.Parse([]byte(line), func(ev StreamEvent) { events = append(events, ev) }); r != nil {
			result = r
		}
	}
	return events, result
}

func TestClaudeStreamParser_Events(t *testing.T) {
	events, _ := parseFixture(t, streamFixture)

	want := []StreamEvent{
		{Kind: StreamSession, Text: "sess-1"},
		{Kind: StreamText, Text: "Let me "},
		{Kind: StreamText, Text: "look."},
		{Kind: StreamToolDone}, // text block stop: no tool name
		{Kind: StreamToolStart, Text: "Read"},
		{Kind: StreamToolStart, Text: "Read", ToolArg: "/x/"},
		{Kind: StreamToolStart, Text: "Read", ToolArg: "/x/main.go"},
		{Kind: StreamToolDone, Text: "Read", ToolArg: "/x/main.go"},
		{Kind: StreamStatus, Text: "requesting"},
	}
	if len(events) != len(want) {
		t.Fatalf("got %d events, want %d: %+v", len(events), len(want), events)
	}
	for i := range want {
		if events[i] != want[i] {
			t.Errorf("event %d = %+v, want %+v", i, events[i], want[i])
		}
	}
}

func TestClaudeStreamParser_Result(t *testing.T) {
	_, r := parseFixture(t, streamFixture)
	if r == nil {
		t.Fatal("no result parsed")
	}
	if r.SessionID != "sess-1" || r.Output != "Done." || r.IsError {
		t.Errorf("unexpected result: %+v", r)
	}
	if r.Duration != 1500*time.Millisecond {
		t.Errorf("Duration = %v", r.Duration)
	}
	if r.TokensIn != 100 || r.TokensOut != 5 {
		t.Errorf("tokens = %d in / %d out", r.TokensIn, r.TokensOut)
	}
	if r.CostUSD != 0.25 {
		t.Errorf("CostUSD = %v", r.CostUSD)
	}
}

func TestClaudeStreamParser_ErrorResult(t *testing.T) {
	_, r := parseFixture(t, `{"type":"result","subtype":"error_during_execution","is_error":true,"result":"Not logged in","session_id":"s"}`)
	if r == nil || !r.IsError || r.Output != "Not logged in" {
		t.Fatalf("unexpected result: %+v", r)
	}
}

func TestClaudeStreamParser_ThinkingBlock(t *testing.T) {
	events, _ := parseFixture(t, `{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"thinking"}}}`)
	if len(events) != 1 || events[0].Kind != StreamThinking {
		t.Fatalf("events = %+v", events)
	}
}
