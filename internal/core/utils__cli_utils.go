package core

import (
	"context"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// (a *App) CliUtilsWrapCmd(ctx, command, args, passthrough_opts, opts)
// Wraps a command in a shell script that will terminate the command if stdin is closed.
// Useful for stopping commands if the job runner is cancelled.
func (a *App) CliUtilsWrapCmd(ctx context.Context, command string, args []string, passthroughOpts KW, opts KW) (string, int, error) {
	// PrivDirectory is not in Config yet; compute from app structure or use a fixed path
	// For now, assume cmd_wrapper.sh is in priv/ relative to the binary or repo root
	wrapperCommand := filepath.Join(filepath.Dir(filepath.Dir(a.Config.DatabasePath)), "priv", "cmd_wrapper.sh")
	actualCommand := append([]string{command}, args...)
	commandOpts := setCommandOpts(a)
	// Combine commandOpts with passthroughOpts
	allOpts := append(commandOpts, passthroughOpts...)

	loggingArgOverride := ""
	if v, ok := opts.Get("logging_arg_override"); ok {
		loggingArgOverride = v.(string)
	} else {
		loggingArgOverride = strings.Join(args, " ")
	}

	slog.Info("[command_wrapper]: " + command + " called with: " + loggingArgOverride)

	cmd := exec.CommandContext(ctx, wrapperCommand, actualCommand...)

	// Apply command options (cd is the main one)
	if cdVal, ok := allOpts.Get("cd"); ok {
		cmd.Dir = cdVal.(string)
	}

	output, err := cmd.CombinedOutput()
	status := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			status = exitErr.ExitCode()
		} else {
			// Actual system error (file not found, etc.)
			return "", 0, err
		}
	}

	logCmdResult(command, loggingArgOverride, status, string(output))

	return string(output), status, nil
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
			// Bare atom flag: prepend to list (but Elixir code appends)
			// Following the code, not the comment:
			return append(acc, "--"+StringUtilsToKebabCase(v.Key))
		}
		// Key-value pair with atom key
		return append(acc, "--"+StringUtilsToKebabCase(v.Key), toString(v.Value))
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
