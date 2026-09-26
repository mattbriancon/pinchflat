package core

import "context"

// PostBootStartupTasks runs once the app has booted. main calls
// PostBootStartupTasksInit directly (skipped in the test env).

// init/1
func (a *App) PostBootStartupTasksInit(ctx context.Context) error {
	panic("unported: Pinchflat.Boot.PostBootStartupTasks.init/1")
}
