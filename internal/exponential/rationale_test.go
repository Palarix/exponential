package exponential

import (
	"math"
	"testing"
)

func TestWordTokens(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"hello world", []string{"hello", "world"}},
		{"BM25-scoring", []string{"bm25", "scoring"}},
		{"branch_badge.display", []string{"branch", "badge", "display"}},
		{"", nil},
		{"   ", nil},
		{"UPPER CASE", []string{"upper", "case"}},
		{"code`blocks`here", []string{"code", "blocks", "here"}},
	}
	for _, tt := range tests {
		got := wordTokens(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("wordTokens(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("wordTokens(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestSliceContains(t *testing.T) {
	tokens := []string{"alpha", "beta", "gamma"}
	if !sliceContains(tokens, "beta") {
		t.Error("expected true for 'beta'")
	}
	if sliceContains(tokens, "delta") {
		t.Error("expected false for 'delta'")
	}
}

func TestProximityBonus(t *testing.T) {
	// All terms within 20 tokens.
	tokens := wordTokens("the branch badge is displayed on each issue card in the list view")
	if got := proximityBonus(tokens, []string{"branch", "badge"}); got != 1.0 {
		t.Errorf("expected 1.0 for nearby terms, got %f", got)
	}

	// Terms far apart — build a 50-token gap.
	far := make([]string, 0, 55)
	far = append(far, "branch")
	for i := 0; i < 50; i++ {
		far = append(far, "filler")
	}
	far = append(far, "badge")
	if got := proximityBonus(far, []string{"branch", "badge"}); got != 0.0 {
		t.Errorf("expected 0.0 for distant terms, got %f", got)
	}

	// Single-term query: not called, but should return 1.0 if the term exists.
	if got := proximityBonus(tokens, []string{"branch"}); got != 1.0 {
		t.Errorf("expected 1.0 for single-term, got %f", got)
	}

	// Empty doc.
	if got := proximityBonus(nil, []string{"branch"}); got != 0.0 {
		t.Errorf("expected 0.0 for empty doc, got %f", got)
	}
}

func TestBestFragment(t *testing.T) {
	content := "# Introduction\n\nThis document covers the branch badge display logic.\n\n## Details\n\nThe badge shows the git ref name shortened to 8 chars."
	idf := map[string]float64{
		"branch":  2.0,
		"badge":   2.5,
		"display": 1.0,
	}
	fragment := bestFragment(content, []string{"branch", "badge", "display"}, idf)
	if fragment == "" {
		t.Fatal("expected non-empty fragment")
	}
	if len(fragment) > 600 {
		t.Errorf("fragment too long: %d chars", len(fragment))
	}
}

func TestBestFragment_Empty(t *testing.T) {
	fragment := bestFragment("", []string{"term"}, map[string]float64{"term": 1.0})
	if fragment != "" {
		t.Errorf("expected empty fragment for empty content, got %q", fragment)
	}
}

func TestTokenCharPositions(t *testing.T) {
	content := "hello world!"
	positions := tokenCharPositions(content)
	if len(positions) != 2 {
		t.Fatalf("expected 2 positions, got %d", len(positions))
	}
	if positions[0] != [2]int{0, 5} {
		t.Errorf("positions[0] = %v, want [0, 5]", positions[0])
	}
	if positions[1] != [2]int{6, 11} {
		t.Errorf("positions[1] = %v, want [6, 11]", positions[1])
	}
}

func TestExpandToParagraphs(t *testing.T) {
	content := "Paragraph one.\n\nParagraph two has target text.\n\nParagraph three."
	// Target is within paragraph two.
	result := expandToParagraphs(content, 17, 40)
	if result != "Paragraph two has target text." {
		t.Errorf("expandToParagraphs = %q, want %q", result, "Paragraph two has target text.")
	}
}

func TestExpandToParagraphs_StartOfContent(t *testing.T) {
	content := "First paragraph.\n\nSecond paragraph."
	result := expandToParagraphs(content, 0, 10)
	if result != "First paragraph." {
		t.Errorf("expandToParagraphs = %q, want %q", result, "First paragraph.")
	}
}

func TestBM25ScoreComponents(t *testing.T) {
	const k1 = 1.2
	const b = 0.75
	avgDL := 50.0

	bm25 := func(tf, dl float64, termIDF float64) float64 {
		return termIDF * (tf * (k1 + 1)) / (tf + k1*(1-b+b*dl/avgDL))
	}

	// Doc A: both terms present once, length 50.
	scoreA := bm25(1, 50, 1.5) + bm25(1, 50, 1.5)
	// Doc B: only one term present once, length 50.
	scoreB := bm25(1, 50, 1.5)

	if scoreA <= scoreB {
		t.Errorf("doc with both terms should score higher: %f <= %f", scoreA, scoreB)
	}

	// Doc C: one term present 5 times, length 50 — saturation.
	scoreC := bm25(5, 50, 1.5)
	// Should be higher than 1 occurrence but less than 5x.
	if scoreC <= scoreB {
		t.Errorf("more occurrences should score higher: %f <= %f", scoreC, scoreB)
	}
	if scoreC >= scoreB*5 {
		t.Errorf("saturation should prevent linear scaling: %f >= %f", scoreC, scoreB*5)
	}

	// Length normalization: longer doc scores lower for same tf.
	scoreLong := bm25(1, 200, 1.5)
	scoreShort := bm25(1, 20, 1.5)
	if scoreLong >= scoreShort {
		t.Errorf("longer doc should score lower: %f >= %f", scoreLong, scoreShort)
	}

	// IDF effect: rare term scores higher.
	scoreRare := bm25(1, 50, 3.0)
	scoreCommon := bm25(1, 50, 0.5)
	if scoreRare <= scoreCommon {
		t.Errorf("rare term should score higher: %f <= %f", scoreRare, scoreCommon)
	}

	// Sanity: all scores are positive and finite.
	for _, s := range []float64{scoreA, scoreB, scoreC, scoreLong, scoreShort, scoreRare, scoreCommon} {
		if math.IsNaN(s) || math.IsInf(s, 0) || s < 0 {
			t.Errorf("invalid score: %f", s)
		}
	}
}
