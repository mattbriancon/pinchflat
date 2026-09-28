package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/fsutil"
)

// UserScriptsCommandRunner is Pinchflat.Lifecycle.UserScripts.CommandRunner,
// the real user script runner. It implements UserScriptRunner (app.go).
type UserScriptsCommandRunner struct {
	App *App
}

var _ UserScriptRunner = (*UserScriptsCommandRunner)(nil)

// @event_types
var userScriptsEventTypes = []string{
	"app_init",
	"media_pre_download",
	"media_downloaded",
	"media_deleted",
}

// UserScriptResult is the success value of run/2: {:ok, :no_executable}
// (NoExecutable true) or {:ok, output, exit_code}.
type UserScriptResult struct {
	NoExecutable bool
	Output       string
	ExitCode     int
}

// RunWithResult is run/2 with its full result. An invalid event type is an
// error (Elixir raises ArgumentError "Invalid event type: :foo").
func (r *UserScriptsCommandRunner) RunWithResult(ctx context.Context, eventType string, encodableData any) (*UserScriptResult, error) {
	// Check if event type is valid
	validEventType := false
	for _, et := range userScriptsEventTypes {
		if et == eventType {
			validEventType = true
			break
		}
	}
	if !validEventType {
		return nil, errors.New(fmt.Sprintf("Invalid event type: :%s", eventType))
	}

	// Get the backend executable
	executablePath, err := r.backendExecutable(ctx)
	if err != nil {
		return nil, err
	}

	if executablePath == "" {
		// No executable
		return &UserScriptResult{NoExecutable: true}, nil
	}

	// Encode the data as JSON
	encodedData, err := db.EncodeJSON(encodableData)
	if err != nil {
		return nil, err
	}

	// Run the command
	output, exitCode, err := fsutil.RunCommand(ctx, r.App.Config.TmpfileDirectory, executablePath, []string{eventType, encodedData}, fsutil.RunOptions{LoggingArgOverride: "[suppressed]"})
	if err != nil {
		return nil, err
	}

	return &UserScriptResult{
		NoExecutable: false,
		Output:       output,
		ExitCode:     exitCode,
	}, nil
}

// backendExecutable returns the path to the lifecycle script, or "" if not present or empty.
func (r *UserScriptsCommandRunner) backendExecutable(ctx context.Context) (string, error) {
	baseDir := r.App.Config.ExtrasDirectory
	lifecycleFilepath := filepath.Join(baseDir, "user-scripts", "lifecycle")

	exists := fsutil.ExistsAndNonEmpty(lifecycleFilepath)
	if !exists {
		slog.Info("User scripts lifecyle file either not present or is empty. Skipping.")
		return "", nil
	}

	return lifecycleFilepath, nil
}

// Run satisfies UserScriptRunner by discarding the result, except that a
// non-zero exit code (the only case Pinchflat.Lifecycle.UserScripts.CommandRunner
// callers care about via the plain error-returning interface) becomes a
// *CommandError, since the interface can't otherwise carry the exit code.
func (r *UserScriptsCommandRunner) Run(ctx context.Context, eventType string, data any) error {
	result, err := r.RunWithResult(ctx, eventType, data)
	if err != nil {
		return err
	}
	if !result.NoExecutable && result.ExitCode != 0 {
		return &fsutil.CommandError{Output: result.Output, Status: result.ExitCode}
	}
	return nil
}
