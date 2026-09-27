package fsutil_test

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/mattbriancon/pinchflat/internal/fsutil"
)

func TestRunCommand(t *testing.T) {
	t.Run("runs the command and captures its output", func(t *testing.T) {
		dir := t.TempDir()
		output, status, err := fsutil.RunCommand(context.Background(), dir, "echo", []string{"output"}, fsutil.RunOptions{})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if status != 0 {
			t.Errorf("expected status 0, got %d", status)
		}
		if !strings.Contains(output, "output") {
			t.Errorf("expected output to contain 'output', got: %q", output)
		}
	})

	t.Run("sets the working directory", func(t *testing.T) {
		dir := t.TempDir()
		output, status, err := fsutil.RunCommand(context.Background(), dir, "pwd", []string{}, fsutil.RunOptions{})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if status != 0 {
			t.Errorf("expected status 0, got %d", status)
		}
		if !strings.Contains(output, filepath.Base(dir)) {
			t.Errorf("expected output to contain %q, got: %q", dir, output)
		}
	})

	t.Run("returns a non-zero status without an error", func(t *testing.T) {
		dir := t.TempDir()
		_, status, err := fsutil.RunCommand(context.Background(), dir, "/bin/false", nil, fsutil.RunOptions{})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if status != 1 {
			t.Errorf("expected status 1, got %d", status)
		}
	})

	t.Run("merges stderr into the output when requested", func(t *testing.T) {
		dir := t.TempDir()
		output, _, err := fsutil.RunCommand(context.Background(), dir, "sh", []string{"-c", "echo err 1>&2"}, fsutil.RunOptions{StderrToStdout: true})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !strings.Contains(output, "err") {
			t.Errorf("expected output to contain 'err', got: %q", output)
		}
	})
}

// TestRunCommandCancellationKillsProcessGroup is a Go-only test: cancelling
// the context must kill the command and its children (what
// priv/cmd_wrapper.sh guaranteed in Elixir).
func TestRunCommandCancellationKillsProcessGroup(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		// The child sleep outlives its parent shell unless the group is killed.
		fsutil.RunCommand(ctx, dir, "sh", []string{"-c", "sleep 60 & echo $! > " + pidFile + "; wait"}, fsutil.RunOptions{})
		close(done)
	}()
	var pid int
	for i := 0; i < 100 && pid == 0; i++ {
		time.Sleep(20 * time.Millisecond)
		b, _ := os.ReadFile(pidFile)
		pid, _ = strconv.Atoi(string(trimNL(b)))
	}
	if pid == 0 {
		t.Fatal("child never started")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("RunCommand did not return after cancel")
	}
	time.Sleep(50 * time.Millisecond)
	if err := syscall.Kill(pid, 0); err == nil && !isZombie(pid) {
		t.Fatalf("child process %d still alive after cancel", pid)
	}
}

func trimNL(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r') {
		b = b[:len(b)-1]
	}
	return b
}

// isZombie reports a killed process that nobody has reaped yet (common in
// containers whose PID 1 doesn't reap orphans).
func isZombie(pid int) bool {
	b, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return true
	}
	s := string(b)
	i := strings.LastIndexByte(s, ')')
	return i >= 0 && i+2 < len(s) && s[i+2] == 'Z'
}
