package exponential

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode"

	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/storage"
)

// RationaleResult is a single search hit from a spec or walkthrough.
type RationaleResult struct {
	IssueID   string   `json:"issue_id"`
	Title     string   `json:"title"`
	Status    string   `json:"status"`
	Labels    []string `json:"labels,omitempty"`
	Document  string   `json:"document"`
	Fragment  string   `json:"fragment"`
	Score     float64  `json:"score"`
	UpdatedAt string   `json:"updated_at"`
}

// RationaleSearchResult is the full response for a rationale search.
type RationaleSearchResult struct {
	Results      []RationaleResult `json:"results"`
	Query        string            `json:"query"`
	TotalMatches int               `json:"total_matches"`
}

// SearchRationale performs BM25-ranked full-text search across specs and walkthroughs.
func (t *LocalTransport) SearchRationale(query string, limit int) (*RationaleSearchResult, error) {
	if limit <= 0 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}

	queryTerms := wordTokens(query)
	if len(queryTerms) == 0 {
		return nil, fmt.Errorf("query is required")
	}

	issues, err := t.ListIssues(FilterOptions{All: true})
	if err != nil {
		return nil, err
	}
	issueMap := make(map[string]*model.Issue, len(issues))
	for _, iss := range issues {
		issueMap[iss.ID] = iss
	}

	type doc struct {
		issueID string
		docType string
		tokens  []string
		content string
		issue   *model.Issue
	}

	var docs []doc

	for issueID, issue := range issueMap {
		for _, fname := range []string{"spec.md", "walkthrough.md"} {
			content, err := storage.ReadArtifact(issueID, fname)
			if err != nil || content == "" {
				continue
			}
			if len(content) > MaxArtifactContentLen {
				content = content[:MaxArtifactContentLen]
			}
			docType := strings.TrimSuffix(fname, ".md")
			docs = append(docs, doc{
				issueID: issueID,
				docType: docType,
				tokens:  wordTokens(content),
				content: content,
				issue:   issue,
			})
		}
	}

	if len(docs) == 0 {
		return &RationaleSearchResult{Query: query}, nil
	}

	// Compute IDF for each query term across the corpus.
	docCount := float64(len(docs))
	termDocFreq := make(map[string]int)
	for _, d := range docs {
		seen := make(map[string]bool)
		for _, tok := range d.tokens {
			if !seen[tok] {
				termDocFreq[tok]++
				seen[tok] = true
			}
		}
	}

	idf := make(map[string]float64, len(queryTerms))
	for _, term := range queryTerms {
		n := float64(termDocFreq[term])
		idf[term] = math.Log((docCount-n+0.5)/(n+0.5) + 1)
	}

	// Average document length.
	var totalLen int
	for _, d := range docs {
		totalLen += len(d.tokens)
	}
	avgDL := float64(totalLen) / docCount

	const (
		k1 = 1.2
		b  = 0.75
	)

	type scored struct {
		doc   doc
		score float64
	}

	var results []scored
	for _, d := range docs {
		tf := make(map[string]int)
		for _, tok := range d.tokens {
			tf[tok]++
		}

		dl := float64(len(d.tokens))
		var score float64
		hasMatch := false
		for _, term := range queryTerms {
			termTF := float64(tf[term])
			if termTF == 0 {
				continue
			}
			hasMatch = true
			score += idf[term] * (termTF * (k1 + 1)) / (termTF + k1*(1-b+b*dl/avgDL))
		}

		if !hasMatch {
			continue
		}

		// Title/label boost.
		titleTokens := wordTokens(d.issue.Title)
		labelTokens := wordTokens(strings.Join(d.issue.Labels, " "))
		for _, term := range queryTerms {
			if sliceContains(titleTokens, term) {
				score *= 1.5
			}
			if sliceContains(labelTokens, term) {
				score *= 1.3
			}
		}

		// Proximity bonus for multi-term queries.
		if len(queryTerms) > 1 {
			score += proximityBonus(d.tokens, queryTerms) * score * 0.2
		}

		results = append(results, scored{doc: d, score: score})
	}

	totalMatches := len(results)

	// Sort by score descending, then by recency.
	sort.Slice(results, func(i, j int) bool {
		si, sj := results[i].score, results[j].score
		if math.Abs(si-sj) > si*0.01 {
			return si > sj
		}
		return results[i].doc.issue.UpdatedAt.After(results[j].doc.issue.UpdatedAt)
	})

	if len(results) > limit {
		results = results[:limit]
	}

	out := &RationaleSearchResult{
		Query:        query,
		TotalMatches: totalMatches,
		Results:      make([]RationaleResult, len(results)),
	}
	for i, r := range results {
		out.Results[i] = RationaleResult{
			IssueID:   r.doc.issueID,
			Title:     r.doc.issue.Title,
			Status:    string(r.doc.issue.Status),
			Labels:    r.doc.issue.Labels,
			Document:  r.doc.docType,
			Fragment:  bestFragment(r.doc.content, queryTerms, idf),
			Score:     math.Round(r.score*100) / 100,
			UpdatedAt: r.doc.issue.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return out, nil
}

// wordTokens splits text into lowercase word tokens on non-alphanumeric boundaries.
func wordTokens(s string) []string {
	var tokens []string
	var buf strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			buf.WriteRune(unicode.ToLower(r))
		} else if buf.Len() > 0 {
			tokens = append(tokens, buf.String())
			buf.Reset()
		}
	}
	if buf.Len() > 0 {
		tokens = append(tokens, buf.String())
	}
	return tokens
}

