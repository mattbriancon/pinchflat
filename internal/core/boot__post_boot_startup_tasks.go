package core

import "context"

// PostBootStartupTasks runs once the app has booted. main calls
// PostBootStartupTasksInit directly (skipped in the test env).

// init/1
func (a *App) PostBootStartupTasksInit(ctx context.Context) error {
	return updateYtDlp(ctx, a)
}

// updateYtDlp/0
func updateYtDlp(ctx context.Context, a *App) error {
	_, err := a.UpdateWorkerKickoff(ctx)
	return err
}
