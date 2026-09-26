package core

import "context"

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
	panic("unported: Pinchflat.Lifecycle.UserScripts.CommandRunner.run/2")
}

// Run satisfies UserScriptRunner by discarding the result.
func (r *UserScriptsCommandRunner) Run(ctx context.Context, eventType string, data any) error {
	_, err := r.RunWithResult(ctx, eventType, data)
	return err
}
