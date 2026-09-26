package core

import (
	"context"
)

// PreJobStartupTasks runs startup tasks before the job runner has initialized.

type PreJobStartupTasks struct {
	// GenServer state will be added during implementation
}

// PreJobStartupTasksStartLink/1
func PreJobStartupTasksStartLink(ctx context.Context, opts KW) (any, error) {
	panic("unported: Pinchflat.Boot.PreJobStartupTasks.start_link/1")
}

// PreJobStartupTasksInit/1
func (p *PreJobStartupTasks) PreJobStartupTasksInit(ctx context.Context, state map[string]any) (map[string]any, error) {
	panic("unported: Pinchflat.Boot.PreJobStartupTasks.init/1")
}
