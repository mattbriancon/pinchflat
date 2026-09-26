package core

import (
	"context"
	"errors"
	"strings"
)

// NotificationsCommandRunner is Pinchflat.Lifecycle.Notifications.CommandRunner,
// the real apprise runner. It implements AppriseRunner (app.go).
type NotificationsCommandRunner struct {
	App *App
}

var _ AppriseRunner = (*NotificationsCommandRunner)(nil)

// RunFull returns the full output and error, for testing.
// Returns (string, error) to match Elixir's {:ok, output} | {:error, message}.
func (r *NotificationsCommandRunner) RunFull(ctx context.Context, endpoints []string, commandOpts KW) (string, error) {
	// Check for empty endpoints
	if len(endpoints) == 0 {
		return "", errors.New("no_servers")
	}

	// Build default options and parse
	defaultOpts := KW{Flag("verbose")}
	parsedOpts := CliUtilsParseOptions(append(defaultOpts, commandOpts...))

	// Append endpoints to options
	allArgs := append(parsedOpts, endpoints...)

	// Run the command
	output, status, err := r.App.CliUtilsWrapCmd(ctx, r.App.Config.AppriseExecutable, allArgs, nil, nil)
	if err != nil {
		return "", err
	}

	output = strings.TrimSpace(output)

	if status == 0 {
		return output, nil
	}
	return "", errors.New(output)
}

// run/2 — :ok | {:error, message}. Elixir accepts one endpoint string or a
// list; callers pass a one-element slice for a single string.
func (r *NotificationsCommandRunner) Run(ctx context.Context, endpoints []string, commandOpts KW) error {
	_, err := r.RunFull(ctx, endpoints, commandOpts)
	return err
}

// version/0
func (r *NotificationsCommandRunner) Version(ctx context.Context) (string, error) {
	output, status, err := r.App.CliUtilsWrapCmd(ctx, r.App.Config.AppriseExecutable, []string{"--version"}, nil, nil)
	if err != nil {
		return "", err
	}

	if status != 0 {
		return "", errors.New(output)
	}

	// Parse the output: split by newline, take first line, remove "Apprise", trim
	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return "", nil
	}

	version := strings.TrimSpace(lines[0])
	version = strings.ReplaceAll(version, "Apprise", "")
	version = strings.TrimSpace(version)

	return version, nil
}
