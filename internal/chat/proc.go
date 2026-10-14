package chat

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

// agentProc is a running agent CLI: a line writer on its stdin, its stdout
// lines on a channel, and the tail of its stderr.
type agentProc struct {
	cmd     *exec.Cmd
	stdin   *lineWriter
	lines   chan []byte // closed after the last line
	stderr  *tailBuffer
	started time.Time

	waited  chan struct{} // closed once the process exited
	exited  chan struct{} // closed once the process exited and its output is read
	waitErr error         // set before waited closes
}

// outputGrace is how long the reader keeps reading after the agent exited.
// A child that inherited stdout, such as a detached background command,
// would otherwise keep it open.
const outputGrace = 2 * time.Second

// startAgent starts cmd in its own process group. stdout is a pipe of its
// own, so xpo stops reading once the agent exits, even when a child keeps
// the write end open.
func startAgent(cmd *exec.Cmd) (*agentProc, error) {
	setProcessGroup(cmd)
	p := &agentProc{cmd: cmd, lines: make(chan []byte, 64), stderr: &tailBuffer{max: 4096}, waited: make(chan struct{}), exited: make(chan struct{})}
	cmd.Stderr = p.stderr
	cmd.WaitDelay = outputGrace // a child holding stderr open cannot block Wait
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	cmd.Stdout = w
	if err := cmd.Start(); err != nil {
		r.Close()
		w.Close()
		return nil, fmt.Errorf("start %s: %w", cmd.Path, err)
	}
	w.Close()
	p.started = time.Now()
	p.stdin = &lineWriter{w: stdin}

	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		defer close(p.lines)
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 1024*1024), 16*1024*1024)
		for sc.Scan() {
			p.lines <- append([]byte(nil), sc.Bytes()...)
		}
	}()
	go func() {
		p.waitErr = cmd.Wait()
		close(p.waited)
		select {
		case <-readDone:
		case <-time.After(outputGrace):
		}
		r.Close()
		<-readDone
		close(p.exited)
	}()
	return p, nil
}

// pid is the agent's process ID, which is also its process group ID.
func (p *agentProc) pid() int { return p.cmd.Process.Pid }

// stop closes stdin and sends SIGTERM, so the agent stops its background
// commands. Once the agent exited, or after grace, it kills what is left of
// the process group, such as MCP servers. It returns once the agent has
// exited. The caller must keep draining lines.
func (p *agentProc) stop(grace time.Duration) {
	p.stdin.close()
	terminate(p.pid())
	select {
	case <-p.waited:
	case <-time.After(grace):
	}
	killGroup(p.pid())
	<-p.exited
}

// lineWriter writes JSON lines to a pipe from several goroutines, and
// ignores writes after close.
type lineWriter struct {
	mu     sync.Mutex
	w      io.WriteCloser
	closed bool
}

func (l *lineWriter) write(line []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return
	}
	_, _ = l.w.Write(append(line, '\n'))
}

func (l *lineWriter) close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.closed {
		l.closed = true
		_ = l.w.Close()
	}
}
