# Spec: Refs-based storage for xpo state

## What

Migrate xpo's data layer from `.xpo/issues.db` (a file on the working tree committed
to `main`) to `refs/xpo/data` (a custom git ref with its own commit chain, invisible
to the working tree). Add transparent transport setup so `git fetch` / `git push`
carry the data ref alongside code. No dedicated xpo sync commands — regular git
tooling is the only interface for network operations.

## Why

The current `.xpo/issues.db` file on `main` has three problems:

1. **Merge conflicts** — every branch that touches an issue modifies the same file,
   creating conflicts on merge that have nothing to do with code.
2. **Working-tree pollution** — the data file shows up in `git status`, diffs, and
   PRs, cluttering the code change view.
3. **Clone tax** — contributors who don't use xpo still download the full event
   history. Moving to a custom ref makes it opt-in: only users who enable xpo
   get the data.

The ref-based approach stores events as blobs in a git tree committed to
`refs/xpo/data`, using git's native content-addressable storage. The working tree
is never touched for issue data.

## Prior art

### git-bug

git-bug uses a similar event-sourced model stored on custom git refs
(`refs/bugs/<id>`, one ref per entity). Key differences from our approach:

- git-bug uses **per-entity refs** — better concurrency and per-issue reads, but
  requires `git bug push` / `git bug pull` because custom refs aren't fetched by
  default. We avoid this UX gap with refspec injection + hooks.
- git-bug uses **Lamport clocks** for ordering concurrent edits in a DAG. We use
  CAS (compare-and-swap) on a single ref, which serializes writes. This is
  sufficient for xpo's current scale (single-digit concurrent users).
- git-bug fetches into remote-tracking refs then runs application-level `MergeAll`.
  We adopt the same two-tier ref pattern but merge automatically via git hooks.
- A future migration from single-ref to per-entity refs is mechanical — the event
  model stays the same, only the storage layout changes.

### entire

entire (entireio) is a git-native platform that stores agent session data
alongside commits using hooks for capture. Same data-transportation problem,
similar git-native approach. Key differences:

- entire requires dedicated CLI commands (`entire status`, `entire rewind`, etc.)
  for interacting with stored data. We aim for zero dedicated sync commands.
- entire's onboarding command is `enable` rather than `init` — a better name
  since it works for both repo creators and contributors. We adopt this naming.

## Design decisions

### Single ref vs per-entity refs

**Choice:** single ref (`refs/xpo/data`) with all events in one `issues.db` blob.

**Rationale:** Simpler implementation, fewer ref negotiations on fetch, and xpo
doesn't have the scale (thousands of issues, many concurrent editors) where
per-entity refs pay off. The CAS retry loop handles the concurrency we do have.
Migration path to per-entity refs exists if needed later.

### Local-first — xpo never touches the network

**Choice:** all xpo reads and writes operate on local refs only. xpo never calls
`git fetch`, `git push`, or any network operation.

**Rationale:** the natural appeal of storing issues in git is that they travel
with the repo. Clone the repo, hop on an airplane, create and update issues all
day, push everything when you land. Network sync is delegated entirely to git's
existing transport — the user already runs `git fetch` and `git push` for their
code, and xpo data rides along via the injected refspec.

### Zero dedicated sync commands

**Choice:** no `xpo sync`, `xpo pull`, `xpo push`, or any network-facing command.
The full workflow uses only regular git:

- `git fetch` / `git pull` → brings code + xpo data (via refspec)
- `git push` → sends code + xpo data (via refspec)
- Git hooks handle reconciliation automatically in the background

**Rationale:** every dedicated sync command is a UX tax. If the user has to
remember `xpo sync` after `git pull`, the integration is leaky. The data layer
should be invisible — it travels with git, merges itself, and never asks the user
to think about it.

### `xpo enable` — the universal entry point

**Choice:** rename/split the setup into two commands:

- **`xpo enable`** — sets up local transport (refspec injection, hook
  installation, fast-forward from remote-tracking ref). Works for both repo
  creators and contributors. Idempotent. This is the command you tell a new
  team member to run.
- **`xpo init`** — creates a new xpo project (writes `.xpo/config.yaml`, creates
  the initial empty `refs/xpo/data` ref). Only the repo creator runs this. Calls
  `xpo enable` internally afterward.

**Rationale:** `init` implies "create something new" — wrong for a contributor
joining an existing project. `enable` implies "activate something that's here" —
works for everyone. Borrowed from entire's naming, which gets this right.

### Lazy `ensureTransport()` — automatic fallback

**Choice:** every xpo command calls `ensureTransport()` internally, which does
exactly what `xpo enable` does. If someone goes straight to `xpo list` without
running `xpo enable` first, the transport is set up silently.

**Rationale:** `xpo enable` exists as an explicit, documentable step for
onboarding. But if someone skips it, things should still work. The lazy setup is
a safety net, not the primary path.

