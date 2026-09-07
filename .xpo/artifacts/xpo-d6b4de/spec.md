# Spec: MCP tool — search across specs and walkthroughs

## What

Add a `rationale` MCP tool and CLI command that performs full-text search across all spec and walkthrough artifacts in `.xpo/artifacts/*/spec.md` and `walkthrough.md`. Returns ranked, contextualized fragments so an agent can discover prior design rationale without knowing which issue to look at.

## Why

Today the path to finding design context is: notice something in the code → `git blame` → `show` + `walkthrough read` — three hops requiring the caller to already know which file to blame. There's no way to ask "what design rationale exists around X?" and get relevant fragments back. This tool closes that gap.

## How

### Architecture

Follows the established pattern: `LocalTransport` method → `Transport` interface → `Client` delegation → MCP handler + CLI command.

### Search algorithm — BM25 with enhancements

The search uses a BM25-style ranking algorithm for search-engine-quality results without external dependencies.

#### Tokenization

- Split query and document content into words on non-alphanumeric boundaries.
- Lowercase all tokens.
- The query `"branch badge display"` becomes `["branch", "badge", "display"]`.

#### Scoring (per document)

Each document (a spec or walkthrough file) is scored independently:

1. **BM25 term scoring** — for each query term, compute:
   ```
   score += IDF(term) * (tf * (k1 + 1)) / (tf + k1 * (1 - b + b * dl/avgdl))
   ```
   Where:
   - `tf` = term frequency in the document
   - `IDF(term) = ln((N - n + 0.5) / (n + 0.5) + 1)` where N = total docs, n = docs containing the term
   - `k1 = 1.2` (term frequency saturation constant)
   - `b = 0.75` (document length normalization factor)
   - `dl` = document length in tokens
   - `avgdl` = average document length across corpus

   This gives us: multi-term matching, rare-term boosting (IDF), diminishing returns on repetition (saturation), and length normalization.

2. **Title/label boost** — if a query term appears in the issue's title, multiply the document's BM25 score by 1.5. If it matches a label, multiply by 1.3.

3. **Proximity bonus** — scan the document for windows where multiple query terms cluster together. For multi-term queries, if all query terms appear within a 20-token window, add a proximity bonus equal to 20% of the BM25 score.

4. **Recency tiebreaker** — when scores are within 1% of each other, rank more recently updated issues higher.

#### Fragment extraction

Rather than returning the paragraph around the first match, find the **best fragment**:

1. Slide a window of ~100 tokens across the document.
2. Score each window position by the number of distinct query terms it contains, weighted by IDF.
3. Select the highest-scoring window.
4. Expand to paragraph boundaries for clean display.
5. Trim to ~500 characters max, adding `...` ellipsis markers where trimmed.

#### Corpus construction

- Walk `.xpo/artifacts/*/` directories.
- For each issue directory, read `spec.md` and `walkthrough.md` (skip if missing or unreadable).
- Load the corresponding issue metadata (title, labels, status, updated_at) from the projected issue map.
- Each file is an independent document in the corpus (so an issue can produce two results — one from spec, one from walkthrough).

### MCP tool shape

```json
{
  "name": "rationale",
  "description": "Search across specs and walkthroughs for design rationale. Returns matching fragments ranked by relevance.",
  "parameters": {
    "query": "free-text search query (required)",
    "limit": "max results to return (default 5, max 20)"
  }
}
```

### Output shape

```json
{
  "results": [
    {
      "issue_id": "xpo-abc123",
      "title": "Branch badge display logic",
      "status": "DONE",
      "labels": ["feature"],
      "document": "spec",
      "fragment": "...the branch badge should display the short ref...",
      "score": 3.5,
      "updated_at": "2026-09-01T..."
    }
  ],
  "query": "branch badge",
  "total_matches": 12
}
```

### CLI shape

```
xpo rationale "branch badge display"
xpo rationale "merge strategy" --top 10
xpo rationale "merge strategy" -n 3
xpo rationale "merge strategy" --json
```

Flags:
- `--top N` / `-n N` — maximum results to return (default 5, max 20)
- `--json` — output results as JSON

### CLI output layout

Three-line layout per result:
- **Line 1:** Score (yellow) · Document type (blue=spec, gold=walkthrough), titlecased
- **Line 2:** Issue ID · Title · Labels · Status — standard project styling
- **Line 3:** Artifact file path (blue)
- Fragment in a rounded-border box below

### New code locations

| Layer | File | What |
|---|---|---|
| Search engine | `internal/exponential/rationale.go` | Tokenizer, BM25 scorer, fragment extractor, `SearchRationale` on `LocalTransport` |
| Transport interface | `internal/exponential/transport.go` | Add `SearchRationale` to `Transport` |
| Client delegation | `internal/exponential/client.go` | `Client.SearchRationale(query, limit)` |
| MCP handler | `internal/mcpserver/tools.go` | `rationaleIn`/`rationaleOut` types, `toolset.rationale` handler, registration |
| CLI renderer | `internal/ui/rationale.go` | Rich terminal output with lipgloss styling |
| CLI command | `cmd/exponential/rationale.go` | cobra command |
| Tests | `internal/mcpserver/tools_test.go` | Unit tests for the handler |
| Tests | `internal/exponential/rationale_test.go` | Unit tests for BM25 scoring, fragment extraction, tokenizer |
| Integration test | `internal/mcpserver/integration_test.go` | Add `"rationale"` to the `want` map |

### Edge cases

- **No artifacts exist** — return empty results, not an error.
- **Query is empty** — return an error ("query is required").
- **Artifact file unreadable** — skip it silently (may be corrupt or mid-write).
- **Very large artifacts** — cap content read at 1MB per file (matches existing `MaxArtifactContentLen`).
- **Single-term query** — BM25 works naturally; proximity bonus is skipped.
- **Issue metadata missing** — if an artifact directory has no matching issue in the projection (orphaned artifact), skip it.

## Acceptance Criteria

- [x] New `rationale` MCP tool registered and callable.
- [x] Returns issue ID, title, status, labels, document type (spec/walkthrough), and matching fragment.
- [x] Results ranked by BM25 scoring with title/label boost, proximity bonus, and recency tiebreaker.
- [x] Fragment extraction returns the densest cluster of matching terms, not just first match.
- [x] CLI command `xpo rationale "query"` with `--top` / `-n` and `--json` flags.
- [x] Rich CLI output with colored scores, document types, labels, status badges, and file paths.
- [x] Unit tests covering: BM25 scoring basics, fragment extraction, multi-term ranking, title boost, proximity bonus, limit parameter, empty query error, no-match case.
- [x] Integration test updated to include `rationale` in the tool list check.
