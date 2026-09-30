package storage

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/palarix/exponential/internal/model"
)

const helperDirEnv = "XPO_TEST_APPEND_HELPER_DIR"

// assertAllPresentOnce checks every line of issues.db parses and that each
// expected ID appears exactly once.
func assertAllPresentOnce(t *testing.T, want []string) {
	t.Helper()
	f, err := os.Open(filepath.Join(".xpo", "issues.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	seen := make(map[string]int)
	scanner := bufio.NewScanner(f)
	line := 0
	for scanner.Scan() {
		line++
		var evt model.Event
		if err := json.Unmarshal(scanner.Bytes(), &evt); err != nil {
			t.Fatalf("line %d is corrupt: %v: %q", line, err, scanner.Text())
		}
		seen[evt.ID]++
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}

	for _, id := range want {
		if seen[id] != 1 {
			t.Errorf("event %s present %d times, want 1", id, seen[id])
		}
	}
	if line != len(want) {
		t.Errorf("issues.db has %d lines, want %d", line, len(want))
	}
}

func TestConcurrentAppendAndCollapse_NoLostEvents(t *testing.T) {
	setupTestRepo(t)
	writeEvents(t, []model.Event{makeEvent("seed", model.EventTypeCreate)})
	commitDB(t)

	const n = 40
	want := []string{"seed"}
	for i := 0; i < n; i++ {
		want = append(want, fmt.Sprintf("g%d", i))
	}

	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			evt := makeEvent(fmt.Sprintf("g%d", i), model.EventTypeCreate)
			if i%2 == 0 {
				errs <- AppendEvent(evt)
			} else {
				errs <- AppendEventCollapsed(evt)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	assertAllPresentOnce(t, want)
}

// TestHelperProcessAppend is not a real test: TestCrossProcessAppendAndCollapse
// re-runs the test binary with helperDirEnv set, and this appends from that
// second process.
func TestHelperProcessAppend(t *testing.T) {
	dir := os.Getenv(helperDirEnv)
	if dir == "" {
		t.Skip("helper process only")
	}
	count, _ := strconv.Atoi(os.Getenv(helperDirEnv + "_COUNT"))
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	ResetHubRoot()
	for i := 0; i < count; i++ {
		if err := AppendEvent(makeEvent(fmt.Sprintf("p%d", i), model.EventTypeCreate)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCrossProcessAppendAndCollapse_NoLostEvents(t *testing.T) {
	dir := setupTestRepo(t)
	writeEvents(t, []model.Event{makeEvent("seed", model.EventTypeCreate)})
	commitDB(t)

	const helperCount = 200
	const collapseCount = 40

	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperProcessAppend$")
	cmd.Env = append(os.Environ(),
		helperDirEnv+"="+dir,
		helperDirEnv+"_COUNT="+strconv.Itoa(helperCount),
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < collapseCount; i++ {
		if err := AppendEventCollapsed(makeEvent(fmt.Sprintf("c%d", i), model.EventTypeCreate)); err != nil {
			t.Fatal(err)
		}
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("helper process failed: %v\n%s", err, out.String())
	}

	want := []string{"seed"}
	for i := 0; i < helperCount; i++ {
		want = append(want, fmt.Sprintf("p%d", i))
	}
	for i := 0; i < collapseCount; i++ {
		want = append(want, fmt.Sprintf("c%d", i))
	}
	assertAllPresentOnce(t, want)
}

func TestAppendEvent_WaitsForEventsLock(t *testing.T) {
	setupXpoDir(t)

	release := make(chan struct{})
	locked := make(chan struct{})
	go withEventsLock(func() error {
		close(locked)
		<-release
		return nil
	})
	<-locked

	done := make(chan error, 1)
	go func() { done <- AppendEvent(makeEvent("a", model.EventTypeCreate)) }()

	select {
	case <-done:
		t.Fatal("AppendEvent wrote while another writer held the events lock")
	case <-time.After(100 * time.Millisecond):
	}

	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
