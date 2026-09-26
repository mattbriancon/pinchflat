package core_test

// Go-only test: cancelling the context must kill the command and its
// children (what priv/cmd_wrapper.sh guaranteed in Elixir).

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/mattbriancon/pinchflat/internal/core/coretest"
)

func TestCliUtilsWrapCmdCancellationKillsProcessGroup(t *testing.T) {
	ta := coretest.NewApp(t)
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		// The child sleep outlives its parent shell unless the group is killed.
		ta.CliUtilsWrapCmd(ctx, "sh", []string{"-c", "sleep 60 & echo $! > " + pidFile + "; wait"}, nil, nil)
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
		t.Fatal("WrapCmd did not return after cancel")
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
