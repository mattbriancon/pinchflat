package core

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

// (a *App) CliUtilsWrapCmd(ctx, command, args, passthrough_opts, opts)
// Wraps a command in a shell script that will terminate the command if stdin is closed.
// Useful for stopping commands if the job runner is cancelled.
func (a *App) CliUtilsWrapCmd(ctx context.Context, command string, args []string, passthroughOpts KW, opts KW) (string, int, error) {
	// Elixir ran commands through priv/cmd_wrapper.sh so they died when the
	// job did. Here the command gets its own process group and cancelling
	// ctx (job cancelled, app shutting down) kills the whole group, which
	// also takes out children such as ffmpeg.
	commandOpts := append(setCommandOpts(a), passthroughOpts...)

	loggingArgOverride := strings.Join(args, " ")
	if v, ok := opts.Get("logging_arg_override"); ok {
		loggingArgOverride = v.(string)
	}

	slog.Info("[command_wrapper]: " + command + " called with: " + loggingArgOverride)

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = 5 * time.Second
	if cd, ok := commandOpts.Get("cd"); ok {
		cmd.Dir = cd.(string)
	}
	if env, ok := commandOpts.Get("env"); ok {
		cmd.Env = os.Environ()
		for _, kv := range env.(KW) {
			cmd.Env = append(cmd.Env, kv.Key+"="+fmt.Sprint(kv.Value))
		}
	}

	// System.cmd captures stdout only, unless stderr_to_stdout: true.
	var out bytes.Buffer
	cmd.Stdout = &out
	if commandOpts.Bool("stderr_to_stdout") {
		cmd.Stderr = &out
	} else {
		cmd.Stderr = os.Stderr
	}

	status := 0
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			// System.cmd raises when the executable can't be started.
			return "", 0, err
		}
		status = exitErr.ExitCode()
		if status < 0 { // killed by a signal (e.g. job cancelled)
			status = 137
		}
	}

	output := out.String()
	logCmdResult(command, loggingArgOverride, status, output)
	return output, status, nil
}

// CliUtilsParseOptions(command_opts)
// Parses a list of command options into a list of strings suitable for passing to System.cmd.
// Handles three types of inputs:
// 1. KW (keyword list) of key-value pairs
// 2. Individual items (strings or KV pairs)
// For atom keys, converts to kebab-case strings prefixed with --
// For string keys, keeps as-is
// Returns a flattened slice of strings.
func CliUtilsParseOptions(commandOpts interface{}) []string {
	var items []interface{}

	// Wrap in a slice if it's not already
	switch v := commandOpts.(type) {
	case KW:
		for _, kv := range v {
			items = append(items, kv)
		}
	case []KV:
		for _, kv := range v {
			items = append(items, kv)
		}
	case string:
		items = []interface{}{v}
	case KV:
		items = []interface{}{v}
	default:
		items = []interface{}{v}
	}

	// Reduce over items, building result
	var result []string
	for _, item := range items {
		result = cliUtilsParseOption(item, result)
	}

	return result
}

// cliUtilsParseOption processes one item, appending to accumulator.
// Faithfully ports the Elixir reduce logic.
func cliUtilsParseOption(item interface{}, acc []string) []string {
	switch v := item.(type) {
	case KV:
		if v.Flag {
			// Bare atom flag: needs kebab-case prefix
			return append(acc, "--"+StringUtilsToKebabCase(v.Key))
		}
		// Key-value pair. Check if key is already formatted (starts with -) or needs conversion
		key := v.Key
		if !strings.HasPrefix(key, "-") {
			// It's an atom key that needs kebab-case conversion
			key = "--" + StringUtilsToKebabCase(key)
		}
		// else: it's already a string key like "--under_score", use as-is
		return append(acc, key, toString(v.Value))
	case string:
		// String argument: append as-is
		return append(acc, v)
	default:
		return append(acc, toString(v))
	}
}

func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	default:
		return ""
	}
}

func setCommandOpts(a *App) KW {
	// Resolves an issue where yt-dlp would attempt to write to a read-only directory
	// if you scanned a new video with `--windows-filenames` enabled.
	return KW{Opt("cd", a.Config.TmpfileDirectory)}
}

func logCmdResult(command string, loggingArgOverride string, status int, output string) {
	logMessage := "[command_wrapper]: " + command + " called with: " + loggingArgOverride + " exited: " + strconv.Itoa(status) + " with: " + output
	if status == 0 {
		slog.Debug(logMessage)
	} else {
		slog.Error(logMessage)
	}
}
