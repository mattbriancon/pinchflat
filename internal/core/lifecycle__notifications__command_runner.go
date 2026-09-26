package core

import "context"

// NotificationsCommandRunner is Pinchflat.Lifecycle.Notifications.CommandRunner,
// the real apprise runner. It implements AppriseRunner (app.go).
type NotificationsCommandRunner struct {
	App *App
}

var _ AppriseRunner = (*NotificationsCommandRunner)(nil)

// run/2 — :ok | {:error, message}. Elixir accepts one endpoint string or a
// list; callers pass a one-element slice for a single string.
func (r *NotificationsCommandRunner) Run(ctx context.Context, endpoints []string, commandOpts KW) error {
	panic("unported: Pinchflat.Lifecycle.Notifications.CommandRunner.run/2")
}

// version/0
func (r *NotificationsCommandRunner) Version(ctx context.Context) (string, error) {
	panic("unported: Pinchflat.Lifecycle.Notifications.CommandRunner.version/0")
}
