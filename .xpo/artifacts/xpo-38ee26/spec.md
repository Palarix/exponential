# Spec: Benchmark ref-per-issue storage model

## What

A standalone Go benchmark that creates a temporary git repo and exercises the
"one ref per issue" storage model using git plumbing commands. Each issue lives
at `refs/xpo/issues/<id>` pointing to a commit whose tree contains:

```
issue.json             # materialized state + event log
artifacts/
  spec.md              # ~500 bytes of placeholder text
  walkthrough.md       # ~300 bytes of placeholder text
```

## Why

The current spike (xpo-dbb7fa) stores all issues in a single `issues.db` blob,
rewritten on every event. We brainstormed a ref-per-issue model that could give
O(1) single-issue reads/writes and natural structural sharing. Before committing
to this architecture we need hard numbers on:

- Write throughput for bulk issue creation
- Read latency for single issue and full board scan
- Overhead of N refs (packed vs loose)
- Feasibility of watching for changes

## How

### Storage engine (`engine.go`)

A minimal `RefStore` that wraps git plumbing:

- `Init(dir string)` — `git init --bare`
- `WriteIssue(id string, issue Issue)` — create blob → build tree → commit → update-ref
- `ReadIssue(id string) (Issue, error)` — resolve ref → read blob
- `ListIssues() ([]Issue, error)` — `for-each-ref` → `cat-file --batch`
- `UpdateIssue(id string, mutate func(*Issue))` — read → mutate → write new commit with parent
- `WatchRefs(ctx, callback)` — poll `for-each-ref` at interval, diff against previous snapshot

Tree construction uses `git mktree` with entries for `issue.json` plus the
`artifacts/` subtree. For event-only updates where artifacts don't change, the
artifacts subtree hash is reused from the parent commit.

### Data model (`model.go`)

```go
type Issue struct {
    ID          string   `json:"id"`
    Title       string   `json:"title"`
    Status      string   `json:"status"`
    Description string   `json:"description"`
    Labels      []string `json:"labels"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
    Events      []Event  `json:"events"`
}

type Event struct {
    Type      string    `json:"type"`
    Timestamp time.Time `json:"timestamp"`
    Data      map[string]any `json:"data"`
}
```

### Fake data generator (`fake.go`)

Generates K issues with realistic-looking data: random titles, statuses
distributed across BACKLOG/PLANNED/DOING/DONE, 1-5 events per issue,
placeholder artifact content.

### Benchmark harness (`bench_test.go`)

Standard Go benchmarks (`testing.B`):

| Benchmark | What it measures |
|---|---|
| BenchmarkBulkWrite | Create K issues from scratch |
| BenchmarkSingleRead | Read one issue by ID (ref already exists) |
| BenchmarkBoardRead | List + read all K issues |
| BenchmarkSingleUpdate | Update one issue (new commit, reuse artifact tree) |
| BenchmarkPackRefs | `git pack-refs --all` after K writes, measure size |
| BenchmarkWatch | Detect N changes via polling at 100ms interval |

### CLI runner (`main.go`)

A simple `main` that runs the full sequence with configurable K:

```
go run ./cmd/refbench -k 1000
```

Prints a summary table with timings and sizes.

## Decisions

1. **Bare repo** — no working tree needed, avoids checkout overhead
2. **Direct plumbing** — `hash-object`, `mktree`, `commit-tree`, `update-ref` via `os/exec`
   rather than a Go git library, to measure the real git overhead
3. **Polling for watch** — simplest viable approach; fsnotify on packed-refs
   is an alternative to explore if polling is too slow
4. **No concurrency in v1** — single-writer benchmarks first; concurrent writes
   are a follow-up concern

## Acceptance Criteria

- [ ] Standalone binary runs with `go run ./cmd/refbench -k 1000`
- [ ] Creates a temp bare git repo, writes K issues, benchmarks reads/writes
- [ ] Prints summary: write throughput (issues/sec), single-read latency, board-read latency, repo size
- [ ] `go test -bench .` runs the same benchmarks in standard Go format
- [ ] Watch detects a new issue created after the watcher starts
- [ ] All artifacts reuse the same tree hash when content is unchanged (verified in output)
