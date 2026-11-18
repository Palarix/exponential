package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// useTempDir points the registry at a fresh temp dir for the test.
func useTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Cleanup(SetDirForTesting(dir))
	return dir
}

func readFile(t *testing.T, dir string) []Entry {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "instances.json"))
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("registry is not valid JSON: %v\n%s", err, data)
	}
	return entries
}

func assertNoTmpFiles(t *testing.T, dir string) {
	t.Helper()
	names, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range names {
		if strings.HasSuffix(n.Name(), ".tmp") {
			t.Errorf("leftover tmp file %s", n.Name())
		}
	}
}

func TestSetDirForTestingRestores(t *testing.T) {
	before := dir()
	restore := SetDirForTesting(t.TempDir())
	if dir() == before {
		t.Fatal("SetDirForTesting did not change the dir")
	}
	restore()
	if dir() != before {
		t.Fatalf("restore: dir = %q, want %q", dir(), before)
	}
}

func TestRegisterAndList(t *testing.T) {
	d := useTempDir(t)
	if err := Register("a", 1234, "/root"); err != nil {
		t.Fatal(err)
	}
	got, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].PID != os.Getpid() || got[0].Port != 1234 {
		t.Fatalf("List = %+v", got)
	}
	if err := UnregisterByPID(os.Getpid()); err != nil {
		t.Fatal(err)
	}
	if n := len(readFile(t, d)); n != 0 {
		t.Fatalf("after unregister: %d entries", n)
	}
	assertNoTmpFiles(t, d)
}

func TestListRecoversFromCorruptFile(t *testing.T) {
	for name, content := range map[string]string{
		"extra bracket": "[]\n]",
		"empty":         "",
		"garbage":       "{not json",
	} {
		t.Run(name, func(t *testing.T) {
			d := useTempDir(t)
			path := filepath.Join(d, "instances.json")
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				t.Fatal(err)
			}
			got, err := List()
			if err != nil {
				t.Fatalf("List on corrupt file: %v", err)
			}
			if len(got) != 0 {
				t.Fatalf("List = %+v, want empty", got)
			}
			if n := len(readFile(t, d)); n != 0 {
				t.Fatalf("rewritten file has %d entries", n)
			}
		})
	}
}

func TestRegisterRecoversFromCorruptFile(t *testing.T) {
	d := useTempDir(t)
	if err := os.WriteFile(filepath.Join(d, "instances.json"), []byte("[]]"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Register("a", 1, "/r"); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, d)
	if len(got) != 1 || got[0].PID != os.Getpid() {
		t.Fatalf("entries = %+v", got)
	}
}

func TestUnregisterRecoversFromCorruptFile(t *testing.T) {
	d := useTempDir(t)
	if err := os.WriteFile(filepath.Join(d, "instances.json"), []byte("[]]"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := UnregisterByPID(12345); err != nil {
		t.Fatal(err)
	}
	if n := len(readFile(t, d)); n != 0 {
		t.Fatalf("entries = %d", n)
	}
}

func TestConcurrentGoroutineWritesStayValid(t *testing.T) {
	d := useTempDir(t)
	// Every writer is this process, so liveness pruning never drops its
	// entry. Each goroutine registers, lists and unregisters; the file must
	// stay valid JSON throughout and end empty.
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := Register(fmt.Sprintf("b%d", i), 1000+i, "/r"); err != nil {
				t.Errorf("Register: %v", err)
			}
			if _, err := List(); err != nil {
				t.Errorf("List: %v", err)
			}
			if err := UnregisterByPID(os.Getpid()); err != nil {
				t.Errorf("Unregister: %v", err)
			}
		}(i)
	}
	wg.Wait()
	if n := len(readFile(t, d)); n != 0 {
		t.Fatalf("entries = %d, want 0", n)
	}
	assertNoTmpFiles(t, d)
}

const helperEnv = "XPO_REGISTRY_TEST_HELPER_DIR"

// TestHelperProcess is run as a child process by
// TestConcurrentProcessWritesStayValid. It registers, waits for a release
// file, and exits after unregistering.
func TestHelperProcess(t *testing.T) {
	d := os.Getenv(helperEnv)
	if d == "" {
		t.Skip("helper process only")
	}
	SetDirForTesting(d)
	for i := 0; i < 20; i++ {
		if err := Register("child", 1, "/r"); err != nil {
			fmt.Fprintln(os.Stderr, "register:", err)
			os.Exit(2)
		}
		if _, err := List(); err != nil {
			fmt.Fprintln(os.Stderr, "list:", err)
			os.Exit(2)
		}
	}
	// Stay alive until released so the parent can count live entries.
	release := filepath.Join(d, "release")
	for {
		if _, err := os.Stat(release); err == nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err := UnregisterByPID(os.Getpid()); err != nil {
		fmt.Fprintln(os.Stderr, "unregister:", err)
		os.Exit(2)
	}
	os.Exit(0)
}

func TestConcurrentProcessWritesStayValid(t *testing.T) {
	d := useTempDir(t)
	const n = 8
	cmds := make([]*exec.Cmd, n)
	for i := range cmds {
		cmd := exec.Command(os.Args[0], "-test.run=^TestHelperProcess$")
		cmd.Env = append(os.Environ(), helperEnv+"="+d)
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		cmds[i] = cmd
	}
	t.Cleanup(func() {
		for _, c := range cmds {
			if c.ProcessState == nil {
				_ = c.Process.Kill()
				_ = c.Wait()
			}
		}
	})

	// Wait until every child has registered.
	want := map[string]bool{}
	for _, c := range cmds {
		want[strconv.Itoa(c.Process.Pid)] = true
	}
	deadline := time.Now().Add(20 * time.Second)
	for {
		got, err := List()
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) == n {
			for _, e := range got {
				if !want[strconv.Itoa(e.PID)] {
					t.Fatalf("unexpected entry %+v", e)
				}
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out: %d of %d children registered", len(got), n)
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := os.WriteFile(filepath.Join(d, "release"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	for _, c := range cmds {
		if err := c.Wait(); err != nil {
			t.Fatalf("child %d: %v", c.Process.Pid, err)
		}
	}
	if got := readFile(t, d); len(got) != 0 {
		t.Fatalf("entries after all children exited = %+v", got)
	}
	assertNoTmpFiles(t, d)
}
