// One-time repair for xpo-e9a2eb: re-time issues.db lines whose created_at
// disagrees with file order, so time-ordered replay matches file-ordered
// replay. Not part of xpo; run once and discard.
//
//	go run ./scripts/repair-event-timestamps -db <hub>/.xpo/issues.db -backup <file> [-write]
//
// Run on 2026-09-28 against this repo's issues.db (3,331 events):
// 79 re-timed, 221 tie-nudged (max 178ns), 298 lines changed. Before the
// repair 3 issues differed between file and time order (xpo-70c971,
// xpo-d23f3e, xpo-834350); afterwards projection is identical to the
// original (excluding timestamps) and file order == time order.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"
	"sort"
	"time"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/model"
)

func main() {
	dbPath := flag.String("db", ".xpo/issues.db", "path to issues.db")
	backup := flag.String("backup", "", "backup file to write before rewriting (required with -write)")
	write := flag.Bool("write", false, "rewrite issues.db (default: dry run)")
	flag.Parse()

	if err := run(*dbPath, *backup, *write); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(dbPath, backup string, write bool) error {
	raw, err := os.ReadFile(dbPath)
	if err != nil {
		return err
	}
	lines := bytes.Split(bytes.TrimRight(raw, "\n"), []byte("\n"))

	orig := make([]time.Time, len(lines))
	for i, l := range lines {
		var e struct {
			CreatedAt time.Time `json:"created_at"`
		}
		if err := json.Unmarshal(l, &e); err != nil {
			return fmt.Errorf("line %d: %w", i+1, err)
		}
		orig[i] = e.CreatedAt
	}

	kept := longestNonDecreasing(orig)
	retimed := retime(orig, kept)
	tieNudged, maxNudge := breakTies(retimed)

	out := make([][]byte, len(lines))
	changed := 0
	for i, l := range lines {
		if retimed[i].Equal(orig[i]) {
			out[i] = l
			continue
		}
		nl, err := replaceCreatedAt(l, retimed[i])
		if err != nil {
			return fmt.Errorf("line %d: %w", i+1, err)
		}
		out[i] = nl
		changed++
	}

	fmt.Printf("events:           %d\n", len(lines))
	fmt.Printf("kept in order:    %d\n", countTrue(kept))
	fmt.Printf("re-timed:         %d\n", len(lines)-countTrue(kept))
	fmt.Printf("tie-nudged:       %d (max %s)\n", tieNudged, maxNudge)
	fmt.Printf("lines changed:    %d\n", changed)

	if err := verify(lines, out, retimed); err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	if !write {
		fmt.Println("dry run: nothing written")
		return nil
	}
	if backup == "" {
		return fmt.Errorf("-backup is required with -write")
	}
	if err := os.WriteFile(backup, raw, 0644); err != nil {
		return fmt.Errorf("backup: %w", err)
	}
	tmp := dbPath + ".tmp"
	if err := os.WriteFile(tmp, append(bytes.Join(out, []byte("\n")), '\n'), 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, dbPath); err != nil {
		return err
	}
	fmt.Printf("wrote %s (backup: %s)\n", dbPath, backup)
	return nil
}

// longestNonDecreasing marks one longest non-decreasing subsequence of ts.
func longestNonDecreasing(ts []time.Time) []bool {
	var tails []int // tails[k] = index of the smallest tail of a run of length k+1
	prev := make([]int, len(ts))
	for i, t := range ts {
		// first tail strictly greater than t (upper bound keeps ties in the run)
		pos := sort.Search(len(tails), func(k int) bool { return ts[tails[k]].After(t) })
		if pos > 0 {
			prev[i] = tails[pos-1]
		} else {
			prev[i] = -1
		}
		if pos == len(tails) {
			tails = append(tails, i)
		} else {
			tails[pos] = i
		}
	}
	kept := make([]bool, len(ts))
	if len(tails) == 0 {
		return kept
	}
	for i := tails[len(tails)-1]; i >= 0; i = prev[i] {
		kept[i] = true
	}
	return kept
}

// retime spreads each run of non-kept lines evenly between its kept neighbours.
func retime(ts []time.Time, kept []bool) []time.Time {
	out := make([]time.Time, len(ts))
	copy(out, ts)
	var keptIdx []int
	for i, k := range kept {
		if k {
			keptIdx = append(keptIdx, i)
		}
	}
	if len(keptIdx) == 0 {
		return out
	}
	first, last := keptIdx[0], keptIdx[len(keptIdx)-1]
	for i := 0; i < first; i++ {
		out[i] = ts[first].Add(-time.Duration(first-i) * time.Nanosecond)
	}
	for i := last + 1; i < len(ts); i++ {
		out[i] = ts[last].Add(time.Duration(i-last) * time.Nanosecond)
	}
	for n := 0; n+1 < len(keptIdx); n++ {
		a, b := keptIdx[n], keptIdx[n+1]
		gap := b - a
		if gap == 1 {
			continue
		}
		span := ts[b].Sub(ts[a])
		for j := 1; j < gap; j++ {
			out[a+j] = ts[a].Add(span / time.Duration(gap) * time.Duration(j))
		}
	}
	return out
}

// breakTies makes timestamps strictly increase by nudging forward in ns steps.
func breakTies(ts []time.Time) (nudged int, maxNudge time.Duration) {
	for i := 1; i < len(ts); i++ {
		if ts[i].After(ts[i-1]) {
			continue
		}
		want := ts[i-1].Add(time.Nanosecond)
		if d := want.Sub(ts[i]); d > maxNudge {
			maxNudge = d
		}
		ts[i] = want
		nudged++
	}
	return nudged, maxNudge
}

var createdAtKey = []byte(`"created_at":"`)

// replaceCreatedAt swaps the top-level created_at value, leaving every other
// byte of the line untouched. Go marshals created_at after payload and only
// plain strings follow it, so the last occurrence is the top-level field.
func replaceCreatedAt(line []byte, t time.Time) ([]byte, error) {
	start := bytes.LastIndex(line, createdAtKey)
	if start < 0 {
		return nil, fmt.Errorf("no created_at")
	}
	valStart := start + len(createdAtKey)
	valEnd := bytes.IndexByte(line[valStart:], '"')
	if valEnd < 0 {
		return nil, fmt.Errorf("unterminated created_at")
	}
	valEnd += valStart
	var out []byte
	out = append(out, line[:valStart]...)
	out = append(out, t.UTC().Format(time.RFC3339Nano)...)
	out = append(out, line[valEnd:]...)
	return out, nil
}

func verify(origLines, newLines [][]byte, want []time.Time) error {
	// Each line differs only in created_at, which parses to the intended value.
	for i := range origLines {
		var a, b map[string]any
		if err := json.Unmarshal(origLines[i], &a); err != nil {
			return err
		}
		if err := json.Unmarshal(newLines[i], &b); err != nil {
			return fmt.Errorf("line %d no longer parses: %w", i+1, err)
		}
		delete(a, "created_at")
		delete(b, "created_at")
		if !reflect.DeepEqual(a, b) {
			return fmt.Errorf("line %d changed beyond created_at", i+1)
		}
		var e model.Event
		json.Unmarshal(newLines[i], &e)
		if !e.CreatedAt.Equal(want[i]) {
			return fmt.Errorf("line %d created_at = %s, want %s", i+1, e.CreatedAt, want[i])
		}
		if i > 0 {
			var p model.Event
			json.Unmarshal(newLines[i-1], &p)
			if !e.CreatedAt.After(p.CreatedAt) {
				return fmt.Errorf("line %d not strictly after line %d", i+1, i)
			}
		}
	}

	origEvents, err := parse(origLines)
	if err != nil {
		return err
	}
	newEvents, err := parse(newLines)
	if err != nil {
		return err
	}

	before := normalized(exponential.ProjectIssues(origEvents))
	beforeTime := normalized(exponential.ProjectIssues(byTime(origEvents)))
	after := normalized(exponential.ProjectIssues(newEvents))
	afterTime := normalized(exponential.ProjectIssues(byTime(newEvents)))

	fmt.Printf("original, file vs time order: %d issues differ\n", diffCount(before, beforeTime))
	if n := diffCount(before, after); n != 0 {
		return fmt.Errorf("%d issues differ between original and repaired (file order)", n)
	}
	if n := diffCount(after, afterTime); n != 0 {
		return fmt.Errorf("%d issues differ between repaired file and time order", n)
	}
	fmt.Println("repaired: projection identical to original (excluding timestamps), file order == time order")
	return nil
}

func parse(lines [][]byte) ([]model.Event, error) {
	events := make([]model.Event, len(lines))
	for i, l := range lines {
		if err := json.Unmarshal(l, &events[i]); err != nil {
			return nil, err
		}
	}
	return events, nil
}

func byTime(events []model.Event) []model.Event {
	out := make([]model.Event, len(events))
	copy(out, events)
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

// normalized renders each issue as JSON with the event list dropped and every
// timestamp masked, so only board state is compared.
func normalized(issues map[string]*model.Issue) map[string]any {
	out := map[string]any{}
	for id, issue := range issues {
		b, _ := json.Marshal(issue)
		var m map[string]any
		json.Unmarshal(b, &m)
		delete(m, "Events")
		out[id] = maskTimes(m)
	}
	return out
}

func maskTimes(v any) any {
	switch x := v.(type) {
	case map[string]any:
		for k, vv := range x {
			x[k] = maskTimes(vv)
		}
		return x
	case []any:
		for i, vv := range x {
			x[i] = maskTimes(vv)
		}
		return x
	case string:
		if _, err := time.Parse(time.RFC3339Nano, x); err == nil {
			return "<time>"
		}
		return x
	default:
		return v
	}
}

func diffCount(a, b map[string]any) int {
	n := 0
	for id, v := range a {
		if !reflect.DeepEqual(v, b[id]) {
			n++
		}
	}
	for id := range b {
		if _, ok := a[id]; !ok {
			n++
		}
	}
	return n
}

func countTrue(bs []bool) int {
	n := 0
	for _, b := range bs {
		if b {
			n++
		}
	}
	return n
}