func sliceContains(tokens []string, term string) bool {
	for _, t := range tokens {
		if t == term {
			return true
		}
	}
	return false
}

// proximityBonus returns 1.0 if all query terms appear within a 20-token window, 0.0 otherwise.
func proximityBonus(docTokens, queryTerms []string) float64 {
	if len(docTokens) == 0 {
		return 0
	}
	termSet := make(map[string]bool, len(queryTerms))
	for _, t := range queryTerms {
		termSet[t] = true
	}

	const windowSize = 20
	for i := range docTokens {
		end := i + windowSize
		if end > len(docTokens) {
			end = len(docTokens)
		}
		found := make(map[string]bool)
		for _, tok := range docTokens[i:end] {
			if termSet[tok] {
				found[tok] = true
			}
		}
		if len(found) == len(termSet) {
			return 1.0
		}
	}
	return 0
}

// bestFragment finds the ~100-token window with the highest density of query terms,
// then returns it as a trimmed string.
func bestFragment(content string, queryTerms []string, idf map[string]float64) string {
	tokens := wordTokens(content)
	if len(tokens) == 0 {
		return ""
	}

	termSet := make(map[string]bool, len(queryTerms))
	for _, t := range queryTerms {
		termSet[t] = true
	}

	const windowTokens = 100
	bestStart := 0
	bestScore := -1.0

	tokCount := len(tokens)
	for i := 0; i < tokCount; i++ {
		end := i + windowTokens
		if end > tokCount {
			end = tokCount
		}
		var score float64
		for _, tok := range tokens[i:end] {
			if termSet[tok] {
				score += idf[tok]
			}
		}
		if score > bestScore {
			bestScore = score
			bestStart = i
		}
		if end == tokCount {
			break
		}
	}

	// Map token positions back to character positions in the original content.
	charPositions := tokenCharPositions(content)
	startChar := 0
	endChar := len(content)
	if bestStart < len(charPositions) {
		startChar = charPositions[bestStart][0]
	}
	endIdx := bestStart + windowTokens
	if endIdx >= len(charPositions) {
		endIdx = len(charPositions) - 1
	}
	if endIdx >= 0 && endIdx < len(charPositions) {
		endChar = charPositions[endIdx][1]
	}

	// Expand to paragraph boundaries for cleaner display.
	fragment := expandToParagraphs(content, startChar, endChar)

	const maxLen = 500
	if len(fragment) > maxLen {
		fragment = fragment[:maxLen]
		if idx := strings.LastIndexByte(fragment, ' '); idx > maxLen/2 {
			fragment = fragment[:idx]
		}
		fragment += "..."
	}

	prefix := ""
	if startChar > 0 {
		prefix = "..."
	}
	return prefix + strings.TrimSpace(fragment)
}

// tokenCharPositions returns [start, end) byte offsets for each token in the content.
func tokenCharPositions(content string) [][2]int {
	var positions [][2]int
	start := -1
	for i, r := range content {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if start < 0 {
				start = i
			}
		} else if start >= 0 {
			positions = append(positions, [2]int{start, i})
			start = -1
		}
	}
	if start >= 0 {
		positions = append(positions, [2]int{start, len(content)})
	}
	return positions
}

// expandToParagraphs extends a character range to the nearest paragraph boundaries.
func expandToParagraphs(content string, startChar, endChar int) string {
	paraStart := startChar
	if startChar > 0 {
		if idx := strings.LastIndex(content[:startChar], "\n\n"); idx >= 0 && startChar-idx < 200 {
			paraStart = idx + 2
		}
	}

	paraEnd := endChar
	if endChar < len(content) {
		rest := content[endChar:]
		if idx := strings.Index(rest, "\n\n"); idx >= 0 && idx < 200 {
			paraEnd = endChar + idx
		}
	}

	return content[paraStart:paraEnd]
}
