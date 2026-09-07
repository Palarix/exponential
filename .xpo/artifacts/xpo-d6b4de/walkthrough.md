# Walkthrough: MCP tool — search across specs and walkthroughs

## What changed

Added a `rationale` MCP tool and `xpo rationale` CLI command that performs BM25-ranked full-text search across all spec and walkthrough artifacts. An agent (or human) can now ask "what design rationale exists around X?" and get ranked, contextualized fragments back — no need to know which issue to look at first.

## How it works

### Search engine (`internal/exponential/rationale.go`)

The core is a self-contained BM25 implementation with no external dependencies:

1. **Corpus construction** — walks `.xpo/artifacts/*/`, reads `spec.md` and `walkthrough.md` from each issue directory. Each file is an independent document (so one issue can produce two results). Skips orphaned directories with no matching issue in the projection.

2. **Tokenization** — `wordTokens()` splits text into lowercase tokens on non-alphanumeric boundaries. Named differently from the existing `tokenize()` in `duplicates.go` (which returns `map[string]bool` for set operations) because this one returns `[]string` to preserve order and frequency.

3. **BM25 scoring** — for each query term in each document:
   ```
   score += IDF(term) * (tf * (k1+1)) / (tf + k1*(1 - b + b*dl/avgdl))
   ```
   This gives multi-term matching, IDF weighting (rare terms score higher), term frequency saturation (10th mention adds less than 2nd), and document length normalization (shorter docs with same term count rank higher).

4. **Title/label boost** — if a query term appears in the issue's title (1.5x) or labels (1.3x), the document's score is multiplied. This is why searching "ghost items filtered view" surfaces `xpo-b109e5` ("Completed ghost items appear in filtered views") at the top.

5. **Proximity bonus** — for multi-term queries, scans 20-token windows. If all query terms cluster within one window, adds 20% to the BM25 score. Rewards documents where terms appear near each other.

6. **Recency tiebreaker** — when scores are within 1% of each other, more recently updated issues rank higher.

7. **Fragment extraction** — slides a 100-token window across the document, scoring each position by IDF-weighted term density. Picks the highest-scoring window, expands to paragraph boundaries, trims to 500 chars. This reliably picks the most relevant section rather than just the first match.

### Transport/Client layer

Follows the established pattern:
- `SearchRationale(query, limit)` on `LocalTransport` (the implementation)
- Added to `Transport` interface
- `Client.SearchRationale()` delegation with `syncLocal()`
- `RemoteTransport` returns `ErrLocalOnly` (artifact search is local-only)

### MCP handler (`internal/mcpserver/tools.go`)

Registered as `rationale` tool with `query` (required) and `limit` (optional, default 5, max 20) parameters. The handler converts `exponential.RationaleSearchResult` to the MCP output types. Returns structured output with issue metadata, document type, fragment, and score.

### CLI command (`cmd/exponential/rationale.go`)

```
xpo rationale "branch badge display"
xpo rationale "merge strategy" --top 10
xpo rationale "merge strategy" --json
```

Flags: `--top N` / `-n N` (default 5, max 20), `--json` (machine-readable output).

### Rich CLI output (`internal/ui/rationale.go`)

Three-line layout per result:
- **Line 1:** Score (yellow) · Document type (blue for spec, gold for walkthrough)
- **Line 2:** Issue ID · Title · Labels · Status — all with standard project styling
- **Line 3:** Artifact file path (blue, clickable in most terminals)
- **Fragment:** In a rounded-border box, muted text, word-wrapped

Uses the project's lipgloss styling system — `StatusStyle`, `FormatLabelsBadge`, `StatusIcon`, etc. The `ui.RationaleData`/`ui.RationaleHit` types avoid an import cycle between `ui` and `exponential`.

## Key decisions

- **BM25 over substring matching** — substring matching produces noisy results (no multi-term support, no term weighting, no length normalization). BM25 is ~60 lines of Go with no dependencies and produces search-engine-quality ranking.
- **No persistent index** — tokenizes on the fly per search. With dozens of artifact files at ≤1MB each, this completes in milliseconds. An inverted index would add complexity for negligible gain at this scale.
- **Spec and walkthrough as separate results** — they serve different purposes (design intent vs implementation record), so both appearing for the same issue is useful, not redundant.
- **`wordTokens` vs `tokenize`** — the existing `tokenize` in `duplicates.go` splits on whitespace and returns a set. BM25 needs ordered tokens with frequency, so a separate function that splits on non-alphanumeric boundaries was the clean choice.

## Files changed

| File | What |
|---|---|
| `internal/exponential/rationale.go` | New — BM25 search engine, tokenizer, fragment extraction |
| `internal/exponential/rationale_test.go` | New — unit tests for tokenizer, proximity, fragments, BM25 properties |
| `internal/exponential/transport.go` | Added `SearchRationale` to `Transport` interface |
| `internal/exponential/client.go` | Added `Client.SearchRationale` delegation |
| `internal/exponential/remote_transport.go` | Added `ErrLocalOnly` stub |
| `internal/mcpserver/tools.go` | Rationale types, handler, registration |
| `internal/mcpserver/tools_test.go` | 6 handler tests |
| `internal/mcpserver/integration_test.go` | Added `rationale` to tool list check |
| `internal/ui/rationale.go` | New — rich CLI renderer |
| `cmd/exponential/rationale.go` | New — cobra command |
