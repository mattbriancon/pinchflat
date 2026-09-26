package core

import "context"

// PreJobStartupTasks runs before the job runner starts. The GenServer
// wrapper is gone: main calls PreJobStartupTasksInit directly (and skips
// it in the test env, like the Elixir init(%{env: :test}) clause).

// init/1
func (a *App) PreJobStartupTasksInit(ctx context.Context) error {
	panic("unported: Pinchflat.Boot.PreJobStartupTasks.init/1")
}