### Remote-tracking refs (two-tier)

**Choice:** fetch into `refs/remotes/<remote>/xpo/*`, keep local state at
`refs/xpo/*`. Merge via hooks.

The fetch refspec injected into `.git/config`:

```
[remote "origin"]
    fetch = +refs/xpo/*:refs/remotes/origin/xpo/*
```

The push refspec:

```
[remote "origin"]
    push = refs/xpo/data:refs/xpo/data
```

**Rationale:** the naive refspec `+refs/xpo/*:refs/xpo/*` maps remote directly
onto local, so `git fetch` **clobbers local state** — any events written locally
but not yet pushed are destroyed. The two-tier model mirrors how normal branches
work:

- `refs/xpo/data` — local working copy (what xpo reads and writes)
- `refs/remotes/origin/xpo/data` — last-known remote state (updated by `git fetch`)

### Hook-based reconciliation

**Choice:** install git hooks into `.git/hooks/` to reconcile xpo refs
automatically. No user action needed after `xpo enable`.

**Hooks installed:**

- **`post-merge`** — fires after `git pull` (which does fetch + merge). Runs
  `xpo reconcile` (internal subcommand) to merge remote-tracking xpo ref into
  local xpo ref. By the time the user's terminal prompt returns, xpo data is
  merged.
- **`pre-push`** — fires before `git push`. Runs `xpo reconcile` to ensure the
  local xpo ref is fast-forwardable to the remote. The push never fails for xpo
  reasons.
- **`post-fetch`** *(if supported by git version)* — fires after `git fetch`
  (without merge). Same reconciliation. Covers the case where the user runs
  `git fetch` separately from `git pull`.

**Preserving existing hooks (git-lfs pattern):**

If a hook file already exists when xpo installs its hook:

1. Rename the existing hook to `<hookname>.pre-xpo` (e.g. `pre-push.pre-xpo`)
2. Install an xpo wrapper script that:
   a. Runs the original hook first (`<hookname>.pre-xpo`)
   b. If it succeeds, runs `xpo reconcile`
   c. Passes through the original hook's exit code

This is the same approach `git-lfs install` uses. It's simple, doesn't depend on
external hook managers (husky, lefthook, etc.), and preserves whatever the project
already had.

**`xpo reconcile` — internal subcommand:**

Not user-facing. Called by hooks. Does the actual merge work:

1. Check if `refs/remotes/<remote>/xpo/data` exists and differs from
   `refs/xpo/data`
2. If identical or remote doesn't exist → no-op (common case, fast)
3. If local is ancestor of remote → fast-forward local ref
4. If remote is ancestor of local → already ahead, no-op
5. If diverged → application-level event merge

### Conflict resolution: application-level event merge

Git cannot merge two `issues.db` blobs — they're opaque content. But xpo events
are **append-only, independently meaningful, and idempotent**, which makes
application-level merge straightforward.

**Merge algorithm:**

1. Find the common ancestor commit of local and remote refs (`git merge-base`)
2. Read the ancestor's `issues.db` → baseline events
3. Read local `issues.db` → baseline + local-only events
4. Read remote `issues.db` → baseline + remote-only events
5. Merged result = baseline + union(local-only, remote-only), ordered by timestamp
6. Deduplicate by event ID (if both sides created the same event)
7. Write merged `issues.db` as a new commit with two parents (a merge commit on
   the local ref)
8. Update local `refs/xpo/data` to the merge commit

**Why this works:**

- Events are **append-only** — neither side deletes or edits events in the log,
  so "union" is always safe.
- Events are **self-describing** — each carries its issue ID, type, timestamp,
  and full payload. No positional meaning.
- Event replay is **idempotent** — applying the same event twice produces the
  same state. A duplicated event in the merge is harmless (and the dedup step
  removes it anyway).
- Timestamp ordering is **deterministic** — ties broken by event ID
  (lexicographic), same as git-bug's approach. Unbiased and hard to abuse.

**What can't conflict:** two devs updating different issues — their events simply
interleave.

**What could conflict semantically:** two devs updating the same field on the
same issue (e.g., both change the title). The event model handles this naturally:
last-writer-wins by timestamp ordering. Both updates are preserved in the event
history; the final projected state reflects whichever timestamp is later.

**Artifacts (spec.md, walkthrough.md, etc.):** these are standalone blobs in the
tree, not events. If both sides modified the same artifact, the merge uses the
remote version (fetch wins) and emits a warning. In practice this is rare —
artifacts are written once and not collaboratively edited.

### Collapse / noop pruning

The existing event collapse logic (merging consecutive updates to the same issue
before committing, pruning no-op updates against committed state) is preserved
and applied before writing to the ref. This reduces commit noise in the ref's
history.

## Flow

### 1. `xpo enable` / `ensureTransport()` — transport setup

