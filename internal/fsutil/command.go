// Package fsutil holds small, dependency-free filesystem, process and
// formatting helpers used across the app. Nothing here depends on the
// database or on application config; callers pass in whatever paths or
// values they need.
package fsutil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// CommandError is returned when a command exits with a non-zero status.
type CommandError struct {
	Output string
	Status int
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("command exited %d: %s", e.Status, strings.TrimSpace(e.Output))
}

// RunOptions configures RunCommand.
type RunOptions struct {
	// StderrToStdout merges stderr into the captured output (stdout only by
	// default).
	StderrToStdout bool
	// LoggingArgOverride is logged in place of the joined args, for
	// commands whose arguments shouldn't be logged verbatim.
	LoggingArgOverride string
}

// RunCommand runs command with args in dir, with its own process group so
// that cancelling ctx (the job being cancelled, or the app shutting down)
// kills the whole group, including any children (e.g. ffmpeg). It returns
// the captured output and exit status; an error is only returned if the
// executable itself couldn't be started.
func RunCommand(ctx context.Context, dir string, command string, args []string, opts RunOptions) (string, int, error) {
	loggingArgs := opts.LoggingArgOverride
	if loggingArgs == "" {
		loggingArgs = strings.Join(args, " ")
	}
	slog.Info("[command_wrapper]: " + command + " called with: " + loggingArgs)

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = 5 * time.Second

	var out bytes.Buffer
	cmd.Stdout = &out
	if opts.StderrToStdout {
		cmd.Stderr = &out
	} else {
		cmd.Stderr = os.Stderr
	}

	status := 0
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			// The executable couldn't be started at all.
			return "", 0, err
		}
		status = exitErr.ExitCode()
		if status < 0 { // killed by a signal (e.g. job cancelled)
			status = 137
		}
	}

	output := out.String()
	logCommandResult(command, loggingArgs, status, output)
	return output, status, nil
}

func logCommandResult(command, loggingArgs string, status int, output string) {
	msg := "[command_wrapper]: " + command + " called with: " + loggingArgs + " exited: " + strconv.Itoa(status) + " with: " + output
	if status == 0 {
		slog.Debug(msg)
	} else {
		slog.Error(msg)
	}
}