1. Check if inside a git repo (skip if not)
2. Detect the default remote (usually `origin`; fall back to first remote)
3. Inject fetch refspec `+refs/xpo/*:refs/remotes/<remote>/xpo/*` (if missing)
4. Inject push refspec `refs/xpo/data:refs/xpo/data` (if missing)
5. Install git hooks (`post-merge`, `pre-push`, `post-fetch`) into `.git/hooks/`
   - Preserve existing hooks via the wrapper/rename pattern
6. If local `refs/xpo/data` doesn't exist but `refs/remotes/<remote>/xpo/data`
   does (from a prior `git fetch`) → create local ref pointing to the same
   commit (initial fast-forward for new contributors)

All steps are idempotent. Called explicitly via `xpo enable` or implicitly by any
xpo command via `ensureTransport()`.

### 2. `xpo init` — project creation (repo creator only)

1. Create `.xpo/config.yaml` (as today)
2. Create `refs/xpo/data` ref with empty tree commit
3. Call `xpo enable` / `ensureTransport()` to set up transport
4. Append `.xpo/issues.db` and `.xpo/archive.db` to `.gitignore` (cleanup of
   legacy paths, if they exist)

### 3. Event read — fully local

1. Call `ensureTransport()` (fast no-op after first run)
2. `git show refs/xpo/data:issues.db` → parse NDJSON events → assemble state
   via `ProjectIssues()`
3. No network, no remote check — hooks handle reconciliation separately

### 4. Event write — fully local

- JSON-marshal event → `hash-object -w` → rebuild tree → `commit-tree`
  → `update-ref` (with CAS for local concurrent safety)
- No network, no push attempt
- Data reaches the remote on the user's next `git push`

### 5. `git pull` — automatic reconciliation

1. Git fetches + merges code (normal behavior)
2. Fetch updates `refs/remotes/<remote>/xpo/data` (via refspec)
3. `post-merge` hook fires → `xpo reconcile` merges xpo refs locally
4. User's terminal prompt returns with everything synced

### 6. `git push` — automatic reconciliation

1. `pre-push` hook fires → `xpo reconcile` ensures local xpo ref is ahead
2. Git pushes code + xpo data (via push refspec)
3. Push always succeeds for xpo ref (reconciled in step 1)

### 7. `git fetch` — background data arrival

1. Git fetches all refs including xpo (via refspec)
2. `post-fetch` hook fires → `xpo reconcile` merges xpo refs locally
3. Next xpo command sees merged state immediately

### 8. Artifacts

Stored as blobs at `artifacts/<issue-id>/<filename>` in the `refs/xpo/data`
tree. Read/write/delete via the refstore plumbing. Fully local operations.
Merged by last-writer-wins with a warning on conflict.

## Acceptance criteria

### Storage
- [ ] `refstore` package provides blob/tree/commit plumbing for `refs/xpo/data`
- [ ] `refstate` adapter exposes `ReadEventsRef`, `AppendEventRef`,
      `AppendEventRefCAS`, `ValidateEventsRef` and artifact operations
- [ ] All existing storage callers use the ref-based path instead of file I/O
- [ ] All reads and writes are fully local — no network calls in any xpo code path
- [ ] Collapse/noop pruning works against ref-based storage
- [ ] CAS retry handles local concurrent writes
- [ ] Legacy `.xpo/issues.db` file path is no longer read or written

### Transport setup (`xpo enable` / `ensureTransport()`)
- [ ] `xpo enable` command exists as the explicit entry point for any user
- [ ] `ensureTransport()` is called lazily by every xpo command as a fallback
- [ ] Idempotent — running multiple times is a no-op
- [ ] Injects fetch refspec `+refs/xpo/*:refs/remotes/<remote>/xpo/*`
- [ ] Injects push refspec `refs/xpo/data:refs/xpo/data`
- [ ] Installs `post-merge`, `pre-push`, and `post-fetch` hooks
- [ ] Preserves existing hooks via wrapper/rename pattern (git-lfs style)
- [ ] Creates local ref from remote-tracking ref if local is missing
      (initial fast-forward for new contributors)
- [ ] `xpo init` calls `ensureTransport()` after creating the project

### Reconciliation (`xpo reconcile` — internal, hook-triggered)
- [ ] Fast-forwards local ref when remote-tracking ref is strictly ahead
- [ ] Application-level event merge when refs diverge
- [ ] Merge produces a two-parent commit on the local ref
- [ ] Events ordered by timestamp, deduplicated by event ID during merge
- [ ] Artifact conflicts resolved by last-writer-wins with warning
- [ ] No-op when refs are identical or remote doesn't exist (fast path)

### End-to-end
- [ ] `git pull` → xpo data arrives and merges automatically (no user action)
- [ ] `git push` → xpo data is sent automatically (no user action)
- [ ] No dedicated xpo sync/pull/push commands exist
- [ ] Works fully offline — all xpo commands function without network
- [ ] `make test` passes
